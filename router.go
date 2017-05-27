// Package router provides a fast HTTP router.
package router

import (
	"fmt"
	"net/http"
	"strings"
)

// The Router is the main structure of this package.
// TODO: Have custom "not found" handler.
type Router struct {
	trees map[string]*[]*node // trees is a map of methods with their path nodes.
}

type node struct {
	s        string
	params   []string // Parameter's names from the parent node to this one.
	children []*node
	handler  http.Handler
}

func (n *node) isWildcard() bool {
	return n.s[len(n.s)-1] == '/' && len(n.children) == 0
}

// New returns a fresh rounting unit.
func New() *Router {
	return &Router{trees: make(map[string]*[]*node)}
}

// Handle adds a route with method, path and handler.
// TODO: Specify in doc that "/path" and "/path/" (trailing slash) are 2 different routes.
func (rt *Router) Handle(method, path string, h http.Handler) {
	if len(path) == 0 || path[0] != '/' {
		panic(fmt.Errorf("router: path %q must begin with %q", path, "/"))
	}

	// Extract parameters from path.
	var params []string
	var paramStart, paramEnd int
	for {
		paramStart = strings.IndexByte(path[paramEnd:], ':')
		if paramStart == -1 { // No more parameters: make node.
			break
		}
		paramStart += paramEnd
		paramStart++ // Position on parameter name instead of ":".
		paramEnd = strings.IndexByte(path[paramStart:], '/')
		if paramEnd == -1 { // Parameter is at the end the path.
			params = append(params, path[paramStart:len(path)])
			path = path[:paramStart]
			break
		}
		paramEnd += paramStart
		params = append(params, path[paramStart:paramEnd])
		path = path[:paramStart] + path[paramEnd:]
		paramEnd -= paramEnd - paramStart
	}

	// Get (or set) tree for method.
	tree := rt.trees[method]
	if tree == nil {
		n := make([]*node, 0)
		rt.trees[method] = &n
		tree = &n
	}

	// Put parameters in their own node.
	for _, pos := range paramsPos(path) {
		makeNode(tree, path[:pos], nil, nil) // Make node for part before parameter.
		if pos+1 < len(path) {               // Parameter doesn't close the path: make node (whithout handler) for it.
			makeNode(tree, path[:pos+1], nil, nil)
		}
	}
	makeNode(tree, path, params, h)

	// TODO: Sort trees (most subnodes on top and plain strings before parameters).
}

// paramsPos returns a slice of ':' positions in s.
func paramsPos(s string) (pos []int) {
	for i := 0; i < len(s); i++ {
		p := strings.IndexByte(s[i:], ':')
		if p == -1 {
			break
		}
		pos = append(pos, p+i)
		i = p + i
	}
	return
}

// makeNode adds a node to the tree.
func makeNode(nodes *[]*node, path string, params []string, handler http.Handler) {
NodesLoop:
	for _, n := range *nodes {
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
				children: []*node{
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
				children: []*node{
					{s: n.s[len(path):], params: n.params, children: n.children, handler: n.handler},
				},
				handler: handler,
			}
		} else if len(path) > len(n.s) { // n.s fully matched first part of s: see subnodes for the rest.
			makeNode(&n.children, path[len(n.s):], params, handler)
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
	*nodes = append(*nodes, &node{s: path, params: params, handler: handler}) // Not a single byte match on same-level nodes: append a new one.
}

func (rt Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// TODO: Clean path.
	trees := rt.trees[r.Method]
	if trees != nil {
		var params []string
		n := findNode(*trees, r.URL.Path, &params)
		// TODO: Store parameter values in request.
		if n != nil && n.handler != nil {
			n.handler.ServeHTTP(w, r)
			return
		}
	}
	http.NotFound(w, r)
}

func findNode(nodes []*node, path string, params *[]string) *node {
	for _, n := range nodes {
		if n.s == ":" { // Handle parameter node.
			paramEnd := strings.IndexByte(path, '/')
			if paramEnd == -1 { // Path ends with the parameter.
				if n.handler != nil { // Performance: append parameter only if the node has a handler (otherwise useless).
					*params = append(*params, path)
				}
				return n
			}
			*params = append(*params, path[:paramEnd])
			return findNode(n.children, path[paramEnd:], params)
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
		return findNode(n.children, path[len(n.s):], params)
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
