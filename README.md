# [![gowww](https://avatars.githubusercontent.com/u/18078923?s=20)](https://github.com/gowww) router [![GoDoc](https://godoc.org/github.com/gowww/router?status.svg)](https://godoc.org/github.com/gowww/router) [![Build](https://travis-ci.org/gowww/router.svg?branch=master)](https://travis-ci.org/gowww/router) [![Coverage](https://coveralls.io/repos/github/gowww/router/badge.svg?branch=master)](https://coveralls.io/github/gowww/router?branch=master) [![Go Report](https://goreportcard.com/badge/github.com/gowww/router)](https://goreportcard.com/report/github.com/gowww/router)

Package [router](https://godoc.org/github.com/gowww/router) provides a fast HTTP router.

## Example

```Go
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
```
