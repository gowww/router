# [![gowww](https://avatars.githubusercontent.com/u/18078923?s=20)](https://github.com/gowww) router [![GoDoc](https://godoc.org/github.com/gowww/router?status.svg)](https://godoc.org/github.com/gowww/router) [![Build](https://travis-ci.org/gowww/router.svg?branch=master)](https://travis-ci.org/gowww/router) [![Coverage](https://coveralls.io/repos/github/gowww/router/badge.svg?branch=master)](https://coveralls.io/github/gowww/router?branch=master) [![Go Report](https://goreportcard.com/badge/github.com/gowww/router)](https://goreportcard.com/report/github.com/gowww/router)

Package [router](https://godoc.org/github.com/gowww/router) provides a lightning fast HTTP router.

- [Features](#features)
- [Installing](#installing)
- [Usage](#usage)
  - [Parameters](#parameters)
    - [Named](#named)
    - [Regular expressions](#regular-expressions)
    - [Wildcard](#wildcard)
  - [Static files](#static-files)
  - [Custom "not found" handler](#custom-not-found-handler)

## Features

  - Extreme performance: [sub-microsecond routing](https://gist.github.com/arthurwhite/bb632f6b104deb2a50ce476c25f7bec2) in most cases
  - Full compatibility with the [http.Handler](https://golang.org/pkg/net/http/#Handler) interface
  - Generic: no magic methods, bring your own handlers
  - Path parameters, regular expressions and wildcards
  - Smart prioritized routes
  - Zero memory allocation during serving (unless for parameters)
  - Respecting the principle of least surprise
  - Tested and used in production

## Installing

1. Get package:

	```Shell
	go get -u github.com/gowww/router
	````

2. Import it in your code:

	```Go
	import "github.com/gowww/router"
	```

## Usage

1. Make a new router:

	```Go
	rt := router.New()
	```

2. Make a route:

	```Go
	rt.Handle("GET", "/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello")
	}))
	```

   Remember that HTTP methods are case-sensitive and uppercase by convention ([RFC 7231 4.1](https://tools.ietf.org/html/rfc7231#section-4.1)).  
   So you can directly use shortcuts for standard HTTP methods: [Router.Get](https://godoc.org/github.com/gowww/router#Router.Get), [Router.Post](https://godoc.org/github.com/gowww/router#Router.Post), [Router.Put](https://godoc.org/github.com/gowww/router#Router.Put), [Router.Patch](https://godoc.org/github.com/gowww/router#Router.Patch) and [Router.Delete](https://godoc.org/github.com/gowww/router#Router.Delete).

3. Give the router to the server:

	```Go
	http.ListenAndServe(":8080", rt)
	```

### Parameters

#### Named

A named parameter begins with `:` and matches any value until the next `/` in path.

To retreive its value (stored in request's context), ask [Parameter](https://godoc.org/github.com/gowww/router#Router.Parameter).  
It will return the value as a string (empty if the parameter doesn't exist).

Example, with a parameter `:id`:

```Go
rt.Get("/users/:id", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	id := router.Parameter(r, "id")
	fmt.Fprintf(w, "Page of user #%s", id)
}))
```

<details>
  <summary>No surprise</summary>

  A parameter can be used on the same level as a static route, without conflict:

  ```Go
  rt.Get("/users/all", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  	fmt.Fprint(w, "All users page")
  }))
  
  rt.Get("/users/:id", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  	id := router.Parameter(r, "id")
  	fmt.Fprintf(w, "Page of user #%s", id)
  }))
  ```
</details>  

#### Regular expressions

If a parameter must match an exact pattern (number only, for example), you can also set a [regular expression](https://golang.org/pkg/regexp/syntax) constraint just after the parameter name and another `:`:

```Go
rt.Get(`/users/:id:^\d+$`, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	id := router.Parameter(r, "id")
	fmt.Fprintf(w, "Page of user #%s", id)
}))
```

If you don't need tho retreive the parameter value and just want to use a regular expression, you can omit the parameter name:

```Go
rt.Get(`/shows/::^prison-break-s06-.+`, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "Prison Break S06 — Coming soon…")
}))
```

<details>
  <summary>No surprise</summary>

  A parameter with a regular expression can be used on the same level as a simple parameter, without conflict:

  ```Go
  rt.Get(`/users/:id:^\d+$`, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  	id := router.Parameter(r, "id")
  	fmt.Fprintf(w, "Page of user #%s", id)
  }))
  
  rt.Get("/users/:name", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  	name := router.Parameter(r, "name")
  	fmt.Fprintf(w, "Page of %s", name)
  }))
  ```
</details>  

Don't forget that regular expressions can significantly reduce performance.

#### Wildcard

A trailing slash in a route path is significant.  
It behaves like a wildcard by matching the beginning of the request's path and keeping the rest as a parameter value, under `*`:

```Go
rt.Get("/files/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	filepath := router.Parameter(r, "*")
	fmt.Fprintf(w, "Get file %s", filepath)
}))
```

<details>
  <summary>No surprise</summary>

  Deeper route paths with the same prefix as the wildcard will take precedence, without conflict:

  ```Go
  // Will match:
  // 	/files/one
  // 	/files/two
  // 	...
  rt.Get("/files/:name", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {kv
  	name := router.Parameter(r, "name")
  	fmt.Fprintf(w, "Get root file #%s", name)
  }))
  
  // Will match:
  // 	/files/one/...
  // 	/files/two/...
  // 	...
  rt.Get("/files/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  	filepath := router.Parameter(r, "*")
  	fmt.Fprintf(w, "Get file %s", filepath)
  }))
  
  // Will match:
  // 	/files/movies/one
  // 	/files/movies/two
  // 	...
  rt.Get("/files/movies/:name", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  	name := router.Parameter(r, "name")
  	fmt.Fprintf(w, "Get movie #%s", name)
  }))
  ```
</details>  

Note that a trailing slash in a request path is always trimmed and the client redirected (better for SEO).  
For example, a request for `/files/` will be redirected to `/files` and will never match a `/files/` route.  
In other words, `/files` and `/files/` are two different routes.

### Static files

For serving static files, like for other routes, just bring your own handler.

Exemple, with the standard [net/http.FileServer](https://golang.org/pkg/net/http/#FileServer):

```Go
rt.Get("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
```

### Custom "not found" handler

When a request cannot be matched with a route, a 404 error with an empty body is send by default.

But you can set your own "not found" handler (and send an HTML page, for example):

```Go
rt.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
})
```

Note that is this case, it's up to you to set the correct status code (normally 404) for the response.
