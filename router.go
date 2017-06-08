// Package router provides a lightning fast HTTP router.
package router

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

type contextKey int

// Context keys
const (
	contextKeyParameters contextKey = iota
)

// The Router is the main structure of this package.
type Router struct {
	NotFoundHandler http.Handler
	trees           map[string]*nodes // trees is a map of methods with their path nodes.
}

// New returns a fresh rounting unit.
func New() *Router {
	return &Router{
		trees: make(map[string]*nodes),
	}
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

// Handle adds a route with method, path and handler.
// TODO: Specify in doc that "/path" and "/path/" (trailing slash) are 2 different routes.
func (rt *Router) Handle(method, path string, handler http.Handler) {
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
		if paramEnd == -1 { // Parameter is at the end of the path.
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
		tree.makeChild(path[:pos], nil, nil, true) // Make node for part before parameter.
		if pos+1 < len(path) {                     // Parameter doesn't close the path: make node (whithout handler) for it.
			tree.makeChild(path[:pos+1], nil, nil, true)
		}
	}
	tree.makeChild(path, params, handler, true)
	tree.sort() // Could be done during makeChild for best performance.
}

// Get makes a route for GET method.
func (rt *Router) Get(path string, handler http.Handler) {
	rt.Handle(http.MethodGet, path, handler)
}

// Post makes a route for POST method.
func (rt *Router) Post(path string, handler http.Handler) {
	rt.Handle(http.MethodPost, path, handler)
}

// Put makes a route for PUT method.
func (rt *Router) Put(path string, handler http.Handler) {
	rt.Handle(http.MethodPut, path, handler)
}

// Patch makes a route for PATCH method.
func (rt *Router) Patch(path string, handler http.Handler) {
	rt.Handle(http.MethodPatch, path, handler)
}

// Delete makes a route for DELETE method.
func (rt *Router) Delete(path string, handler http.Handler) {
	rt.Handle(http.MethodDelete, path, handler)
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Remove trailing slash.
	if len(r.URL.Path) > 1 && r.URL.Path[len(r.URL.Path)-1] == '/' {
		r.URL.Path = r.URL.Path[:len(r.URL.Path)-1]
		http.Redirect(w, r, r.URL.String(), http.StatusMovedPermanently)
		return
	}

	// TODO: Handle OPTIONS request.

	if trees := rt.trees[r.Method]; trees != nil {
		n, params := trees.findChild(r.URL.Path, nil)
		if n != nil && n.handler != nil {
			// Store parameters in request's context.
			if len(n.params) > 0 {
				pm := make(map[string]string)
				for i, param := range n.params {
					pm[param] = params[i]
				}
				if len(params) > len(n.params) { // One params over n.params: it's the wildcard.
					pm["*"] = params[len(params)-1]
				}
				r = r.WithContext(context.WithValue(r.Context(), contextKeyParameters, pm))
			}
			n.handler.ServeHTTP(w, r)
			return
		}
	}

	if rt.NotFoundHandler != nil {
		rt.NotFoundHandler.ServeHTTP(w, r)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

// Parameter returns the value of path parameter.
func Parameter(r *http.Request, key string) string {
	params, ok := r.Context().Value(contextKeyParameters).(map[string]string)
	if !ok {
		return ""
	}
	param, _ := params[key]
	return param
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
