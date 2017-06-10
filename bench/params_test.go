package main

import "testing"

var (
	paramsRoutes = []string{
		"/item/:page",
		"/user/:item",
		"/users/:id/carriage",
		"/users/:id/car",
	}

	paramsRequests = []string{
		"/users/42/car",
		"/page/notfound",
		"/item/about",
		"/user/files",
		"/users/42/carriage",
	}
)

func BenchmarkParamsChi(b *testing.B) {
	bench(b, paramsRequests, setChi(paramsRoutes))
}

func BenchmarkParamsFastRoute(b *testing.B) {
	bench(b, paramsRequests, setFastRoute(paramsRoutes))
}

func BenchmarkParamsGowwwRouter(b *testing.B) {
	bench(b, paramsRequests, setGowwwRouter(paramsRoutes))
}

func BenchmarkParamsHTTPRouter(b *testing.B) {
	bench(b, paramsRequests, setHTTPRouter(paramsRoutes))
}
