package router

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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

func TestFindChild(t *testing.T) {
	fmt.Println(rt)
	for reqPath, wantedHandler := range reqTests {
		n, _ := rt.trees["GET"].findChild(reqPath, nil)
		if n == nil {
			if wantedHandler != nil {
				t.Errorf("%q not found", reqPath)
			}
		} else if reflect.ValueOf(n.handler) != reflect.ValueOf(wantedHandler) {
			t.Errorf("%q handler: want %v, got %v", reqPath, wantedHandler, n.handler)
		}
	}
}

func TestServeHTTP(t *testing.T) {
	for reqPath, wantedHandler := range reqTests {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", reqPath, nil)
		rt.ServeHTTP(w, r)
		if w.Code == http.StatusOK && wantedHandler == nil {
			t.Errorf("%q must not be found", reqPath)
		} else if w.Code == http.StatusNotFound && wantedHandler != nil {
			t.Errorf("%q not found", reqPath)
		}
	}
}

func TestParameters(t *testing.T) {
	id := "12"
	office := "london"
	rt := New()
	rt.Get("/users/:id/contact/:office", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := Parameter(r, "id"); v != id {
			t.Errorf("id: want %q, got %q", id, v)
		}
		if v := Parameter(r, "office"); v != office {
			t.Errorf("id: want %q, got %q", office, v)
		}
		if v := Parameter(r, "unknown"); v != "" {
			t.Errorf("id: want %q, got %q", "", v)
		}
	}))
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/users/"+id+"/contact/"+office, nil)
	rt.ServeHTTP(w, r)
}

func TestNoParameters(t *testing.T) {
	rt := New()
	rt.Get("/user", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := Parameter(r, "unknown"); v != "" {
			t.Errorf("id: want %q, got %q", "", v)
		}
	}))
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/user", nil)
	rt.ServeHTTP(w, r)
}

func TestRedirectTrailingSlash(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/user/", nil)
	rt.ServeHTTP(w, r)
	if w.Code != http.StatusMovedPermanently {
		t.Fail()
	}
}

func TestMissingFirstSlash(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fail()
		}
	}()
	rt := New()
	rt.Get("user", nil)
}

func TestDuplicatedRoutes(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fail()
		}
	}()
	rt := New()
	rt.Get("/:id", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	rt.Get("/:name", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	fmt.Println(rt)
}

func TestNoNotFoundHandler(t *testing.T) {
	status := http.StatusForbidden
	body := "foobar"
	rt := New()
	rt.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	rt.ServeHTTP(w, r)
	if w.Code != status {
		t.Errorf("status: want %d, got %d", status, w.Code)
	}
	if b, _ := ioutil.ReadAll(w.Body); string(b) != body {
		t.Errorf("status: want %q, got %q", body, string(b))
	}
}

func BenchmarkRouter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for reqPath := range reqTests {
			rt.trees["GET"].findChild(reqPath, nil)
		}
	}
}
