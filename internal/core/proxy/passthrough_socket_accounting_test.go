package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// accountingSpy 同时记录父记录与尝试明细，供透明 WebSocket 逐轮记账断言。
type accountingSpy struct {
	mu       sync.Mutex
	finished []RequestEvent
	attempts []AttemptEvent
}

func (s *accountingSpy) Begin(context.Context, RequestEvent) error { return nil }

func (s *accountingSpy) Update(context.Context, RequestEvent) error { return nil }

func (s *accountingSpy) Finish(_ context.Context, event RequestEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.finished = append(s.finished, event)
	return nil
}

func (s *accountingSpy) RecordAttempt(_ context.Context, event AttemptEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.attempts = append(s.attempts, event)
	return nil
}

func (s *accountingSpy) records() ([]RequestEvent, []AttemptEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]RequestEvent(nil), s.finished...), append([]AttemptEvent(nil), s.attempts...)
}

// waitRecords 等待至少 want 条结束记录；中继在独立 goroutine 中运行，需要等待完成。
func waitRecords(t *testing.T, spy *accountingSpy, want int) []RequestEvent {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		records, _ := spy.records()
		if len(records) >= want {
			return records
		}
		time.Sleep(10 * time.Millisecond)
	}
	records, attempts := spy.records()
	t.Fatalf("记账记录不足，得到 %d 条记录 %v 与 %d 条尝试", len(records), records, len(attempts))
	return nil
}

// passthroughSocketServer 用固定上游启动透明转发网关，并注入记账观察者。
func passthroughSocketServer(t *testing.T, upstreamURL string, spy *accountingSpy) *Server {
	t.Helper()
	session := &stubSession{attempts: []PassthroughAttempt{{
		AccountID: "acct_socket",
		BaseURL:   upstreamURL,
		Header:    http.Header{"Authorization": {"Bearer pool-token"}},
		Snapshot: TargetSnapshot{ProviderID: "builtin_openai", ProviderKind: "openai", AccountID: "acct_socket",
			AccountName: "socket account", AuthMethod: "subscription_login", Email: "socket@example.invalid", WorkspaceID: "ws_socket"},
		Model: "gpt-6.1-sol",
	}}}
	server := New(Options{Port: 0, Passthrough: stubBinding{session}, Upstream: unusedUpstream(t), Observer: spy})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = server.Stop(context.Background()) })
	return server
}

func dialAccountingSocket(t *testing.T, server *Server) (*websocket.Conn, context.Context) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	url := "ws" + strings.TrimPrefix(server.Status().Address, "http") + "/codex/v1/responses"
	socket, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		t.Fatalf("dial gateway websocket: %v", err)
	}
	t.Cleanup(func() { socket.CloseNow() })
	return socket, ctx
}

// TestPassthroughSocketRecordsTurnUsage 断言透明中继的每一轮都落地请求与 token 记录。
func TestPassthroughSocketRecordsTurnUsage(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		socket, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer socket.CloseNow()
		for {
			if _, _, err := socket.Read(r.Context()); err != nil {
				return
			}
			for _, event := range []string{
				`{"type":"response.created","response":{"id":"resp_socket"}}`,
				`{"type":"response.output_text.delta","delta":"hi"}`,
				`{"type":"response.completed","response":{"id":"resp_socket","usage":{"input_tokens":12,"output_tokens":4}}}`,
			} {
				if err := socket.Write(r.Context(), websocket.MessageText, []byte(event)); err != nil {
					return
				}
			}
		}
	}))
	defer upstream.Close()

	spy := &accountingSpy{}
	server := passthroughSocketServer(t, upstream.URL, spy)
	socket, ctx := dialAccountingSocket(t, server)

	for turn := 0; turn < 2; turn++ {
		if err := socket.Write(ctx, websocket.MessageText, []byte(`{"type":"response.create","model":"gpt-6.1-sol","input":[]}`)); err != nil {
			t.Fatal(err)
		}
		for range 3 {
			if _, _, err := socket.Read(ctx); err != nil {
				t.Fatalf("read relayed frame: %v", err)
			}
		}
	}
	records := waitRecords(t, spy, 2)
	_, attempts := spy.records()
	for _, record := range records {
		if record.Transport != "websocket" || record.Protocol != "responses" || record.Model != "gpt-6.1-sol" {
			t.Fatalf("请求归属不正确: %+v", record)
		}
		if record.State != "success" || record.StatusCode != http.StatusOK || record.ErrorCode != "" {
			t.Fatalf("成功轮被记成失败: %+v", record)
		}
		if !record.Usage.Complete || record.Usage.Input != 12 || record.Usage.Output != 4 {
			t.Fatalf("token 用量未落地: %+v", record.Usage)
		}
		if record.Target.AccountID != "acct_socket" || record.Target.Email != "socket@example.invalid" || record.Target.WorkspaceID != "ws_socket" {
			t.Fatalf("账号身份未绑定: %+v", record.Target)
		}
	}
	// 每次尝试先登记 pending 再更新结果，与 HTTP 路径一致。
	settled := 0
	for _, attempt := range attempts {
		if attempt.State == "pending" {
			continue
		}
		settled++
		if attempt.State != "success" || attempt.StatusCode != http.StatusOK || attempt.Usage.Input != 12 {
			t.Fatalf("尝试明细不正确: %+v", attempt)
		}
	}
	if settled != 2 || len(attempts) != 4 {
		t.Fatalf("尝试明细数量不正确: %d / %d", settled, len(attempts))
	}
}

// TestPassthroughSocketTurnEndsWithoutTerminalEvent 断言没有终结事件就断开时按真实原因收尾：
// 客户端主动断开是未完成的回答，上游抖动才是传输失败。
func TestPassthroughSocketTurnEndsWithoutTerminalEvent(t *testing.T) {
	t.Run("client disconnect is cancelled", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			socket, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
			if err != nil {
				return
			}
			defer socket.CloseNow()
			if _, _, err := socket.Read(r.Context()); err != nil {
				return
			}
			_ = socket.Write(r.Context(), websocket.MessageText, []byte(`{"type":"response.created","response":{"id":"resp_wait"}}`))
			<-r.Context().Done()
		}))
		defer upstream.Close()

		spy := &accountingSpy{}
		server := passthroughSocketServer(t, upstream.URL, spy)
		socket, ctx := dialAccountingSocket(t, server)
		if err := socket.Write(ctx, websocket.MessageText, []byte(`{"type":"response.create","model":"gpt-6.1-sol"}`)); err != nil {
			t.Fatal(err)
		}
		if _, _, err := socket.Read(ctx); err != nil {
			t.Fatalf("read relayed frame: %v", err)
		}
		socket.CloseNow()
		record := waitRecords(t, spy, 1)[0]
		if record.State != "cancelled" || record.ErrorCode != "request_cancelled" || record.StatusCode != socketCancelledStatus {
			t.Fatalf("客户端断开被记成传输失败: %+v", record)
		}
	})

	t.Run("upstream disconnect is failure", func(t *testing.T) {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			socket, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
			if err != nil {
				return
			}
			if _, _, err := socket.Read(r.Context()); err != nil {
				return
			}
			_ = socket.Write(r.Context(), websocket.MessageText, []byte(`{"type":"response.created","response":{"id":"resp_cut"}}`))
			socket.CloseNow()
		}))
		defer upstream.Close()

		spy := &accountingSpy{}
		server := passthroughSocketServer(t, upstream.URL, spy)
		socket, ctx := dialAccountingSocket(t, server)
		if err := socket.Write(ctx, websocket.MessageText, []byte(`{"type":"response.create","model":"gpt-6.1-sol"}`)); err != nil {
			t.Fatal(err)
		}
		_, _, _ = socket.Read(ctx)
		record := waitRecords(t, spy, 1)[0]
		if record.State != "failed" || record.ErrorCode != "upstream_disconnected" || record.StatusCode != http.StatusBadGateway {
			t.Fatalf("上游中途断开被记成成功: %+v", record)
		}
	})
}

// TestPassthroughSocketWarmupIsNotRecorded 断言 generate:false 的预热轮不产生请求记录。
func TestPassthroughSocketWarmupIsNotRecorded(t *testing.T) {
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

	spy := &accountingSpy{}
	server := passthroughSocketServer(t, echo.URL, spy)
	socket, ctx := dialAccountingSocket(t, server)
	if err := socket.Write(ctx, websocket.MessageText, []byte(`{"type":"response.create","model":"gpt-6.1-sol","generate":false}`)); err != nil {
		t.Fatal(err)
	}
	if _, _, err := socket.Read(ctx); err != nil {
		t.Fatalf("warmup frame was not relayed: %v", err)
	}
	records, attempts := spy.records()
	if len(records) != 0 || len(attempts) != 0 {
		t.Fatalf("预热轮不应产生记账: %v / %v", records, attempts)
	}
}
