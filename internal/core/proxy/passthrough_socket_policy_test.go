package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// TestPassthroughSocketStopsFramesAfterPolicyDisabled 断言关闭 WebSocket 开关后，
// 已建立的透明转发连接不再把新的客户端帧转给上游，而不是继续静默转发。
func TestPassthroughSocketStopsFramesAfterPolicyDisabled(t *testing.T) {
	var frames atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		socket, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer socket.CloseNow()
		for {
			kind, data, err := socket.Read(r.Context())
			if err != nil {
				return
			}
			frames.Add(1)
			if err := socket.Write(r.Context(), kind, data); err != nil {
				return
			}
		}
	}))
	defer upstream.Close()

	var allowed atomic.Bool
	allowed.Store(true)
	session := &stubSession{attempts: []PassthroughAttempt{{
		AccountID: "acct_a",
		BaseURL:   upstream.URL,
		Header:    http.Header{"Authorization": {"Bearer pool-token"}},
	}}}
	server := New(Options{
		Port: 0, Passthrough: stubBinding{session}, Upstream: unusedUpstream(t),
		WebSocketAllowed: func(context.Context) (bool, error) { return allowed.Load(), nil },
	})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = server.Stop(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	dialURL := "ws" + strings.TrimPrefix(server.Status().Address, "http") + "/codex/v1/responses"
	socket, _, err := websocket.Dial(ctx, dialURL, nil)
	if err != nil {
		t.Fatalf("dial gateway websocket: %v", err)
	}
	defer socket.CloseNow()

	if err = socket.Write(ctx, websocket.MessageText, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if _, data, err := socket.Read(ctx); err != nil || string(data) != "first" {
		t.Fatalf("开启状态下的中继失败: %v %q", err, data)
	}

	allowed.Store(false)
	// 服务端可能在读取前就关闭连接，因此这里只要求「第二帧不得到达上游」，
	// 并把连接最终不可用作为开关生效的证据。
	_ = socket.Write(ctx, websocket.MessageText, []byte("second"))
	for {
		if _, _, readErr := socket.Read(ctx); readErr != nil {
			break
		}
	}
	if frames.Load() != 1 {
		t.Fatalf("关闭开关后仍向上游转发了客户端帧: %d", frames.Load())
	}
}
