// Package router provides a lightning fast HTTP router.
package router

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

// The Router is the main structure of this package.
type Router struct {
	NotFoundHandler http.Handler
	trees           map[string]*node // trees is a map of methods with their path nodes.
}

// New returns a fresh rounting unit.
func New() *Router {
	return &Router{
		trees: make(map[string]*node),
	}
}

func (rt *Router) String() (s string) {
	for method, node := range rt.trees {
		s += method + "\n"
		for _, n := range node.children {
			s += n.string(strings.Repeat(" ", len(method)+1))
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
	n := rt.trees[method]
	if n == nil {
		n = new(node)
		rt.trees[method] = n
	}

	// Put parameters in their own node.
	parts := strings.Split(path[1:], "/")
	var s string
	var params []string // Parameter names by path part index ("" for a part without name).
	for i, part := range parts {
		s += "/"
		if len(part) > 0 && part[0] == ':' { // It's a parameter.
			n.makeChild(s, params, nil, nil, (i == 0 && s == "/")) // Make child without ":".
			// Separate name from regular expression.
			name, res, hasRe := strings.Cut(part[1:], ":")
			var re *regexp.Regexp
			if !hasRe { // No regular expression.
				if name == "" {
					panic(fmt.Errorf("router: path %q has anonymous parameter", path))
				}
			} else { // Parameter comes with regular expression.
				if res == "" {
					panic(fmt.Errorf("router: path %q has empty regular expression", path))
				}
				re = regexp.MustCompile(res)
			}
			params = append(params, make([]string, i-len(params))...) // Pad previous parts without parameter.
			params = append(params, name)
			s += ":"               // Only keep colon to represent parameter in tree.
			if i == len(parts)-1 { // Parameter is the last part: make it with handler.
				n.makeChild(s, params, re, handler, false)
			} else {
				n.makeChild(s, params, re, nil, false)
			}
		} else {
			s += part
			if i == len(parts)-1 { // Last part: make it with handler.
				if s != "/" && isWildcard(s) {
					params = append(params, make([]string, i-len(params))...) // Pad previous parts without parameter.
					params = append(params, "*")
				}
				n.makeChild(s, params, nil, handler, (i == 0 && s == "/"))
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

	if n := rt.trees[r.Method]; n != nil {
		n = n.findChild(r.URL.Path)
		if n != nil && n.handler != nil {
			path := r.URL.Path[1:] // Store parameters in request.
			for _, name := range n.params {
				part, rest, _ := strings.Cut(path, "/")
				if name == "*" { // Wildcard takes the rest of path.
					part = path
				}
				if name != "" {
					r.SetPathValue(name, part)
				}
				path = rest
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
// It's the same as the standard r.PathValue(key).
func Parameter(r *http.Request, key string) string {
	return r.PathValue(key)
}

// isWildcard tells if s ends with '/'.
func isWildcard(s string) bool {
	return s[len(s)-1] == '/'
}
