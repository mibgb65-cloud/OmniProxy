package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithLoopbackHostAllowsLocalClients(t *testing.T) {
	hosts := []string{
		"127.0.0.1:3000",
		"127.0.0.1",
		"localhost:3000",
		"LOCALHOST:3890",
		"[::1]:3890",
		"127.0.0.2:3000",
	}
	for _, host := range hosts {
		req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
		req.Host = host
		res := httptest.NewRecorder()
		reached := false
		withLoopbackHost(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			reached = true
		})).ServeHTTP(res, req)
		if !reached || res.Code != http.StatusOK {
			t.Fatalf("expected host %q to be allowed, got status %d reached=%v", host, res.Code, reached)
		}
	}
}

func TestWithLoopbackHostRejectsReboundHosts(t *testing.T) {
	hosts := []string{
		"evil.com",
		"evil.com:3000",
		"127.0.0.1.evil.com:3000",
		"10.0.0.5:3000",
		"",
	}
	for _, host := range hosts {
		req := httptest.NewRequest(http.MethodGet, "/v1/messages", nil)
		req.Host = host
		res := httptest.NewRecorder()
		withLoopbackHost(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			t.Fatalf("handler must not run for host %q", host)
		})).ServeHTTP(res, req)
		if res.Code != http.StatusForbidden {
			t.Fatalf("expected host %q to be rejected, got status %d", host, res.Code)
		}
	}
}
