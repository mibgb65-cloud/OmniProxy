package httpapi

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"omniproxy/internal/platform/profile"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	prof, err := profile.New(profile.Test)
	if err != nil {
		t.Fatalf("profile.New error: %v", err)
	}
	return New(prof)
}

// freePort asks the OS for a currently unused TCP port.
func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatalf("release port: %v", err)
	}
	return port
}

func TestHealthz(t *testing.T) {
	srv := httptest.NewServer(newTestServer(t).Router())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q, want application/json", ct)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("status field = %v, want ok", body["status"])
	}
	if body["profile"] != string(profile.Test) {
		t.Errorf("profile field = %v, want %q", body["profile"], profile.Test)
	}
}

func TestUnknownRouteReturns404(t *testing.T) {
	srv := httptest.NewServer(newTestServer(t).Router())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/nope")
	if err != nil {
		t.Fatalf("GET /nope: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}

func TestStartBindsToLoopbackControlPort(t *testing.T) {
	prof, err := profile.New(profile.Test)
	if err != nil {
		t.Fatal(err)
	}
	prof.ControlPort = freePort(t)

	s := New(prof)
	if err := s.Start(); err != nil {
		t.Fatalf("Start error: %v", err)
	}
	if err := s.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown error: %v", err)
	}
}
