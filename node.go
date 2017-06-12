package router

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type node struct {
	s        string
	params   map[string]uint16 // Parameter's names from the parent node to this one, and their path part index (between "/").
	children []*node
	handler  http.Handler
	isRoot   bool // Need to know if node is root to not use it as wildcard.
}

func (n *node) string(prefix string) (s string) {
	s = fmt.Sprintf("%s%s", prefix, n.s)
	if n.params != nil {
		s = fmt.Sprintf("%s  %v", s, n.params)
	}
	if n.handler != nil {
		s = fmt.Sprintf("%s  %v", s, n.handler)
	}
	if n.isRoot {
		s += "  root"
	}
	s += "\n"
	for _, child := range n.children {
		s += child.string(prefix + strings.Repeat("∙", len(n.s)) + " ")
	}
	return
}

func (n *node) isWildcard() bool {
	return isWildcard(n.s)
}

func (n *node) countChildren() (i int) {
	for _, n := range n.children {
		i++
		i += n.countChildren()
	}
	return
}

// makeChild adds a node to the tree.
func (n *node) makeChild(path string, params map[string]uint16, handler http.Handler, isRoot bool) {
NodesLoop:
	for _, child := range n.children {
		minlen := len(child.s)
		if len(path) < minlen {
			minlen = len(path)
		}
		for i := 0; i < minlen; i++ {
			if child.s[i] == path[i] {
				continue
			}
			if i == 0 { // No match from the first byte: see next same-level node.
				continue NodesLoop
			}
			// Difference in the middle of a node: split current node to make subnode and transfer handler to it.
			*child = node{
				s: child.s[:i],
				children: []*node{
					{s: child.s[i:], params: child.params, children: child.children, handler: child.handler},
					{s: path[i:], params: params, handler: handler},
				},
				isRoot: child.isRoot,
			}
			child.sortChildren()
			return
		}
		if len(path) < len(child.s) { // s fully matched first part of n.s: split node.
			*child = node{
				s:      child.s[:len(path)],
				params: params,
				children: []*node{
					{s: child.s[len(path):], params: child.params, children: child.children, handler: child.handler},
				},
				handler: handler,
				isRoot:  child.isRoot,
			}
		} else if len(path) > len(child.s) { // n.s fully matched first part of s: see subnodes for the rest.
			child.makeChild(path[len(child.s):], params, handler, false)
		} else { // s == n.s and no rest: node has no handler or route is duplicated.
			if handler == nil { // No handler provided (must be a non-ending path parameter): don't overwrite.
				return
			}
			if child.handler != nil { // Handler provided but n.handler already set: route is duplicated.
				panic(fmt.Errorf("router: two or more routes have same path"))
			}
			child.params = params
			child.handler = handler
		}
		child.sortChildren()
		return
	}
	n.children = append(n.children, &node{s: path, params: params, handler: handler, isRoot: isRoot}) // Not a single byte match on same-level nodes: append a new one.
	n.sortChildren()
}

func (n *node) findChild(path string) *node {
	for _, n = range n.children {
		if n.s == ":" { // Handle parameter node.
			paramEnd := strings.IndexByte(path, '/')
			if paramEnd == -1 { // Path ends with the parameter.
				return n
			}
			return n.findChild(path[paramEnd:])
		}
		if !strings.HasPrefix(path, n.s) { // Node doesn't match beginning of path.
			continue
		}
		if len(path) == len(n.s) { // Node matched until the end of path.
			return n
		}
		child := n.findChild(path[len(n.s):])
		if child == nil || child.handler == nil {
			if !n.isRoot && n.isWildcard() { // If node is a wildcard, don't use it when it's root.
				return n
			}
			continue // No match from children and current node is not a wildcard, maybe there is a parameter in next same-level node.
		}
		return child
	}
	return nil
}

// sortChildren puts children with most subnodes on top and plain strings before parameter.
func (n *node) sortChildren() {
	sort.Slice(n.children, func(i, j int) bool {
		if n.children[i].s == ":" {
			return false
		}
		if n.children[j].s == ":" {
			return true
		}
		return n.children[i].countChildren() > n.children[j].countChildren()
	})
}
