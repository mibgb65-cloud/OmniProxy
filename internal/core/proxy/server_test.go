package proxy

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type upstreamFunc func(context.Context, Request, func(Response) error) error

func (f upstreamFunc) Forward(ctx context.Context, req Request, consume func(Response) error) error {
	return f(ctx, req, consume)
}

const validBody = `{"model":"deepseek-flash","messages":[{"role":"user","content":"hello"}]}`

func testServer(t *testing.T, upstream Upstream, timeout time.Duration) *Server {
	t.Helper()
	server := New(Options{Upstream: upstream, Models: []string{"deepseek-flash"}, RequestTimeout: timeout})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if _, err := server.Stop(ctx); err != nil {
			t.Error(err)
		}
	})
	return server
}

func successfulUpstream(ctx context.Context, req Request, consume func(Response) error) error {
	return consume(Response{StatusCode: 200, ContentType: "application/json", Body: strings.NewReader(`{"choices":[{"message":{"content":"OK"}}]}`)})
}

func request(t *testing.T, server *Server, body string) (int, string) {
	t.Helper()
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Post(server.Status().Address+"/v1/chat/completions", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(data)
}

func TestLifecycleAndConcurrentStart(t *testing.T) {
	server := testServer(t, upstreamFunc(successfulUpstream), 0)
	before := server.Status()
	if before.State != "running" || before.Port == 0 || before.StartedAt == "" {
		t.Fatalf("unexpected status: %+v", before)
	}
	var group sync.WaitGroup
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			status, err := server.Start()
			if err != nil || status.Address != before.Address {
				t.Errorf("non-idempotent start: %+v %v", status, err)
			}
		}()
	}
	group.Wait()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	status, err := server.Stop(ctx)
	if err != nil || status.State != "stopped" || status.Address != "" {
		t.Fatalf("stop: %+v %v", status, err)
	}
	if connection, err := net.DialTimeout("tcp", strings.TrimPrefix(before.Address, "http://"), 100*time.Millisecond); err == nil {
		connection.Close()
		t.Fatal("listener remains available")
	}
	if _, err := server.Start(); err != nil {
		t.Fatal("restart:", err)
	}
}

func TestPortInUse(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	server := New(Options{Port: listener.Addr().(*net.TCPAddr).Port, Upstream: upstreamFunc(successfulUpstream)})
	status, err := server.Start()
	if err == nil || status.State != "error" || status.ErrorCode != "port_unavailable" {
		t.Fatalf("unexpected status: %+v %v", status, err)
	}
}

func TestStopCancelsActiveUpstream(t *testing.T) {
	entered, cancelled := make(chan struct{}), make(chan struct{})
	server := testServer(t, upstreamFunc(func(ctx context.Context, req Request, consume func(Response) error) error {
		close(entered)
		<-ctx.Done()
		close(cancelled)
		return ctx.Err()
	}), 0)
	finished := make(chan struct{})
	go func() { defer close(finished); request(t, server, validBody) }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("upstream did not start")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := server.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-cancelled:
	case <-time.After(time.Second):
		t.Fatal("upstream not cancelled")
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("local request did not finish")
	}
}

func TestRequestTimeout(t *testing.T) {
	server := testServer(t, upstreamFunc(func(ctx context.Context, req Request, consume func(Response) error) error {
		<-ctx.Done()
		return ctx.Err()
	}), 30*time.Millisecond)
	status, body := request(t, server, validBody)
	if status != 504 || !strings.Contains(body, "request_timeout") {
		t.Fatalf("timeout result: %d %s", status, body)
	}
}
