package proxy

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestStreamingFlushBeforeUpstreamCompletes(t *testing.T) {
	finished := make(chan struct{})
	server := testServer(t, upstreamFunc(func(ctx context.Context, req Request, consume func(Response) error) error {
		reader, writer := io.Pipe()
		defer reader.Close()
		go func() {
			defer writer.Close()
			if _, err := io.WriteString(writer, "data: {\"choices\":[{\"delta\":{\"content\":\"first\"}}]}\n\n"); err != nil {
				return
			}
			select {
			case <-finished:
				io.WriteString(writer, "data: [DONE]\n\n")
			case <-ctx.Done():
			}
		}()
		return consume(Response{StatusCode: 200, ContentType: "text/event-stream", Body: reader})
	}), 2*time.Second)
	response, err := http.Post(server.Status().Address+"/v1/chat/completions", "application/json", strings.NewReader(strings.TrimSuffix(validBody, "}")+`,"stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	first, err := bufio.NewReader(response.Body).ReadString('\n')
	if err != nil || !strings.Contains(first, "first") {
		t.Fatalf("first event not delivered: %q %v", first, err)
	}
	close(finished)
}

type waitingBody struct {
	entered, release chan struct{}
	once             sync.Once
	body             io.Reader
}

func (b *waitingBody) Read(p []byte) (int, error) {
	b.once.Do(func() { close(b.entered) })
	<-b.release
	return b.body.Read(p)
}

func TestConcurrencyLimitIncludesReadingBodies(t *testing.T) {
	server := New(Options{MaxConcurrent: 1, Models: []string{"deepseek-flash"}, Upstream: upstreamFunc(successfulUpstream)})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop(context.Background())
	body := &waitingBody{entered: make(chan struct{}), release: make(chan struct{}), body: strings.NewReader(validBody)}
	req, err := http.NewRequest(http.MethodPost, server.Status().Address+"/v1/chat/completions", body)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	finished := make(chan struct{})
	go func() { defer close(finished); server.ServeHTTP(httptest.NewRecorder(), req) }()
	defer func() {
		close(body.release)
		select {
		case <-finished:
		case <-time.After(time.Second):
			t.Error("body reader did not finish")
		}
	}()
	select {
	case <-body.entered:
	case <-time.After(time.Second):
		t.Fatal("body reader did not start")
	}
	status, result := request(t, server, validBody)
	if status != 429 || !strings.Contains(result, "gateway_busy") {
		t.Fatalf("reading body bypassed concurrency limit: %d %s", status, result)
	}
}

func TestIncompleteStreamIsNotCleanSuccess(t *testing.T) {
	server := testServer(t, upstreamFunc(func(ctx context.Context, req Request, consume func(Response) error) error {
		return consume(Response{StatusCode: 200, ContentType: "text/event-stream", Body: strings.NewReader("data: {\"choices\":[]}\n\n")})
	}), 0)
	response, err := http.Post(server.Status().Address+"/v1/chat/completions", "application/json", strings.NewReader(strings.TrimSuffix(validBody, "}")+`,"stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if _, err := io.ReadAll(response.Body); err == nil {
		t.Fatal("truncated SSE response completed without a transport error")
	}
}

func TestConcurrencyLimit(t *testing.T) {
	entered := make(chan struct{})
	server := New(Options{MaxConcurrent: 1, Models: []string{"deepseek-flash"}, Upstream: upstreamFunc(func(ctx context.Context, req Request, consume func(Response) error) error {
		close(entered)
		<-ctx.Done()
		return ctx.Err()
	})})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	finished := make(chan struct{})
	go func() { defer close(finished); request(t, server, validBody) }()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("request did not start")
	}
	status, body := request(t, server, validBody)
	if status != 429 || !strings.Contains(body, "gateway_busy") {
		t.Fatalf("limit not enforced: %d %s", status, body)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if _, err := server.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("pending request did not stop")
	}
}
