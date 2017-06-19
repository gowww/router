package router

import (
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
)

type node struct {
	s        string
	params   map[string]uint16 // Parameter's names from the parent node to this one, and their path part index (between "/").
	re       *regexp.Regexp
	children []*node
	handler  http.Handler
	isRoot   bool // Need to know if node is root to not use it as wildcard.
}

func (n *node) string(prefix string) (s string) {
	s = fmt.Sprintf("%s%s", prefix, n.s)
	if n.params != nil {
		s = fmt.Sprintf("%s  %v", s, n.params)
	}
	if n.re != nil {
		s = fmt.Sprintf("%s  %v", s, n.re)
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

// isParameter tells if the node is a parameter.
func (n *node) isParameter() bool {
	return n.s == ":"
}

// isParameter tells if the node is a wildcard.
func (n *node) isWildcard() bool {
	return isWildcard(n.s)
}

// countChildren returns the number of children + grandchildren in node.
func (n *node) countChildren() (i int) {
	for _, n := range n.children {
		i++
		i += n.countChildren()
	}
	return
}

// makeChild adds a node to the tree.
func (n *node) makeChild(path string, params map[string]uint16, re *regexp.Regexp, handler http.Handler, isRoot bool) {
	defer n.sortChildren()
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
					{s: child.s[i:], params: child.params, re: child.re, children: child.children, handler: child.handler},
					{s: path[i:], params: params, re: re, handler: handler},
				},
			}
			// BUG(arthurwhite): Child.s == "/" but child.isRoot == false on first level after this split.
			return
		}
		if len(path) < len(child.s) { // s fully matched first part of n.s: split node.
			*child = node{
				s:      child.s[:len(path)],
				params: params,
				re:     re,
				children: []*node{
					{s: child.s[len(path):], params: child.params, re: child.re, children: child.children, handler: child.handler},
				},
				handler: handler,
				isRoot:  isRoot,
			}
		} else if len(path) > len(child.s) { // n.s fully matched first part of s: see subnodes for the rest.
			child.makeChild(path[len(child.s):], params, re, handler, false)
		} else { // s == n.s and no rest: node has no handler or route is duplicated.
			if handler == nil { // No handler provided (must be a non-ending path parameter): don't overwrite.
				return
			}
			if child.handler != nil { // Handler provided but n.handler already set: route is duplicated.
				if re == nil && child.re == nil || re != nil && child.re != nil && re.String() == child.re.String() {
					panic("router: two or more routes have same path")
				}
				continue NodesLoop // It's a parameter with a regular expression: check next child for "same path" error. Otherwise, node will be appended.
			}
			child.params = params
			child.re = re
			child.handler = handler
			child.isRoot = isRoot
		}
		return
	}
	n.children = append(n.children, &node{s: path, params: params, re: re, handler: handler, isRoot: isRoot}) // Not a single byte match on same-level nodes: append a new one.
}

// findChild returns the deepest node matching path.
func (n *node) findChild(path string) *node {
	for _, n = range n.children {
		if n.isParameter() {
			paramEnd := strings.IndexByte(path, '/')
			if paramEnd == -1 { // Path ends with the parameter.
				if n.re != nil && !n.re.MatchString(path) {
					continue
				}
				return n
			}
			if n.re != nil && !n.re.MatchString(path[:paramEnd]) {
				continue
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

// sortChildren puts children with most subnodes on top, plain strings before parameters, and parameters with regular expressions before the parameter without.
func (n *node) sortChildren() {
	sort.Slice(n.children, func(i, j int) bool {
		a := n.children[i]
		b := n.children[j]
		return a.isParameter() && b.isParameter() && a.re != nil ||
			!a.isParameter() && b.isParameter() ||
			a.countChildren() > b.countChildren()
	})
}
