package router_test

import (
	"fmt"
	"github.com/gowww/router"
	"net/http"
)

func Example() {
	rt := router.New()

	// File server
	rt.Get("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Static route
	rt.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello")
	}))

	// Path parameter
	rt.Get("/users/:id", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Get user %s", router.Parameter(r, "id"))
	}))

	// Path parameter + Trailing slash for wildcard
	rt.Post("/users/:id/files/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Post file %s to user %s", router.Parameter(r, "*"), router.Parameter(r, "id"))
	}))

	// Custom "not found"
	rt.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	http.ListenAndServe(":8080", rt)
}
