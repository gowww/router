package router_test

import (
	"fmt"
	"net/http"

	"github.com/gowww/router"
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

	http.ListenAndServe(":8080", rt)
}
