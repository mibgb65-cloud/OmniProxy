package proxy

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"
)

func freePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

func TestSetPortRepointsRunningServer(t *testing.T) {
	server := testServer(t, upstreamFunc(successfulUpstream), 0)
	target := freePort(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	status, err := server.SetPort(ctx, target)
	if err != nil {
		t.Fatalf("SetPort error: %v", err)
	}
	if status.State != "running" || status.Port != target {
		t.Fatalf("status after repoint: %+v", status)
	}
	if want := "http://127.0.0.1:" + strconv.Itoa(target); status.Address != want {
		t.Fatalf("address = %q, want %q", status.Address, want)
	}
	if code, _ := request(t, server, validBody); code != 200 {
		t.Fatalf("request on new port = %d, want 200", code)
	}
}

func TestSetPortOnStoppedServerKeepsItStopped(t *testing.T) {
	server := New(Options{Upstream: upstreamFunc(successfulUpstream)})
	target := freePort(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	status, err := server.SetPort(ctx, target)
	if err != nil {
		t.Fatalf("SetPort error: %v", err)
	}
	if status.State != "stopped" || status.Port != target {
		t.Fatalf("status = %+v, want stopped on %d", status, target)
	}
	if _, err := server.Start(); err != nil {
		t.Fatalf("start after SetPort: %v", err)
	}
	defer func() {
		if _, err := server.Stop(ctx); err != nil {
			t.Error(err)
		}
	}()
	if server.Status().Port != target {
		t.Fatalf("listening port = %d, want %d", server.Status().Port, target)
	}
}

func TestSetPortRejectsInvalidPort(t *testing.T) {
	server := New(Options{Upstream: upstreamFunc(successfulUpstream)})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := server.SetPort(ctx, -1); err == nil {
		t.Fatal("negative port accepted")
	}
	if _, err := server.SetPort(ctx, 70000); err == nil {
		t.Fatal("out of range port accepted")
	}
}

func TestSetPortReportsUnavailablePort(t *testing.T) {
	server := testServer(t, upstreamFunc(successfulUpstream), 0)
	occupied, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	status, err := server.SetPort(ctx, occupied.Addr().(*net.TCPAddr).Port)
	if err == nil || status.State != "error" || status.ErrorCode != "port_unavailable" {
		t.Fatalf("status = %+v, err = %v", status, err)
	}
}
