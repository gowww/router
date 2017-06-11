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
	contextKeyParamsIdx contextKey = iota
	contextKeyParams
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
func (rt *Router) Handle(method, path string, handler http.Handler) {
	if len(path) == 0 || path[0] != '/' {
		panic(fmt.Errorf("router: path %q must begin with %q", path, "/"))
	}

	// Get (or set) tree for method.
	nn := rt.trees[method]
	if nn == nil {
		var n nodes
		rt.trees[method] = &n
		nn = &n
	}

	// Put parameters in their own node.
	parts := splitPath(path)
	var s string
	var params map[string]int
	for i, part := range parts {
		s += "/"
		if len(part) > 0 && part[0] == ':' { // It's a parameter.
			if len(part) < 2 {
				panic(fmt.Errorf("router: path %q has anonymous field", path))
			}
			nn.makeChild(s, params, nil, (i == 0 && s == "/")) // Make child without ":"
			if params == nil {
				params = make(map[string]int)
			}
			params[part[1:]] = i   // Store parameter name with part index.
			s += ":"               // Only keep "/:".
			if i == len(parts)-1 { // Parameter is the last part: make it with handler.
				nn.makeChild(s, params, handler, false)
			} else {
				nn.makeChild(s, params, nil, false)
			}
		} else {
			s += part
			if i == len(parts)-1 { // Last part: make it with handler.
				if s != "/" && isWildcard(s) {
					if params == nil {
						params = make(map[string]int)
					}
					params["*"] = i
				}
				nn.makeChild(s, params, handler, (i == 0 && s == "/"))
			}
		}
	}
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
		n := trees.findChild(r.URL.Path)
		if n != nil && n.handler != nil {
			// Store parameters in request's context.
			if n.params != nil {
				r = r.WithContext(context.WithValue(r.Context(), contextKeyParamsIdx, n.params))
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
// Result is empty if parameter doesn't exist.
func Parameter(r *http.Request, key string) string {
	params, ok := r.Context().Value(contextKeyParams).(map[string]string)
	if ok { // Parameters already parsed.
		return params[key]
	}
	paramsIdx, ok := r.Context().Value(contextKeyParamsIdx).(map[string]int)
	if !ok {
		return ""
	}
	params = make(map[string]string, len(paramsIdx))
	parts := splitPath(r.URL.Path)
	for name, idx := range paramsIdx {
		switch name {
		case "*":
			for idx < len(parts) {
				params[name] += parts[idx]
				idx++
			}
		default:
			params[name] = parts[idx]
		}
	}
	*r = *r.WithContext(context.WithValue(r.Context(), contextKeyParams, params))
	return params[key]
}

func isWildcard(s string) bool {
	return s[len(s)-1] == '/'
}

func splitPath(path string) []string {
	return strings.Split(path, "/")[1:]
}
