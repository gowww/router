package router

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

var (
	rt = New()

	rtTests = map[string]http.Handler{
		"/":                           http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		"/user":                       http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		"/:page":                      http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		"/user/files/":                http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		"/users/:id/car":              http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		"/user/:item":                 http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		"/user/contact/home":          http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		"/user/contact/home/dubai":    http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		"/user/contact/office/london": http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}

	reqTests = map[string]http.Handler{
		"/":                           rtTests["/"],
		"/user":                       rtTests["/user"],
		"/about":                      rtTests["/:page"],
		"/user/files/foo":             rtTests["/user/files/"],
		"/user/files/foo/bar":         rtTests["/user/files/"],
		"/user/files":                 rtTests["/user/:item"],
		"/user/contact/office/london": rtTests["/user/contact/office/london"],
		"/user/contact/office/paris":  nil,
		"/page/notfound":              nil,
	}
)

func init() {
	for path, handler := range rtTests {
		rt.Get(path, handler)
		rt.Post(path, handler)
		rt.Put(path, handler)
		rt.Patch(path, handler)
		rt.Delete(path, handler)
	}
}

func TestHandle(t *testing.T) {
	fmt.Println(rt)
	for reqPath, wantedHandler := range reqTests {
		n, _ := rt.trees["GET"].findChild(true, reqPath, nil)
		if n == nil {
			if wantedHandler != nil {
				t.Errorf("%q not found", reqPath)
			}
		} else if reflect.ValueOf(n.handler) != reflect.ValueOf(wantedHandler) {
			t.Errorf("%q handler: want %v, got %v", reqPath, wantedHandler, n.handler)
		}
	}
}

func BenchmarkRouter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for reqPath := range reqTests {
			rt.trees["GET"].findChild(true, reqPath, nil)
		}
	}
}
