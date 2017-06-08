package router

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	// "github.com/DATA-DOG/fastroute"
	// "github.com/julienschmidt/httprouter"
)

type rtTest struct {
	path    string
	handler http.Handler
}

type reqTest struct {
	path   string
	rtTest *rtTest
}

var (
	rt = New()

	rtTests = []*rtTest{
		{path: "/", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})},
		{path: "/usage", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})},
		{path: "/user", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})},
		{path: "/us", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})},
		{path: "/:page", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})},
		{path: "/user/:item", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})},
		{path: "/user/files/", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})},
		{path: "/users/:id/carriage", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})},
		{path: "/users/:id/car", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})},
		{path: "/user/contact/office/london", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})},
		{path: "/user/contact/office/losangeles", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})},
		{path: "/user/contact/home", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})},
		{path: "/user/contact/home/dubai", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})},
		{path: "/user/contacted", handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})},
	}

	reqTests = []*reqTest{
		{path: "/", rtTest: findRtTest("/")},
		{path: "/user", rtTest: findRtTest("/user")},
		{path: "/about", rtTest: findRtTest("/:page")},
		{path: "/user/files/foo", rtTest: findRtTest("/user/files/")},
		{path: "/user/files/foo/bar", rtTest: findRtTest("/user/files/")},
		{path: "/user/files", rtTest: findRtTest("/user/:item")},
		{path: "/user/contact/office/london", rtTest: findRtTest("/user/contact/office/london")},
		{path: "/usage", rtTest: findRtTest("/usage")},
		{path: "/users/notfound", rtTest: nil},
		{path: "/user/contact/office/lo", rtTest: nil},
		{path: "/user/contact", rtTest: nil},
		{path: "/page/notfound", rtTest: nil},
	}
)

func findRtTest(path string) *rtTest {
	for _, t := range rtTests {
		if t.path == path {
			return t
		}
	}
	return nil
}

func init() {
	for _, rtt := range rtTests {
		rt.Get(rtt.path, rtt.handler)
		rt.Post(rtt.path, rtt.handler)
		rt.Put(rtt.path, rtt.handler)
		rt.Patch(rtt.path, rtt.handler)
		rt.Delete(rtt.path, rtt.handler)
	}
}

func TestFindChild(t *testing.T) {
	fmt.Println(rt)
	for _, reqt := range reqTests {
		n, _ := rt.trees[http.MethodGet].findChild(reqt.path, nil)
		if n == nil {
			if reqt.rtTest != nil {
				t.Errorf("%q not found", reqt.path)
			}
		} else if reqt.rtTest == nil {
			if n.handler != nil {
				t.Errorf("%q must not be found", reqt.path)
			}
		} else if reflect.ValueOf(n.handler) != reflect.ValueOf(reqt.rtTest.handler) {
			t.Errorf("%q handler: want %v, got %v", reqt.path, reqt.rtTest.handler, n.handler)
		}
	}
}

func TestServeHTTP(t *testing.T) {
	for _, reqt := range reqTests {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, reqt.path, nil)
		rt.ServeHTTP(w, r)
		if w.Code == http.StatusOK && reqt.rtTest == nil {
			t.Errorf("%q must not be found", reqt.path)
		} else if w.Code == http.StatusNotFound && reqt.rtTest != nil {
			t.Errorf("%q not found", reqt.path)
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
	r := httptest.NewRequest(http.MethodGet, "/users/"+id+"/contact/"+office, nil)
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
	r := httptest.NewRequest(http.MethodGet, "/user", nil)
	rt.ServeHTTP(w, r)
}

func TestRedirectTrailingSlash(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/user/", nil)
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

func TestNotFoundHandler(t *testing.T) {
	status := http.StatusForbidden
	body := "foobar"
	rt := New()
	rt.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	rt.ServeHTTP(w, r)
	if w.Code != status {
		t.Errorf("status: want %d, got %d", status, w.Code)
	}
	if b, _ := ioutil.ReadAll(w.Body); string(b) != body {
		t.Errorf("status: want %q, got %q", body, string(b))
	}
}

func BenchmarkFindRoute(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, reqt := range reqTests {
			rt.trees[http.MethodGet].findChild(reqt.path, nil)
		}
	}
}

func BenchmarkServeHTTP(b *testing.B) {
	w := httptest.NewRecorder()
	for i := 0; i < b.N; i++ {
		for _, reqt := range reqTests {
			rt.ServeHTTP(w, httptest.NewRequest(http.MethodGet, reqt.path, nil))
		}
	}
}
