// Package router provides a fast HTTP router.
package router

import (
	"fmt"
	"net/http"
	"strings"
)

// The Router is the main structure of this package.
type Router struct {
	NotFoundHandler http.Handler
	trees           map[string]*nodes // trees is a map of methods with their path nodes.
}

// New returns a fresh rounting unit.
func New() *Router {
	return &Router{
		NotFoundHandler: http.NotFoundHandler(),
		trees:           make(map[string]*nodes),
	}
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
			params = append(params, path[paramStart:])
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
		n := make(nodes, 0)
		rt.trees[method] = &n
		tree = &n
	}

	// Put parameters in their own node.
	for _, pos := range paramsPos(path) {
		tree.makeChild(path[:pos], nil, nil) // Make node for part before parameter.
		if pos+1 < len(path) {               // Parameter doesn't close the path: make node (whithout handler) for it.
			tree.makeChild(path[:pos+1], nil, nil)
		}
	}
	tree.makeChild(path, params, h)

	// TODO: Sort trees (most subnodes on top and plain strings before parameters).
}

func (rt Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Clean path
	if len(r.URL.Path) > 1 && r.URL.Path[len(r.URL.Path)-1] == '/' {
		http.Redirect(w, r, r.URL.Path[:len(r.URL.Path)-1], http.StatusPermanentRedirect)
	}

	trees := rt.trees[r.Method]
	if trees != nil {
		var params []string
		n := trees.findChild(r.URL.Path, &params)
		// TODO: Store parameter values in request.
		if n != nil && n.handler != nil {
			n.handler.ServeHTTP(w, r)
			return
		}
	}

	if rt.NotFoundHandler == nil {
		http.NotFound(w, r)
		return
	}
	rt.NotFoundHandler.ServeHTTP(w, r)
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
