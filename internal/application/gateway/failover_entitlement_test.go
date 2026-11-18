package gateway

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/application/provider"
	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/deepseek"
	"omniproxy/internal/provider/workbuddyai"
)

// 账号级拒绝只在能换号时才参与分类；服务商级拒绝换号会得到同样的拒绝。
func TestAccountRejectionClassifiesAccountLevelRefusals(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code string
	}{
		{"entitlement", proxy.Failure("model_not_entitled", 403), "model_not_entitled"},
		{"sign in", proxy.Failure("sign_in_expired", 401), "sign_in_expired"},
		{"upstream 401", proxy.Failure("upstream_authentication_failed", 401), "upstream_authentication_failed"},
		{"channel", proxy.Failure("channel_rejected", 400), ""},
		{"fingerprint", proxy.Failure("user_agent_refused", 400), ""},
		{"upstream down", proxy.Failure("upstream_unavailable", 502), ""},
		{"untranslated business code", workbuddyai.Failure(workbuddyai.CodeModelNotEntitled), ""},
		{"unrelated", errors.New("boom"), ""},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			reason, failure := accountRejection(test.err)
			if test.code == "" {
				if reason != nil || failure != nil {
					t.Fatalf("unexpected rejection: %+v", reason)
				}
				return
			}
			if reason == nil || reason.code != test.code || reason.delay <= 0 || failure == nil || failure.Code != test.code {
				t.Fatalf("wrong rejection: %+v %+v", reason, failure)
			}
		})
	}
}

type stubWorkBuddyAccess struct{ credential workbuddyai.Credentials }

func (s stubWorkBuddyAccess) Use(_ context.Context, _ string, consume func(workbuddyai.Credentials) error) error {
	return consume(s.credential)
}

func (s stubWorkBuddyAccess) UseAt(ctx context.Context, id, _ string, consume func(workbuddyai.Credentials) error) error {
	return s.Use(ctx, id, consume)
}

// callLog 记录适配器实际发出的上游请求数，供并发安全的断言使用。
type callLog struct {
	mu    sync.Mutex
	paths []string
}

func (c *callLog) add(path string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.paths = append(c.paths, path)
	return len(c.paths)
}

func (c *callLog) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.paths)
}

// workBuddyFixture 建一个或两个 WorkBuddy AI 登录账号，都发布同一个模型，
// 并把适配器指向契约服务器。目录按账号版本绑定，只有版本一致的列表参与路由。
func workBuddyFixture(t *testing.T, count int, handler http.HandlerFunc) (*Service, *activity.Service) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "workbuddy.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	rows := `INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_email,auth_state,auth_account_id) VALUES
	  ('acct_a','builtin_workbuddy_ai','alpha','ref-a',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z','workbuddy_login','alpha@example.invalid','authenticated','uid-a'),
	  ('acct_b','builtin_workbuddy_ai','beta','ref-b',1,'2026-01-02T00:00:00Z','2026-01-02T00:00:00Z','workbuddy_login','beta@example.invalid','authenticated','uid-b')`
	if count == 1 {
		rows = `INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_email,auth_state,auth_account_id) VALUES
	  ('acct_a','builtin_workbuddy_ai','alpha','ref-a',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z','workbuddy_login','alpha@example.invalid','authenticated','uid-a')`
	}
	if _, err = db.Exec(rows); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version,metadata_json)
	  SELECT a.id,'gemini-4-carbon','Gemini-4-Carbon','2026-01-01T00:00:00Z',v.version,'{}'
	  FROM accounts a JOIN account_catalog_versions v ON a.id=v.id`); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := workbuddyai.NewClientAtFor(workbuddyai.International, server.URL, server.Client().Transport)
	if err != nil {
		t.Fatal(err)
	}
	providers := provider.NewService(db)
	accountService := account.NewService(db, failoverStore{})
	observer := activity.New(db)
	s := New(proxy.Options{Observer: observer, FailoverEnabled: NewFailoverSettingsService(db).Enabled}, providers, accountService, deepseek.New(nil))
	s.SetRouter(routes.New(db, providers, accountService))
	s.workbuddy[workbuddyai.Kind] = client
	s.SetWorkBuddySubscription(stubWorkBuddyAccess{credential: workbuddyai.Credentials{AccessToken: "fix-access", UID: "uid-a"}})
	if _, err = s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Stop(ctx) })
	return s, observer
}

// 第一个上游调用被 11158 拒绝，其余返回一次成功的 Chat 流。
func entitlementServer(log *callLog) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if log.add(r.URL.Path) == 1 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusForbidden)
			io.WriteString(w, `{"code":11158,"msg":"no entitlement"}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"OK\"}}]}\n\n"+
			"data: {\"choices\":[],\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":4}}\n\n"+
			"data: [DONE]\n\n")
	}
}

// 首个账号被 11158 拒绝时换到同池第二个账号，并给该（账号，模型）写冷却。
func TestFailoverSwitchesAccountOnModelEntitlement(t *testing.T) {
	log := &callLog{}
	s, observer := workBuddyFixture(t, 2, entitlementServer(log))
	ctx := context.Background()
	latest := func() activity.RequestItem {
		t.Helper()
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

	status, body := localGatewayRequest(t, s, "gemini-4-carbon", false)
	if status != http.StatusOK || !gatewayReplyHasContent(body, false) {
		t.Fatalf("first request = %d %s", status, body)
	}
	item := latest()
	if item.AccountID != "acct_b" || item.AttemptCount != 2 {
		t.Fatalf("wrong accounting: %+v", item)
	}
	if len(item.Attempts) != 2 || item.Attempts[0].AccountID != "acct_a" ||
		item.Attempts[0].ErrorCode != "model_not_entitled" || item.Attempts[0].StatusCode != http.StatusForbidden ||
		item.Attempts[1].AccountID != "acct_b" || item.Attempts[1].State != "success" {
		t.Fatalf("wrong attempts: %+v", item.Attempts)
	}

	// 冷却生效：下一次请求不再先撞同一个没有权益的账号。
	status, body = localGatewayRequest(t, s, "gemini-4-carbon", false)
	if status != http.StatusOK || !gatewayReplyHasContent(body, false) {
		t.Fatalf("second request = %d %s", status, body)
	}
	if item := latest(); item.AccountID != "acct_b" || item.AttemptCount != 1 {
		t.Fatalf("cooled account retried: %+v", item)
	}
	if got := log.count(); got != 3 {
		t.Fatalf("upstream calls = %d, want 3", got)
	}
}

// 只有一个候选时把真实拒绝交还客户端，而不是伪装成账号池冷却。
func TestFailoverKeepsEntitlementErrorWithoutCandidate(t *testing.T) {
	log := &callLog{}
	s, observer := workBuddyFixture(t, 1, entitlementServer(log))
	status, body := localGatewayRequest(t, s, "gemini-4-carbon", false)
	if status != http.StatusForbidden || gatewayResponseError(t, body) != "model_not_entitled" {
		t.Fatalf("single candidate = %d %s", status, body)
	}
	page, err := observer.Requests(context.Background(), activity.RequestQuery{Limit: 1})
	if err != nil || len(page.Items) != 1 || page.Items[0].AttemptCount != 1 || page.Items[0].ErrorCode != "model_not_entitled" {
		t.Fatalf("requests: %+v %v", page, err)
	}
	if got := log.count(); got != 1 {
		t.Fatalf("upstream calls = %d, want 1", got)
	}
}

// 记录里的错误码必须是真实码：网关边界翻译后不再兜底成 credential_unavailable，
// 服务商级拒绝（11128）也不会被当成账号级拒绝重复发送。
func TestWorkBuddyBusinessRefusalIsReportedVerbatim(t *testing.T) {
	log := &callLog{}
	s, observer := workBuddyFixture(t, 2, func(w http.ResponseWriter, r *http.Request) {
		log.add(r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"code":11128,"msg":"unapproved channel"}`)
	})
	status, body := localGatewayRequest(t, s, "gemini-4-carbon", false)
	if status != http.StatusBadRequest || gatewayResponseError(t, body) != "channel_rejected" {
		t.Fatalf("refusal = %d %s", status, body)
	}
	page, err := observer.Requests(context.Background(), activity.RequestQuery{Limit: 1})
	if err != nil || len(page.Items) != 1 || page.Items[0].ErrorCode != "channel_rejected" || page.Items[0].AttemptCount != 1 {
		t.Fatalf("requests: %+v %v", page, err)
	}
	// 11128 是渠道问题：适配器自己会把系统提示词下移重试一次，网关不再换号。
	if got := log.count(); got != 2 {
		t.Fatalf("provider-level refusal failover: %d upstream calls", got)
	}
}
