package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func completedSocketUpstream() Upstream {
	return upstreamFunc(func(_ context.Context, _ Request, consume func(Response) error) error {
		return consume(Response{StatusCode: 200, ContentType: "text/event-stream",
			Body: strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp-1\",\"output\":[]}}\n\n")})
	})
}

func dialPrefixedSocket(t *testing.T, options Options, header http.Header) (*websocket.Conn, context.Context) {
	t.Helper()
	options.Port = 0
	options.Models = []string{"gpt-6.1-sol"}
	server := New(options)
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := server.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	endpoint := strings.Replace(server.Status().Address, "http:", "ws:", 1) + "/codex/v1/responses"
	socket, _, err := websocket.Dial(ctx, endpoint, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { socket.CloseNow() })
	return socket, ctx
}

// TestCodexWebSocketRecordsRuntimeLabel 断言协议入口的 WebSocket 同样按来源标识细化
// Codex 运行形态；access.client 的原始值保持不变，逐帧复核仍走同一判定。
func TestCodexWebSocketRecordsRuntimeLabel(t *testing.T) {
	for _, tc := range []struct{ name, originator, want string }{
		{"cli", "codex_cli_rs", "codex-cli"},
		{"desktop", "codex_desktop", "codex-desktop"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			observer := &testObserver{}
			socket, ctx := dialPrefixedSocket(t, Options{
				Upstream:         completedSocketUpstream(),
				Observer:         observer,
				Authorize:        func(context.Context, string) (string, bool) { return "", false },
				PrefixAccess:     func(context.Context, string) (string, bool) { return "codex", true },
				WebSocketAllowed: func(context.Context) (bool, error) { return true, nil },
			}, http.Header{"Authorization": {"Bearer " + strings.Repeat("k", 1400)}, "Originator": {tc.originator}})

			event := sendSocket(t, ctx, socket, `{"type":"response.create","model":"gpt-6.1-sol","input":[]}`)
			if string(event["type"]) != `"response.completed"` {
				t.Fatalf("socket turn failed: %s", event["type"])
			}
			deadline := time.Now().Add(5 * time.Second)
			for len(observer.events) == 0 && time.Now().Before(deadline) {
				time.Sleep(10 * time.Millisecond)
			}
			if len(observer.events) == 0 {
				t.Fatal("socket turn was not recorded")
			}
			if got := observer.events[0].Client; got != tc.want {
				t.Fatalf("请求归属 = %q，期望 %q", got, tc.want)
			}
		})
	}
}

// TestCodexWebSocketSurvivesPrefixedAuthorization 断言 Codex 固定前缀入口不写接入 Key，
// 连接内逐帧复核必须沿用同一判定，否则首帧就被判 unauthorized，客户端反复重连。
func TestCodexWebSocketSurvivesPrefixedAuthorization(t *testing.T) {
	var authorized atomic.Int32
	var prefixAllowed atomic.Bool
	prefixAllowed.Store(true)
	socket, ctx := dialPrefixedSocket(t, Options{
		Upstream: completedSocketUpstream(),
		Authorize: func(context.Context, string) (string, bool) {
			authorized.Add(1)
			return "", false
		},
		PrefixAccess: func(_ context.Context, prefix string) (string, bool) {
			return "codex", prefix == "/codex" && prefixAllowed.Load()
		},
		WebSocketAllowed: func(context.Context) (bool, error) { return true, nil },
	}, http.Header{"Authorization": {"Bearer " + strings.Repeat("j", 1400)}})

	event := sendSocket(t, ctx, socket, `{"type":"response.create","model":"gpt-6.1-sol","input":[]}`)
	if string(event["type"]) != `"response.completed"` {
		t.Fatalf("prefixed socket rejected on first frame: %s", event["type"])
	}
	// 官方登录令牌超过接入 Key 上限，任何阶段都不能进入接入 Key 比较。
	if authorized.Load() != 0 {
		t.Fatalf("official token reached access key comparison %d time(s)", authorized.Load())
	}

	// 停用该固定入口后，同一连接不能再创建响应。
	prefixAllowed.Store(false)
	event = sendSocket(t, ctx, socket, `{"type":"response.create","model":"gpt-6.1-sol","input":[]}`)
	var failure struct {
		Status int `json:"status"`
		Error  struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	raw, _ := json.Marshal(event)
	if json.Unmarshal(raw, &failure) != nil || failure.Status != 401 || failure.Error.Code != "unauthorized" {
		t.Fatalf("disabled prefix entry kept the session: %s", raw)
	}
}

// 带接入 Key 的连接仍按 Key 复核，停用后下一帧被拒。
func TestCodexWebSocketRevokesDisabledAccessKey(t *testing.T) {
	key := "opk_" + strings.Repeat("a", 64)
	var keyEnabled atomic.Bool
	keyEnabled.Store(true)
	socket, ctx := dialPrefixedSocket(t, Options{
		Upstream: completedSocketUpstream(),
		Authorize: func(_ context.Context, value string) (string, bool) {
			return "codex", value == key && keyEnabled.Load()
		},
		WebSocketAllowed: func(context.Context) (bool, error) { return true, nil },
	}, http.Header{"X-Api-Key": {key}})

	if event := sendSocket(t, ctx, socket, `{"type":"response.create","model":"gpt-6.1-sol","input":[]}`); string(event["type"]) != `"response.completed"` {
		t.Fatalf("keyed socket rejected: %s", event["type"])
	}
	keyEnabled.Store(false)
	event := sendSocket(t, ctx, socket, `{"type":"response.create","model":"gpt-6.1-sol","input":[]}`)
	if !strings.Contains(string(event["error"]), "unauthorized") || string(event["status"]) != "401" {
		t.Fatalf("disabled key kept the session: %v", event)
	}
}
