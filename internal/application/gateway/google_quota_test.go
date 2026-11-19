package gateway

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/application/provider"
	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/antigravity"
)

// googlePoolAccess 按账号返回各自的订阅凭据，让换号在测试里真实发生。
type googlePoolAccess struct {
	mu    sync.Mutex
	calls []string
}

func (a *googlePoolAccess) Use(_ context.Context, id string, consume func(antigravity.Credentials) error) error {
	a.mu.Lock()
	a.calls = append(a.calls, id)
	a.mu.Unlock()
	if id != "acct_a" && id != "acct_b" {
		return antigravity.Failure("invalid_input")
	}
	return consume(antigravity.Credentials{AccessToken: "token-" + id, Project: "test-project"})
}

func (a *googlePoolAccess) sequence() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return strings.Join(a.calls, ",")
}

// fakeGoogleGate 用固定的（账号，模型）记录模拟已发布的额度快照。
type fakeGoogleGate struct{ blocked map[string]time.Time }

func gateKey(accountID, model string) string { return accountID + "|" + model }

func (g fakeGoogleGate) BlockedUntil(_ context.Context, accountID, _, model string, now time.Time) (time.Time, bool) {
	if at, ok := g.blocked[gateKey(accountID, model)]; ok && at.After(now) {
		return at, true
	}
	return time.Time{}, false
}

// googleQuotaFixture 建两个同池 Google 订阅账号，都发布同一个模型；
// 上游对 acct_a 返回 429，对 acct_b 返回一次成功响应。
func googleQuotaFixture(t *testing.T) (*Service, *activity.Service, *googlePoolAccess, *callLog) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "google-quota.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_state,auth_account_id,auth_email) VALUES
	  ('acct_a','builtin_google','alpha','ref-a',1,'2026-11-01T00:00:00Z','2026-11-01T00:00:00Z','subscription_login','authenticated','1001','alpha@example.invalid'),
	  ('acct_b','builtin_google','beta','ref-b',1,'2026-11-02T00:00:00Z','2026-11-02T00:00:00Z','subscription_login','authenticated','1002','beta@example.invalid')`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version) VALUES
	  ('acct_a','gemini-test','Gemini','now','ref-a:1001'),
	  ('acct_b','gemini-test','Gemini','now','ref-b:1002')`); err != nil {
		t.Fatal(err)
	}
	providers, accounts := provider.NewService(db), account.NewService(db, &testStore{})
	observer := activity.New(db)
	service := New(proxy.Options{Observer: observer}, providers, accounts, nil)
	access := &googlePoolAccess{}
	service.SetGoogleSubscription(access)
	calls := &callLog{}
	service.antigravity = antigravity.New(googleTransport(func(r *http.Request) (*http.Response, error) {
		calls.add(r.URL.Path)
		if r.Header.Get("Authorization") == "Bearer token-acct_a" {
			return &http.Response{StatusCode: http.StatusTooManyRequests,
				Header: http.Header{"Content-Type": {"application/json"}},
				Body:   io.NopCloser(strings.NewReader(`{"error":{"message":"rate limited"}}`))}, nil
		}
		body := `{"response":{"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":4,"candidatesTokenCount":1,"totalTokenCount":5}}}`
		return &http.Response{StatusCode: http.StatusOK,
			Header: http.Header{"Content-Type": {"application/json"}},
			Body:   io.NopCloser(strings.NewReader(body))}, nil
	}))
	router := routes.New(db, providers, accounts)
	service.SetRouter(router)
	if _, err := router.Create(ctx, routes.Input{Alias: "google-pool", ProviderID: "builtin_google", Model: "gemini-test", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { service.Stop(ctx) })
	return service, observer, access, calls
}

func latestRequest(t *testing.T, observer *activity.Service) activity.RequestItem {
	t.Helper()
	ctx := context.Background()
	page, err := observer.Requests(ctx, activity.RequestQuery{Limit: 1})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("requests: %+v %v", page, err)
	}
	item, err := observer.Request(ctx, page.Items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	return item
}

// googlePoolCall 走真实网关端口发一次 Google 订阅 Agent 请求（chat 协议）。
func googlePoolCall(t *testing.T, service *Service) (int, []byte) {
	t.Helper()
	body, path := googleAgentRequest("google-pool", "chat", false, false)
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/v1/%s", service.Info().Runtime.Port, path)
	response, err := http.Post(endpoint, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, data
}

// 429 是账号级的：同一请求内换到同池另一个账号，并把该（账号，模型）冷却。
func TestGoogleRateLimitSwitchesAccountAndCoolsModel(t *testing.T) {
	service, observer, access, calls := googleQuotaFixture(t)

	status, body := googlePoolCall(t, service)
	if status != http.StatusOK {
		t.Fatalf("first request = %d %s", status, body)
	}
	item := latestRequest(t, observer)
	if item.AttemptCount != 2 || item.AccountID != "acct_b" {
		t.Fatalf("wrong accounting: %+v", item)
	}
	if len(item.Attempts) != 2 || item.Attempts[0].AccountID != "acct_a" ||
		item.Attempts[0].StatusCode != http.StatusTooManyRequests || item.Attempts[0].ErrorCode != "rate_limited" ||
		item.Attempts[1].AccountID != "acct_b" || item.Attempts[1].State != "success" {
		t.Fatalf("wrong attempts: %+v", item.Attempts)
	}

	// 冷却生效：下一条请求不再先撞同一个 429 账号。
	status, body = googlePoolCall(t, service)
	if status != http.StatusOK {
		t.Fatalf("second request = %d %s", status, body)
	}
	if item := latestRequest(t, observer); item.AccountID != "acct_b" || item.AttemptCount != 1 {
		t.Fatalf("cooled account retried: %+v", item)
	}
	if got := access.sequence(); got != "acct_a,acct_b,acct_b" {
		t.Fatalf("credential use = %q", got)
	}
	if got := calls.count(); got != 3 {
		t.Fatalf("upstream calls = %d, want 3", got)
	}
}

// 快照里该模型所属分组已耗尽时不发送：一次尝试都不发生。
func TestGoogleQuotaGateSkipsExhaustedAccount(t *testing.T) {
	service, observer, access, calls := googleQuotaFixture(t)
	service.SetGoogleQuotaGate(fakeGoogleGate{blocked: map[string]time.Time{
		gateKey("acct_a", "gemini-test"): time.Now().Add(10 * time.Minute),
	}})
	status, body := googlePoolCall(t, service)
	if status != http.StatusOK {
		t.Fatalf("request = %d %s", status, body)
	}
	if item := latestRequest(t, observer); item.AccountID != "acct_b" || item.AttemptCount != 1 {
		t.Fatalf("exhausted account was not skipped: %+v", item)
	}
	if got := access.sequence(); got != "acct_b" {
		t.Fatalf("credential use = %q", got)
	}
	if got := calls.count(); got != 1 {
		t.Fatalf("upstream calls = %d, want 1", got)
	}
}

// 同池全部耗尽时快速返回账号池冷却，不向已耗尽账号发送探测。
func TestGoogleQuotaGateReportsPoolCooling(t *testing.T) {
	service, observer, _, calls := googleQuotaFixture(t)
	until := time.Now().Add(5 * time.Minute)
	service.SetGoogleQuotaGate(fakeGoogleGate{blocked: map[string]time.Time{
		gateKey("acct_a", "gemini-test"): until,
		gateKey("acct_b", "gemini-test"): until,
	}})
	status, body := googlePoolCall(t, service)
	if status != http.StatusTooManyRequests || gatewayResponseError(t, body) != "account_pool_cooling" {
		t.Fatalf("pool cooling = %d %s", status, body)
	}
	if item := latestRequest(t, observer); item.AttemptCount != 0 {
		t.Fatalf("attempts on a cooling pool: %+v", item)
	}
	if got := calls.count(); got != 0 {
		t.Fatalf("upstream calls = %d, want 0", got)
	}
}

// 只有 Google 订阅账号参与：API Key 共享项目额度，换号没有意义。
func TestGoogleRateLimitClassificationIsSubscriptionOnly(t *testing.T) {
	service := &Service{failover: newFailoverState()}
	ctx := context.Background()
	subscription := target{kind: "google", authMethod: account.AuthSubscription}
	apiKey := target{kind: "google", authMethod: account.AuthAPIKey}
	limited := &proxy.Error{Code: "rate_limited", StatusCode: http.StatusTooManyRequests}

	if reason, failure := service.googleRateLimited(ctx, apiKey, "gemini-test", limited); reason != nil || failure != nil {
		t.Fatalf("api key account cooled: %+v", reason)
	}
	reason, failure := service.googleRateLimited(ctx, subscription, "gemini-test", limited)
	if reason == nil || failure == nil || reason.code != "rate_limited" || reason.delay != googleRateLimitCooldown {
		t.Fatalf("subscription 429 = %+v %+v", reason, failure)
	}
	// 上游给出重置秒数时以它为准。
	withReset := &proxy.Error{Code: "rate_limited", StatusCode: http.StatusTooManyRequests, RetryAfter: "120"}
	if reason, _ := service.googleRateLimited(ctx, subscription, "gemini-test", withReset); reason == nil || reason.delay != 2*time.Minute {
		t.Fatalf("retry-after ignored: %+v", reason)
	}
	// 与额度无关的失败不写账号级冷却。
	if reason, _ := service.googleRateLimited(ctx, subscription, "gemini-test", proxy.Failure("upstream_unavailable", 502)); reason != nil {
		t.Fatalf("unrelated failure cooled: %+v", reason)
	}

	// 快照给出该模型的重置时间时，冷却到那一刻而不是固定 60 秒。
	until := time.Now().Add(20 * time.Minute)
	service.SetGoogleQuotaGate(fakeGoogleGate{blocked: map[string]time.Time{gateKey("acct_a", "gemini-test"): until}})
	withAccount := subscription
	withAccount.accountID = "acct_a"
	if reason, _ := service.googleRateLimited(ctx, withAccount, "gemini-test", limited); reason == nil || reason.delay < 19*time.Minute {
		t.Fatalf("snapshot reset ignored: %+v", reason)
	}
}

// 预筛只作用于 Google 订阅账号，且超过阻断上限的已知重置不参与。
func TestGoogleQuotaWaitScopeAndBound(t *testing.T) {
	service := &Service{failover: newFailoverState()}
	ctx := context.Background()
	model := "gemini-test"
	blocked := fakeGoogleGate{blocked: map[string]time.Time{
		gateKey("acct_a", model): time.Now().Add(time.Hour),
		gateKey("acct_c", model): time.Now().Add(30 * 24 * time.Hour),
	}}
	service.SetGoogleQuotaGate(blocked)
	now := time.Now()
	if wait := service.googleQuotaWait(ctx, target{kind: "google", authMethod: account.AuthSubscription, accountID: "acct_a"}, model, now); wait <= 0 {
		t.Fatal("exhausted account not gated")
	}
	if wait := service.googleQuotaWait(ctx, target{kind: "google", authMethod: account.AuthAPIKey, accountID: "acct_a"}, model, now); wait != 0 {
		t.Fatal("api key account gated")
	}
	if wait := service.googleQuotaWait(ctx, target{kind: "google", authMethod: account.AuthSubscription, accountID: "acct_c"}, model, now); wait != 0 {
		t.Fatal("beyond-limit reset gated")
	}
	if wait := service.googleQuotaWait(ctx, target{kind: "openai", authMethod: account.AuthSubscription, accountID: "acct_a"}, model, now); wait != 0 {
		t.Fatal("other provider gated")
	}
	// 未安装额度门时保持原有行为。
	plain := &Service{failover: newFailoverState()}
	if wait := plain.googleQuotaWait(ctx, target{kind: "google", authMethod: account.AuthSubscription, accountID: "acct_a"}, model, now); wait != 0 {
		t.Fatal("gate applied without installation")
	}
}
