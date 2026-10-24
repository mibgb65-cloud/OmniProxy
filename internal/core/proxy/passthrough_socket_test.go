package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// TestPassthroughSocketRelaysFrames 断言 WebSocket 真的拨到上游并原样中继帧，
// 且客户端头随握手带到上游，握手自身管理的头（Origin）不透传。
func TestPassthroughSocketRelaysFrames(t *testing.T) {
	var gotOriginator, gotSession, gotOrigin string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer pool-token" {
			t.Errorf("upstream Authorization = %q", got)
		}
		gotOriginator, gotSession, gotOrigin = r.Header.Get("originator"), r.Header.Get("session_id"), r.Header.Get("Origin")
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
			if err := socket.Write(r.Context(), kind, data); err != nil {
				return
			}
		}
	}))
	defer upstream.Close()

	session := &stubSession{attempts: []PassthroughAttempt{{
		AccountID: "acct_a",
		BaseURL:   upstream.URL,
		Header:    http.Header{"Authorization": {"Bearer pool-token"}},
	}}}
	server := New(Options{Port: 0, Passthrough: stubBinding{session}, Upstream: unusedUpstream(t)})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = server.Stop(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dialURL := "ws" + strings.TrimPrefix(server.Status().Address, "http") + "/codex/v1/responses"
	header := http.Header{}
	header.Set("originator", "codex_cli_rs")
	header.Set("session_id", "sess-socket")
	socket, _, err := websocket.Dial(ctx, dialURL, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		t.Fatalf("dial gateway websocket: %v", err)
	}
	defer socket.CloseNow()

	if err := socket.Write(ctx, websocket.MessageText, []byte(`{"type":"response.create"}`)); err != nil {
		t.Fatal(err)
	}
	kind, data, err := socket.Read(ctx)
	if err != nil {
		t.Fatalf("read relayed frame: %v", err)
	}
	if kind != websocket.MessageText || string(data) != `{"type":"response.create"}` {
		t.Fatalf("relayed frame = %v %q", kind, data)
	}
	if gotOriginator != "codex_cli_rs" || gotSession != "sess-socket" {
		t.Fatalf("客户端头未透传到上游握手: originator=%q session_id=%q", gotOriginator, gotSession)
	}
	if gotOrigin != "" {
		t.Fatalf("Origin 不应透传给上游: %q", gotOrigin)
	}
	if len(session.done) != 1 || session.done[0] != http.StatusSwitchingProtocols {
		t.Fatalf("socket attempt outcomes = %v", session.done)
	}
}

// TestPassthroughSocketSwitchesAccountOnHandshakeRejection 断言握手被明确拒绝时换号重拨。
func TestPassthroughSocketSwitchesAccountOnHandshakeRejection(t *testing.T) {
	rejecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer rejecting.Close()
	echo := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			if err := socket.Write(r.Context(), kind, data); err != nil {
				return
			}
		}
	}))
	defer echo.Close()

	session := &stubSession{retryable: map[int]bool{429: true}, attempts: []PassthroughAttempt{
		{AccountID: "acct_a", BaseURL: rejecting.URL, Header: http.Header{"Authorization": {"Bearer a"}}},
		{AccountID: "acct_b", BaseURL: echo.URL, Header: http.Header{"Authorization": {"Bearer b"}}},
	}}
	server := New(Options{Port: 0, Passthrough: stubBinding{session}, Upstream: unusedUpstream(t)})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = server.Stop(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dialURL := "ws" + strings.TrimPrefix(server.Status().Address, "http") + "/codex/v1/responses"
	socket, _, err := websocket.Dial(ctx, dialURL, nil)
	if err != nil {
		t.Fatalf("dial gateway websocket: %v", err)
	}
	defer socket.CloseNow()
	if err := socket.Write(ctx, websocket.MessageText, []byte("ping")); err != nil {
		t.Fatal(err)
	}
	if _, data, err := socket.Read(ctx); err != nil || string(data) != "ping" {
		t.Fatalf("换号后未连到第二个账号: %v %q", err, data)
	}
	if len(session.done) != 2 || session.done[0] != http.StatusTooManyRequests || session.done[1] != http.StatusSwitchingProtocols {
		t.Fatalf("socket attempt outcomes = %v", session.done)
	}
}
