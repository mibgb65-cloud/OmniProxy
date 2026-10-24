package gateway

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// postCodexPassthrough 走 v2 一键配置写进客户端的 `/codex/v1` 入口。
func postCodexPassthrough(t *testing.T, s *Service) (int, string) {
	t.Helper()
	resp, err := http.Post(s.Info().Runtime.Address+"/codex/v1/responses", "application/json",
		strings.NewReader(`{"model":"gpt-test","input":[{"role":"user","content":"Hi"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "private-quota-marker") {
		t.Fatalf("raw quota error leaked to the client: %q", body)
	}
	return resp.StatusCode, string(body)
}

// TestPassthroughQuotaRejectionSwitchesAccountAndBlocksIt 断言 Codex 透明入口在 HTTP 200
// 包装的执行前额度拒绝时同样换号：客户端只看到备用账号的回答，被拒绝的账号按真实重置
// 时间写入额度阻断，下一条消息不再先撞它。
func TestPassthroughQuotaRejectionSwitchesAccountAndBlocksIt(t *testing.T) {
	s, db, _, access := subscriptionQuotaFixture(t)
	reset := time.Now().Add(6 * 24 * time.Hour).Truncate(time.Second)
	var mu sync.Mutex
	calls := []string{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer placeholder-subscription-")
		mu.Lock()
		calls = append(calls, id)
		mu.Unlock()
		if id == "a" {
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprintf(w, `{"error":{"type":"usage_limit_reached","message":"private-quota-marker","resets_at":%d}}`, reset.Unix())
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_ok\"}}\n\n")
	}))
	defer upstream.Close()
	if _, err := db.Exec(`UPDATE codex_settings SET upstream_base_url=? WHERE id=1`, upstream.URL+"/backend-api/codex"); err != nil {
		t.Fatal(err)
	}
	s.SetCodexUpstreamBase(NewCodexSettingsService(db).UpstreamBaseURL)

	if status, body := postCodexPassthrough(t, s); status != http.StatusOK || !strings.Contains(body, "response.completed") {
		t.Fatalf("status = %d body = %q", status, body)
	}
	mu.Lock()
	first := strings.Join(calls, ",")
	mu.Unlock()
	if first != "a,b" {
		t.Fatalf("candidate sequence = %q, want a then b", first)
	}
	until, blocked := access.QuotaBlockedUntil("a", "", time.Now())
	if !blocked || until.Sub(reset) > time.Minute || reset.Sub(until) > time.Minute {
		t.Fatalf("rejected account not blocked until its real reset: until=%v blocked=%v want≈%v", until, blocked, reset)
	}

	// 第二条消息：已知耗尽的账号直接跳过，不再先撞一次。
	if status, body := postCodexPassthrough(t, s); status != http.StatusOK || !strings.Contains(body, "response.completed") {
		t.Fatalf("second status = %d body = %q", status, body)
	}
	mu.Lock()
	defer mu.Unlock()
	if got := strings.Join(calls, ","); got != "a,b,b" {
		t.Fatalf("candidate sequence = %q, want the exhausted account skipped", got)
	}
}

// TestPassthroughReturnsControlledQuotaFailureWhenPoolIsExhausted 断言整池都被额度拒绝时
// 客户端收到受控 429 与稳定错误码，且不泄露上游正文。
func TestPassthroughReturnsControlledQuotaFailureWhenPoolIsExhausted(t *testing.T) {
	s, db, _, _ := subscriptionQuotaFixture(t)
	// 只保留一个可用账号，让整池必然耗尽。
	if _, err := db.Exec(`UPDATE accounts SET enabled=(id='a')`); err != nil {
		t.Fatal(err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"error":{"type":"usage_limit_reached","message":"private-quota-marker","resets_at":%d}}`, time.Now().Add(time.Hour).Unix())
	}))
	defer upstream.Close()
	if _, err := db.Exec(`UPDATE codex_settings SET upstream_base_url=? WHERE id=1`, upstream.URL+"/backend-api/codex"); err != nil {
		t.Fatal(err)
	}
	s.SetCodexUpstreamBase(NewCodexSettingsService(db).UpstreamBaseURL)

	status, body := postCodexPassthrough(t, s)
	if status != http.StatusTooManyRequests || !strings.Contains(body, "upstream_quota_exhausted") {
		t.Fatalf("status = %d body = %q", status, body)
	}
}

// TestPassthroughRejectionUsesUpstreamRetryBoundary 断言冷却取上游给出的重试提示，
// 而不是一律 30 秒；执行前额度拒绝则按真实重置时间换算。
func TestPassthroughRejectionUsesUpstreamRetryBoundary(t *testing.T) {
	now := time.Now()
	if got := passthroughRejection(http.StatusTooManyRequests, "120", now); got == nil || got.delay != 120*time.Second {
		t.Fatalf("retry-after ignored: %+v", got)
	}
	if got := passthroughRejection(http.StatusTooManyRequests, "", now); got == nil || got.delay != 30*time.Second {
		t.Fatalf("missing retry-after must keep the short default: %+v", got)
	}
	if got := passthroughRejection(http.StatusTooManyRequests, "not-a-date", now); got.delay != 30*time.Second {
		t.Fatalf("invalid retry-after accepted: %+v", got)
	}
	if got := passthroughRejection(http.StatusBadRequest, "120", now); got != nil {
		t.Fatalf("non-rejection status cooled down: %+v", got)
	}

	reset := now.Add(5 * 24 * time.Hour)
	if got := quotaRetryReason(reset, now); got.code != "upstream_quota_exhausted" || got.delay != reset.Sub(now) {
		t.Fatalf("quota reset boundary lost: %+v", got)
	}
	if got := quotaRetryReason(time.Time{}, now); got.delay != 30*time.Second {
		t.Fatalf("unknown reset must only skip briefly: %+v", got)
	}
	if got := quotaRetryReason(now.Add(30*24*time.Hour), now); got.delay != 30*time.Second {
		t.Fatalf("unbounded quota denial: %+v", got)
	}
}
