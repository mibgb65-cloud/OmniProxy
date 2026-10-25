package proxy

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

type socketBindingFunc func(context.Context, PassthroughRequest) (PassthroughSession, error)

func (f socketBindingFunc) OpenPassthrough(ctx context.Context, r PassthroughRequest) (PassthroughSession, error) {
	return f(ctx, r)
}

type slowSocketSession struct{ failingSession }

func (slowSocketSession) Next(ctx context.Context) (PassthroughAttempt, error) {
	<-ctx.Done()
	return PassthroughAttempt{}, errors.New("private-credential-material")
}

type cancelledSuccessSession struct {
	failingSession
	cancel context.CancelFunc
	target PassthroughAttempt
	done   int
}

func (s *cancelledSuccessSession) Next(context.Context) (PassthroughAttempt, error) {
	s.cancel()
	return s.target, nil
}

func (s *cancelledSuccessSession) Done(context.Context, PassthroughAttempt, PassthroughOutcome) {
	s.done++
}

func (s *cancelledSuccessSession) Switchable() bool { return true }

func TestSocketCancelledPreparationDiscardsEvenSuccessfulLateResult(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	server := New(Options{})
	ctx, finish := server.socketHandshakeContext(parent)
	defer finish()
	header := http.Header{"Authorization": {"Bearer test-only-token"}}
	session := &cancelledSuccessSession{cancel: cancel, target: PassthroughAttempt{
		AccountID: "a", BaseURL: "invalid-target-must-not-be-evaluated", Header: header,
	}}
	request := httptest.NewRequest(http.MethodGet, "http://localhost/codex/v1/responses", nil).WithContext(ctx)
	socket, _, err := server.dialPassthroughSocket(request, session, PassthroughRequest{}, nil)
	if socket != nil || !errors.Is(err, context.Canceled) || session.done != 1 || len(header) != 0 {
		t.Fatal("取消后仍使用迟到的准备结果或遗留租约／凭据", err)
	}
}

type socketLogBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *socketLogBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(data)
}

func (b *socketLogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}

func startSocketFixture(t *testing.T, opts Options) string {
	t.Helper()
	opts.Upstream = unusedUpstream(t)
	server := New(opts)
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = server.Stop(context.Background()) })
	return "ws" + strings.TrimPrefix(server.Status().Address, "http") + "/codex/v1/responses"
}

func TestSocketHandshakeTotalBudgetAndSafeStageDiagnostics(t *testing.T) {
	for _, phase := range []string{"selection", "credentials", "upstream_dial"} {
		t.Run(phase, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				<-r.Context().Done()
			}))
			defer upstream.Close()
			logs := &socketLogBuffer{}
			binding := socketBindingFunc(func(ctx context.Context, _ PassthroughRequest) (PassthroughSession, error) {
				switch phase {
				case "selection":
					<-ctx.Done()
					return nil, errors.New("private-credential-material")
				case "credentials":
					return slowSocketSession{}, nil
				default:
					return &stubSession{attempts: []PassthroughAttempt{{BaseURL: upstream.URL, AccountID: "a"}}}, nil
				}
			})
			url := startSocketFixture(t, Options{Passthrough: binding, SocketHandshakeTimeout: 60 * time.Millisecond,
				Logger: slog.New(slog.NewJSONHandler(logs, nil))})
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			start := time.Now()
			socket, resp, err := websocket.Dial(ctx, url, nil)
			if socket != nil {
				socket.CloseNow()
			}
			if err == nil || resp == nil || resp.StatusCode != 504 || time.Since(start) > time.Second {
				t.Fatal("握手未在总预算内明确超时")
			}
			if resp.Body != nil {
				resp.Body.Close()
			}
			text := logs.String()
			if !strings.Contains(text, `"stage":"`+phase+`"`) || !strings.Contains(text, `"code":"request_timeout"`) ||
				!strings.Contains(text, `"connection_id":`) || strings.Contains(text, "private-credential-material") {
				t.Fatal("阶段诊断缺失或泄露底层错误")
			}
		})
	}
}

func TestSocketHandshakeRechecksPolicyBeforeAccept(t *testing.T) {
	var allowed atomic.Bool
	allowed.Store(true)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed.Store(false)
		socket, err := websocket.Accept(w, r, nil)
		if err == nil {
			defer socket.CloseNow()
			_, _, _ = socket.Read(r.Context())
		}
	}))
	defer upstream.Close()
	session := &stubSession{attempts: []PassthroughAttempt{{BaseURL: upstream.URL, AccountID: "a"}}}
	url := startSocketFixture(t, Options{Passthrough: stubBinding{session},
		WebSocketAllowed: func(context.Context) (bool, error) { return allowed.Load(), nil }})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	socket, resp, err := websocket.Dial(ctx, url, nil)
	if socket != nil {
		socket.CloseNow()
	}
	if err == nil || resp == nil || resp.StatusCode != 403 {
		t.Fatal("准备期间关闭开关后仍接受了握手")
	}
	if resp.Body != nil {
		resp.Body.Close()
	}
}

func TestSocketHandshakeBudgetDoesNotLimitRelayAndWarmConnections(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		socket, err := websocket.Accept(w, r, nil)
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
	binding := socketBindingFunc(func(context.Context, PassthroughRequest) (PassthroughSession, error) {
		return &stubSession{attempts: []PassthroughAttempt{{BaseURL: upstream.URL, AccountID: "a"}}}, nil
	})
	url := startSocketFixture(t, Options{Passthrough: binding, SocketHandshakeTimeout: 200 * time.Millisecond})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	durations := make([]time.Duration, 0, 20)
	for i := range 20 {
		start := time.Now()
		socket, _, err := websocket.Dial(ctx, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		durations = append(durations, time.Since(start))
		if i == 0 {
			time.Sleep(250 * time.Millisecond)
		}
		for _, kind := range []websocket.MessageType{websocket.MessageText, websocket.MessageBinary} {
			if err := socket.Write(ctx, kind, []byte("round-trip")); err != nil {
				t.Fatal("握手预算错误地取消了中继", err)
			}
			gotKind, data, err := socket.Read(ctx)
			if err != nil || gotKind != kind || string(data) != "round-trip" {
				t.Fatal("中继内容或生命周期错误", err)
			}
		}
		socket.CloseNow()
	}
	slices.Sort(durations)
	t.Logf("20 次本地握手 P95=%v", durations[18])
	if durations[18] >= 2*time.Second {
		t.Fatal("本地健康握手 P95 超出目标")
	}
}

func TestTransparentSocketPreservesAuthorizationModeOnEveryFrame(t *testing.T) {
	for _, mode := range []string{"access_key", "official_prefix", "changed_client"} {
		t.Run(mode, func(t *testing.T) {
			byPrefix := mode == "official_prefix"
			var allowed atomic.Bool
			var frames atomic.Int32
			allowed.Store(true)
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				socket, err := websocket.Accept(w, r, nil)
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
			binding := socketBindingFunc(func(context.Context, PassthroughRequest) (PassthroughSession, error) {
				return &stubSession{attempts: []PassthroughAttempt{{AccountID: "a", BaseURL: upstream.URL}}}, nil
			})
			url := startSocketFixture(t, Options{Passthrough: binding,
				Authorize: func(_ context.Context, key string) (string, bool) {
					if mode == "changed_client" && !allowed.Load() {
						return "claude", true
					}
					if len(key) > 512 {
						t.Error("官方长令牌进入接入 Key 比较")
					}
					return "codex", !byPrefix && allowed.Load() && key == "test-only-key"
				},
				PrefixAccess: func(context.Context, string) (string, bool) {
					// 接入 Key 被撤销时前缀仍开启，不能因此降级放行原 Key 会话。
					return "codex", !byPrefix || allowed.Load()
				},
			})
			token := "test-only-key"
			if byPrefix {
				token = strings.Repeat("x", 1400)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			socket, _, err := websocket.Dial(ctx, url, &websocket.DialOptions{
				HTTPHeader: http.Header{"Authorization": {"Bearer " + token}},
			})
			if err != nil {
				t.Fatal(err)
			}
			defer socket.CloseNow()
			if err := socket.Write(ctx, websocket.MessageText, []byte("first")); err != nil {
				t.Fatal(err)
			}
			if _, _, err := socket.Read(ctx); err != nil {
				t.Fatal("授权有效时第一帧被拒绝", err)
			}
			allowed.Store(false)
			_ = socket.Write(ctx, websocket.MessageText, []byte("revoked"))
			if _, _, err := socket.Read(ctx); err == nil || frames.Load() != 1 {
				t.Fatal("撤销原授权后仍转发了后续帧")
			}
		})
	}
}

func TestSocketRelayDiagnosticsDoNotLeakCloseReason(t *testing.T) {
	logs := &socketLogBuffer{}
	server := New(Options{Logger: slog.New(slog.NewJSONHandler(logs, nil))})
	ctx, cancel := server.socketHandshakeContext(context.Background())
	defer cancel()
	stage := BeginSocketStage(ctx, "relay")
	stage(socketRelayFailure(ctx, websocket.CloseError{Code: websocket.StatusNormalClosure, Reason: "private-credential-material"}, "upstream_disconnected"))
	text := logs.String()
	if strings.Contains(text, "private-credential-material") || !strings.Contains(text, `"close_code":1000`) ||
		!strings.Contains(text, `"code":""`) || !strings.Contains(text, `"direction":"upstream_disconnected"`) {
		t.Fatal("关闭事件被误报为失败或泄露了原因")
	}
}
