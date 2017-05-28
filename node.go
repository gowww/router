package router

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type node struct {
	s        string
	params   []string // Parameter's names from the parent node to this one.
	children nodes
	handler  http.Handler
}

func (n *node) string(level int) (s string) {
	s += fmt.Sprintf("%s%s  %v  %v\n", strings.Repeat("\t", level), n.s, n.params, n.handler)
	for _, n := range n.children {
		s += n.string(level + 1)
	}
	return
}

func (n *node) isWildcard() bool {
	return n.s[len(n.s)-1] == '/' && len(n.children) == 0
}

func (n *node) countChildren() (i int) {
	for _, n := range n.children {
		i++
		i += n.countChildren()
	}
	return
}

type nodes []*node

// makeChild adds a node to the tree.
func (nn *nodes) makeChild(path string, params []string, handler http.Handler) {
NodesLoop:
	for _, n := range *nn {
		minlen := len(n.s)
		if len(path) < minlen {
			minlen = len(path)
		}
		for i := 0; i < minlen; i++ {
			if n.s[i] == path[i] {
				continue
			}
			if i == 0 { // No match from the first byte: see next same-level node.
				continue NodesLoop
			}
			// Difference in the middle of a node: split current node to make subnode and transfer handler to it.
			*n = node{
				s: n.s[:i],
				children: nodes{
					{s: n.s[i:], params: n.params, children: n.children, handler: n.handler},
					{s: path[i:], params: params, handler: handler},
				},
			}
			return
		}
		if len(path) < len(n.s) { // s fully matched first part of n.s: split node.
			*n = node{
				s:      n.s[:len(path)],
				params: params,
				children: nodes{
					{s: n.s[len(path):], params: n.params, children: n.children, handler: n.handler},
				},
				handler: handler,
			}
		} else if len(path) > len(n.s) { // n.s fully matched first part of s: see subnodes for the rest.
			n.children.makeChild(path[len(n.s):], params, handler)
		} else { // s == n.s and no rest: node has no handler or route is duplicated.
			if handler == nil { // No handler provided (must be a non-ending path parameter): don't overwrite.
				return
			}
			if n.handler != nil { // Handler provided but n.handler already set: route is duplicated.
				panic(fmt.Errorf("router: two or more routes have same path"))
			}
			n.params = params
			n.handler = handler
		}
		return
	}
	*nn = append(*nn, &node{s: path, params: params, handler: handler}) // Not a single byte match on same-level nodes: append a new one.
}

func (nn *nodes) findChild(path string, params *[]string) *node {
	for _, n := range *nn {
		if n.s == ":" { // Handle parameter node.
			paramEnd := strings.IndexByte(path, '/')
			if paramEnd == -1 { // Path ends with the parameter.
				if n.handler != nil { // Performance: append parameter only if the node has a handler (otherwise useless).
					*params = append(*params, path)
				}
				return n
			}
			*params = append(*params, path[:paramEnd])
			return n.children.findChild(path[paramEnd:], params)
		}
		if !strings.HasPrefix(path, n.s) { // Node doesn't match beginning of path.
			continue
		}
		if n.isWildcard() {
			*params = append(*params, path[len(n.s):])
			return n
		}
		if len(path) == len(n.s) { // Node matched until the end of path.
			return n
		}
		child := n.children.findChild(path[len(n.s):], params)
		if child == nil {
			continue // No match from children, maybe there is a parameter in next same-level node.
		}
		return child
	}
	return nil
}

// sort puts nodes with most subnodes on top and plain strings before parameter and wildcard.
func (nn *nodes) sort() {
	sort.Slice(*nn, func(i, j int) bool {
		if (*nn)[i].s == ":" || (*nn)[i].isWildcard() {
			return false
		}
		if (*nn)[j].s == ":" || (*nn)[j].isWildcard() {
			return true
		}
		return (*nn)[i].countChildren() > (*nn)[j].countChildren()
	})
	for _, n := range *nn {
		n.children.sort()
	}
}
