package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestProxyHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	if !proxyHealthy(srv.URL) {
		t.Error("a server returning 200 on /health should be healthy")
	}
	if proxyHealthy("http://127.0.0.1:1") {
		t.Error("a dead address should be unhealthy")
	}
}
