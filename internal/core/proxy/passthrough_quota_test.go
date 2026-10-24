package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// quotaDeniedUpstream 复刻官方以 HTTP 200 包装执行前额度拒绝的形态：不带媒体类型，
// 正文只有一条明确的 usage_limit_reached。
func quotaDeniedUpstream(t *testing.T, reset time.Time, calls *int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls != nil {
			*calls++
		}
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, fmt.Sprintf(`{"error":{"type":"usage_limit_reached","message":"private-quota-marker","resets_at":%d}}`, reset.Unix()))
	}))
}

func completedStreamUpstream(t *testing.T, calls *int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls != nil {
			*calls++
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_ok\"}}\n\n")
	}))
}

// startPassthroughServer 只开放透明入口，协议上游保持不可达。
func startPassthroughServer(t *testing.T, session *stubSession) *Server {
	t.Helper()
	server := New(Options{Port: 0, Passthrough: stubBinding{session}, Upstream: unusedUpstream(t)})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = server.Stop(context.Background()) })
	return server
}

func postCodexResponses(t *testing.T, server *Server) (int, string) {
	t.Helper()
	resp, err := http.Post(server.Status().Address+"/codex/v1/responses", "application/json", strings.NewReader(`{"model":"gpt-test"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "private-quota-marker") {
		t.Fatalf("raw quota error reached the client: %q", body)
	}
	return resp.StatusCode, string(body)
}

// TestPassthroughSwitchesAccountOnPreExecutionQuotaRejection 断言 HTTP 200 包装的执行前
// 额度拒绝在透明转发里也能换号，并且这段拒绝字节绝不交付给客户端。
func TestPassthroughSwitchesAccountOnPreExecutionQuotaRejection(t *testing.T) {
	reset := time.Now().Add(6 * 24 * time.Hour).Truncate(time.Second)
	deniedCalls, okCalls := 0, 0
	denied := quotaDeniedUpstream(t, reset, &deniedCalls)
	defer denied.Close()
	ok := completedStreamUpstream(t, &okCalls)
	defer ok.Close()

	session := &stubSession{switchable: true, attempts: []PassthroughAttempt{
		{BaseURL: denied.URL, AccountID: "acct_a", Header: http.Header{"Authorization": {"Bearer a"}}},
		{BaseURL: ok.URL, AccountID: "acct_b", Header: http.Header{"Authorization": {"Bearer b"}}},
	}}
	status, body := postCodexResponses(t, startPassthroughServer(t, session))

	if status != http.StatusOK || !strings.Contains(body, "response.completed") {
		t.Fatalf("status = %d body = %q", status, body)
	}
	if deniedCalls != 1 || okCalls != 1 {
		t.Fatalf("upstream calls: denied=%d ok=%d", deniedCalls, okCalls)
	}
	if len(session.done) != 2 {
		t.Fatalf("attempt outcomes = %v", session.done)
	}
	if first := session.done[0]; !first.UpstreamQuotaRejected || first.StatusCode != http.StatusOK || !first.QuotaResetAt.Equal(reset) {
		t.Fatalf("quota attempt must keep the real status and reset time: %+v", first)
	}
	if second := session.done[1]; second.UpstreamQuotaRejected || second.StatusCode != http.StatusOK {
		t.Fatalf("successful attempt misclassified: %+v", second)
	}
}

// TestPassthroughReturnsControlledQuotaFailureWhenAllAccountsExhausted 断言全部候选都被
// 执行前额度拒绝时，客户端收到受控 429，明细保留上游真实状态，且不泄露上游正文。
func TestPassthroughReturnsControlledQuotaFailureWhenAllAccountsExhausted(t *testing.T) {
	reset := time.Now().Add(time.Hour).Truncate(time.Second)
	denied := quotaDeniedUpstream(t, reset, nil)
	defer denied.Close()
	session := &stubSession{switchable: true, attempts: []PassthroughAttempt{
		{BaseURL: denied.URL, AccountID: "acct_a", Header: http.Header{"Authorization": {"Bearer a"}}},
	}}
	status, body := postCodexResponses(t, startPassthroughServer(t, session))

	if status != http.StatusTooManyRequests || !strings.Contains(body, "upstream_quota_exhausted") {
		t.Fatalf("status = %d body = %q", status, body)
	}
	if len(session.done) != 1 || !session.done[0].UpstreamQuotaRejected {
		t.Fatalf("attempt outcomes = %v", session.done)
	}
}

// TestPassthroughFixedAccountKeepsQuotaRejectionWithoutSwitching 断言固定账号或关闭故障
// 切换时仍会把额度拒绝交给宿主机登记，但不再遍历下一个候选。
func TestPassthroughFixedAccountKeepsQuotaRejectionWithoutSwitching(t *testing.T) {
	reset := time.Now().Add(time.Hour).Truncate(time.Second)
	denied := quotaDeniedUpstream(t, reset, nil)
	defer denied.Close()
	okCalls := 0
	ok := completedStreamUpstream(t, &okCalls)
	defer ok.Close()

	session := &stubSession{attempts: []PassthroughAttempt{
		{BaseURL: denied.URL, AccountID: "acct_fixed", Header: http.Header{"Authorization": {"Bearer a"}}},
		{BaseURL: ok.URL, AccountID: "acct_backup", Header: http.Header{"Authorization": {"Bearer b"}}},
	}}
	status, body := postCodexResponses(t, startPassthroughServer(t, session))

	if status != http.StatusTooManyRequests || okCalls != 0 {
		t.Fatalf("status = %d backup calls = %d body = %q", status, okCalls, body)
	}
	if len(session.done) != 1 || !session.done[0].UpstreamQuotaRejected {
		t.Fatalf("fixed account must still report the quota rejection: %v", session.done)
	}
}

// TestPassthroughDeliversLargeFirstEventUnchanged 断言首事件超过预读上限时仍按原字节交付：
// 判定不能因为读不完事件而截断或延迟真实回答。
func TestPassthroughDeliversLargeFirstEventUnchanged(t *testing.T) {
	huge := strings.Repeat("x", 8<<10)
	payload := "data: {\"type\":\"response.created\",\"response\":{\"id\":\"resp_big\",\"tools\":[{\"description\":\"" + huge + "\"}]}}\n\n" +
		"data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_big\"}}\n\n"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 与官方一致地不声明 Content-Type，走推断分支。
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, payload)
	}))
	defer upstream.Close()

	session := &stubSession{attempts: []PassthroughAttempt{{
		BaseURL: upstream.URL, AccountID: "acct_a", Header: http.Header{"Authorization": {"Bearer a"}},
	}}}
	status, body := postCodexResponses(t, startPassthroughServer(t, session))
	if status != http.StatusOK || body != payload {
		t.Fatalf("status = %d, body length = %d (want %d)", status, len(body), len(payload))
	}
	if len(session.done) != 1 || session.done[0].UpstreamQuotaRejected {
		t.Fatalf("large first event was misclassified: %v", session.done)
	}
}
