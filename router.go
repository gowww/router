// Package router provides a fast HTTP router.
package router

import (
	"fmt"
	"net/http"
	"strings"
)

// The Router is the main structure of this package.
type Router struct {
	trees map[string]*[]*node
}

type node struct {
	s        string
	params   []string
	children []*node
	handler  http.Handler
}

// New returns a fresh rounting unit.
func New() *Router {
	return &Router{trees: make(map[string]*[]*node)}
}

// Handle adds a route with method, path and handler.
func (rt *Router) Handle(method, path string, h http.Handler) {
	if len(method) == 0 {
		panic(fmt.Errorf("router: route %q method is empty", path))
	}
	if len(path) == 0 || path[0] != '/' {
		panic(fmt.Errorf("router: route path %q must begin with %q", path, "/"))
	}
	tree := rt.trees[method]
	if tree == nil {
		n := make([]*node, 0)
		rt.trees[method] = &n
		tree = &n
	}
	makeNode(tree, path, h)
}

// makeNode adds a node to the tree.
func makeNode(nodes *[]*node, s string, h http.Handler) {
LoopNodes:
	for _, n := range *nodes {
		minlen := len(n.s)
		if len(s) < minlen {
			minlen = len(s)
		}
		for i := 0; i < minlen; i++ {
			if n.s[i] != s[i] {
				if i == 0 {
					continue LoopNodes // No match from the first byte: see next same-level node.
				}
				// Difference in the middle of a node: split current node to make subnode and transfer handler to it.
				*n = node{
					s: n.s[:i],
					children: []*node{
						{s: n.s[i:], children: n.children, handler: n.handler},
						{s: s[i:], handler: h},
					},
				}
				return
			}
		}
		if len(s) < len(n.s) { // s fully matched first part of n.s: split node.
			*n = node{
				s: n.s[:len(s)],
				children: []*node{
					{s: n.s[len(s):], children: n.children, handler: n.handler},
				},
				handler: h,
			}
		} else if len(s) > len(n.s) { // n.s fully matched first part of s: see subnodes for the rest.
			makeNode(&n.children, s[len(n.s):], h)
		} else { // s == n.s and no rest: node hasn't handler or route is duplicated.
			if n.handler == nil {
				n.handler = h
				return
			}
			panic(fmt.Errorf("router: route %q is duplicated", s))
		}
		return
	}
	*nodes = append(*nodes, &node{s: s, handler: h}) // Not a single byte match on same-level nodes: append a new one.
}

func (rt Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	trees := rt.trees[r.Method]
	if trees != nil {
		h := findNode(*trees, r.URL.Path)
		if h != nil {
			h.ServeHTTP(w, r)
			return
		}
	}
	http.NotFound(w, r)
}

func findNode(nodes []*node, s string) http.Handler {
	for _, n := range nodes {
		if strings.HasPrefix(s, n.s) {
			s = s[len(n.s):]
			if len(s) == 0 {
				return n.handler
			}
			return findNode(n.children, s)
		}
	}
	return nil
}

func (rt *Router) String() (s string) {
	for method, nodes := range rt.trees {
		s += method + "\n"
		for _, n := range *nodes {
			s += n.string(1)
		}
	}
	return
}

func (n *node) string(level int) (s string) {
	s += fmt.Sprintf("%s%q  %v  %v\n", strings.Repeat("\t", level), n.s, n.params, n.handler)
	for _, n := range n.children {
		s += n.string(level + 1)
	}
	return
}
