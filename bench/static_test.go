package main

import "testing"

var (
	staticRoutes = []string{
		"/",
		"/usage",
		"/user",
		"/us",
		"/user/contact/office/london",
		"/user/contact/office/losangeles",
		"/user/contact/home",
		"/user/contact/home/dubai",
		"/user/contacted",
	}

	staticRequests = []string{
		"/us",
		"/user/contact/home",
		"/user/contacted",
		"/user/contact/home/dubai",
		"/usage",
		"/user",
		"/user/contact/office/losangeles",
		"/",
		"/page/notfound",
		"/user/contact/office/london",
	}
)

func BenchmarkStaticChi(b *testing.B) {
	bench(b, staticRequests, setChi(staticRoutes))
}

func BenchmarkStaticFastRoute(b *testing.B) {
	bench(b, staticRequests, setFastRoute(staticRoutes))
}

func BenchmarkStaticGowwwRouter(b *testing.B) {
	bench(b, staticRequests, setGowwwRouter(staticRoutes))
}

func BenchmarkStaticHTTPRouter(b *testing.B) {
	bench(b, staticRequests, setHTTPRouter(staticRoutes))
}
