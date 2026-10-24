package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stubSession 用固定尝试序列实现转发会话，便于断言网络层行为。
type stubSession struct {
	attempts  []PassthroughAttempt
	retryable map[int]bool
	done      []int
	index     int
}

func (s *stubSession) Next(context.Context) (PassthroughAttempt, error) {
	if s.index >= len(s.attempts) {
		return PassthroughAttempt{}, Failure("no_account", 503)
	}
	attempt := s.attempts[s.index]
	s.index++
	return attempt, nil
}

func (s *stubSession) Retryable(_ PassthroughAttempt, status int) bool { return s.retryable[status] }

func (s *stubSession) Done(_ context.Context, _ PassthroughAttempt, status int) {
	s.done = append(s.done, status)
}

type stubBinding struct{ session *stubSession }

func (b stubBinding) OpenPassthrough(context.Context, PassthroughRequest) (PassthroughSession, error) {
	return b.session, nil
}

// unusedUpstream 占位满足 Server 的协议上游要求：透明转发路径不得调用它。
func unusedUpstream(t *testing.T) upstreamFunc {
	t.Helper()
	return func(context.Context, Request, func(Response) error) error {
		t.Error("protocol upstream must not be used on the passthrough path")
		return nil
	}
}

// TestPassthroughURLJoinsOfficialCodexPath 覆盖 v1 的路径归一规则。
func TestPassthroughURLJoinsOfficialCodexPath(t *testing.T) {
	cases := []struct{ base, path, want string }{
		{"https://chatgpt.com/backend-api/codex", "/v1/responses", "https://chatgpt.com/backend-api/codex/responses"},
		{"https://chatgpt.com/backend-api/codex", "/v1/responses/compact", "https://chatgpt.com/backend-api/codex/responses/compact"},
		{"https://chatgpt.com/backend-api/codex", "/responses", "https://chatgpt.com/backend-api/codex/responses"},
		{"https://chatgpt.com/backend-api/codex", "/v1/models", "https://chatgpt.com/backend-api/codex/models"},
		{"https://mirror.example/codex/v1", "/v1/responses", "https://mirror.example/codex/v1/responses"},
		{"https://mirror.example/codex/v1", "/responses/compact", "https://mirror.example/codex/v1/responses/compact"},
	}
	for _, item := range cases {
		got, err := PassthroughURL(item.base, item.path, "")
		if err != nil || got != item.want {
			t.Fatalf("PassthroughURL(%q,%q) = %q, %v; want %q", item.base, item.path, got, err, item.want)
		}
	}
	if _, err := PassthroughURL("ftp://example.test/x", "/responses", ""); err == nil {
		t.Fatal("expected非 http(s) 基址被拒绝")
	}
}

// TestPassthroughForwardsClientHeadersAndReplacesAuth 断言头部透传与鉴权替换。
func TestPassthroughForwardsClientHeadersAndReplacesAuth(t *testing.T) {
	var gotPath string
	var gotHeader http.Header
	var gotBody string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotHeader = r.URL.Path, r.Header.Clone()
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_1"}`))
	}))
	defer upstream.Close()

	session := &stubSession{attempts: []PassthroughAttempt{{
		BaseURL: upstream.URL + "/backend-api/codex",
		Header:  http.Header{"Authorization": {"Bearer pool-token"}, "ChatGPT-Account-ID": {"acct_1"}},
	}}}
	server := New(Options{Port: 0, Passthrough: stubBinding{session}, Upstream: unusedUpstream(t)})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = server.Stop(context.Background()) }()

	request, err := http.NewRequest(http.MethodPost, server.Status().Address+"/codex/v1/responses", strings.NewReader(`{"model":"gpt-5.5","stream":true}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer official-login-token")
	request.Header.Set("originator", "codex_cli_rs")
	request.Header.Set("session_id", "sess-1")
	request.Header.Set("OpenAI-Beta", "responses_websockets=2026-02-06")
	request.Header.Set("X-OmniProxy-Key", "must-not-leak")
	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if gotPath != "/backend-api/codex/responses" || gotBody != `{"model":"gpt-5.5","stream":true}` {
		t.Fatalf("unexpected upstream request: path=%q body=%q", gotPath, gotBody)
	}
	if got := gotHeader.Get("Authorization"); got != "Bearer pool-token" {
		t.Fatalf("Authorization = %q", got)
	}
	for _, key := range []string{"originator", "session_id", "OpenAI-Beta"} {
		if gotHeader.Get(key) == "" {
			t.Fatalf("client header %s was dropped", key)
		}
	}
	if gotHeader.Get("ChatGPT-Account-ID") != "acct_1" {
		t.Fatalf("ChatGPT-Account-ID = %q", gotHeader.Get("ChatGPT-Account-ID"))
	}
	if gotHeader.Get("X-OmniProxy-Key") != "" {
		t.Fatal("local access key leaked to upstream")
	}
}

// TestPassthroughSwitchesAccountOnRetryableStatus 断言明确拒绝才换号。
func TestPassthroughSwitchesAccountOnRetryableStatus(t *testing.T) {
	first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"limited"}`))
	}))
	defer first.Close()
	second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_ok"}`))
	}))
	defer second.Close()

	session := &stubSession{retryable: map[int]bool{429: true}, attempts: []PassthroughAttempt{
		{BaseURL: first.URL, AccountID: "acct_a", Header: http.Header{"Authorization": {"Bearer a"}}},
		{BaseURL: second.URL, AccountID: "acct_b", Header: http.Header{"Authorization": {"Bearer b"}}},
	}}
	server := New(Options{Port: 0, Passthrough: stubBinding{session}, Upstream: unusedUpstream(t)})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = server.Stop(context.Background()) }()

	resp, err := http.Post(server.Status().Address+"/codex/v1/responses", "application/json", strings.NewReader(`{"model":"gpt-5.5"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200 after switching account", resp.StatusCode)
	}
	if len(session.done) != 2 || session.done[0] != 429 || session.done[1] != 200 {
		t.Fatalf("attempt outcomes = %v", session.done)
	}
}

// TestPassthroughAcceptsLegacyBackendAPIPrefix 断言 v1 的另一种客户端基址
// `…/backend-api/codex` 仍然可用，并与 `/codex/v1` 走同一套路径归一。
func TestPassthroughAcceptsLegacyBackendAPIPrefix(t *testing.T) {
	for _, path := range []string{"/backend-api/codex/responses", "/backend-api/codex/v1/responses"} {
		t.Run(path, func(t *testing.T) {
			var gotPath string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"resp_legacy"}`))
			}))
			defer upstream.Close()

			session := &stubSession{attempts: []PassthroughAttempt{{
				BaseURL: upstream.URL + "/backend-api/codex",
				Header:  http.Header{"Authorization": {"Bearer pool-token"}},
			}}}
			server := New(Options{Port: 0, Passthrough: stubBinding{session}, Upstream: unusedUpstream(t)})
			if _, err := server.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() { _, _ = server.Stop(context.Background()) }()

			resp, err := http.Post(server.Status().Address+path, "application/json", strings.NewReader(`{"model":"gpt-5.5"}`))
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK || gotPath != "/backend-api/codex/responses" {
				t.Fatalf("status=%d upstream path=%q", resp.StatusCode, gotPath)
			}
		})
	}
}

// TestPassthroughReturnsUpstreamRejectionVerbatim 断言非明确拒绝不换号、不改写正文。
func TestPassthroughReturnsUpstreamRejectionVerbatim(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"bad model"}}`))
	}))
	defer upstream.Close()

	session := &stubSession{attempts: []PassthroughAttempt{
		{BaseURL: upstream.URL, AccountID: "acct_a", Header: http.Header{"Authorization": {"Bearer a"}}},
		{BaseURL: upstream.URL, AccountID: "acct_b", Header: http.Header{"Authorization": {"Bearer b"}}},
	}}
	server := New(Options{Port: 0, Passthrough: stubBinding{session}, Upstream: unusedUpstream(t)})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = server.Stop(context.Background()) }()

	resp, err := http.Post(server.Status().Address+"/codex/v1/responses", "application/json", strings.NewReader(`{"model":"gpt-5.5"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(body), "bad model") {
		t.Fatalf("status=%d body=%q", resp.StatusCode, body)
	}
	if len(session.done) != 1 {
		t.Fatalf("non-retryable rejection switched account: %v", session.done)
	}
}
