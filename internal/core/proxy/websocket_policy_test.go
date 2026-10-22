package proxy

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/coder/websocket"
)

func TestWebSocketPolicyChangesWithoutRestartAndKeepsHTTP(t *testing.T) {
	var enabled atomic.Bool
	enabled.Store(true)
	var calls atomic.Int32
	key := "opk_" + strings.Repeat("a", 64)
	server := New(Options{
		Port: 0, Models: []string{"deepseek-flash"},
		Authorize:        func(_ context.Context, value string) (string, bool) { return "", value == key },
		WebSocketAllowed: func(context.Context) (bool, error) { return enabled.Load(), nil },
		Upstream: upstreamFunc(func(_ context.Context, _ Request, consume func(Response) error) error {
			calls.Add(1)
			return consume(Response{StatusCode: 200, ContentType: "text/event-stream", Body: strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"test\",\"output\":[]}}\n\n")})
		}),
	})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop(context.Background())
	endpoint := server.Status().Address + "/agent/codex/" + key + "/v1/responses"
	socketURL := strings.Replace(endpoint, "http:", "ws:", 1)
	ctx := context.Background()
	socket, _, err := websocket.Dial(ctx, socketURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer socket.CloseNow()
	// WebSocket 输入按协议使用数组，确保允许请求确实调用上游。
	message := []byte(`{"type":"response.create","model":"deepseek-flash","input":[{"role":"user","content":"hello"}]}`)
	if err := socket.Write(ctx, websocket.MessageText, message); err != nil {
		t.Fatal(err)
	}
	if _, body, err := socket.Read(ctx); err != nil || !strings.Contains(string(body), "response.completed") {
		t.Fatal("enabled socket failed", err)
	}
	enabled.Store(false)
	_, response, err := websocket.Dial(ctx, socketURL, nil)
	if err == nil || response == nil || response.StatusCode != 403 {
		t.Fatal("disabled handshake accepted", err)
	}
	if err := socket.Write(ctx, websocket.MessageText, message); err != nil {
		t.Fatal(err)
	}
	if _, body, err := socket.Read(ctx); err != nil || !strings.Contains(string(body), "websocket_disabled") {
		t.Fatal("existing connection bypassed policy", err)
	}
	if calls.Load() != 1 {
		t.Fatal("disabled socket reached upstream")
	}
	response, err = http.Post(endpoint, "application/json", strings.NewReader(`{"model":"deepseek-flash","input":"hello","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		t.Fatal("HTTP disabled with WebSocket")
	}
	enabled.Store(true)
	restored, _, err := websocket.Dial(ctx, socketURL, nil)
	if err != nil {
		t.Fatal("cannot re-enable without restart", err)
	}
	restored.CloseNow()
}

func TestWebSocketPolicyReadFailureFailsClosedAfterAuthorization(t *testing.T) {
	server := New(Options{Port: 0, WebSocketAllowed: func(context.Context) (bool, error) { return false, errors.New("private database error") },
		Authorize: func(_ context.Context, key string) (string, bool) { return "", key == "test-key" },
		Upstream:  upstreamFunc(func(context.Context, Request, func(Response) error) error { t.Fatal("unexpected upstream"); return nil }),
	})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Stop(context.Background())
	endpoint := strings.Replace(server.Status().Address, "http:", "ws:", 1) + "/v1/responses"
	for _, test := range []struct {
		key    string
		status int
	}{{"", 401}, {"test-key", 503}} {
		_, response, err := websocket.Dial(context.Background(), endpoint, &websocket.DialOptions{HTTPHeader: http.Header{"X-Api-Key": {test.key}}})
		if err == nil || response == nil || response.StatusCode != test.status {
			t.Fatal("wrong failure status", err)
		}
		body, err := io.ReadAll(response.Body)
		if err != nil || strings.Contains(string(body), "private") {
			t.Fatal("unsafe failure", err)
		}
	}
}
