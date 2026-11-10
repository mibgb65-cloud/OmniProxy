package modelcatalog

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/infrastructure/codex"
	"omniproxy/internal/infrastructure/database"
)

type openAIAutoSource struct {
	mu      sync.Mutex
	models  []codex.Model
	err     error
	calls   []string
	entered chan context.Context
	release chan struct{}
}

func (s *openAIAutoSource) Models(ctx context.Context, id string) ([]codex.Model, error) {
	s.mu.Lock()
	s.calls = append(s.calls, id)
	s.mu.Unlock()
	if s.entered != nil {
		s.entered <- ctx
		select {
		case <-s.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]codex.Model(nil), s.models...), s.err
}

func (s *openAIAutoSource) set(models []codex.Model, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.models, s.err = models, err
}

func (s *openAIAutoSource) count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.calls)
}

func openAIAutoFixture(t *testing.T) (*Service, *sql.DB, *openAIAutoSource) {
	t.Helper()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "openai-auto.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b"} {
		if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_state,auth_account_id)
			VALUES(?,'builtin_openai','OpenAI','cred-'||?,1,'now','now','subscription_login','authenticated','remote-'||?)`, id, id, id); err != nil {
			t.Fatal(err)
		}
	}
	source := &openAIAutoSource{models: []codex.Model{{ID: "gpt-test", Name: "GPT Test"}}}
	s := New(db, account.NewService(db, nil), provider.NewService(db), source)
	return s, db, source
}

func awaitAuto(t *testing.T, results <-chan error) error {
	t.Helper()
	select {
	case err := <-results:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("OpenAI 后台同步未完成")
		return nil
	}
}

func TestOpenAIAutoSyncFillsStaleCatalogOncePerVersion(t *testing.T) {
	s, _, source := openAIAutoFixture(t)
	results := make(chan error, 4)
	auto := StartOpenAIAutoSync(context.Background(), s, nil, func(err error) { results <- err })
	t.Cleanup(auto.Close)
	if err := awaitAuto(t, results); err != nil {
		t.Fatal(err)
	}
	if source.count() != 2 {
		t.Fatal("启动未同步两个过期账号", source.calls)
	}
	items, err := s.List(context.Background())
	if err != nil || len(items) != 2 {
		t.Fatal("同步后目录不完整", len(items), err)
	}
	auto.Request()
	if err := awaitAuto(t, results); err != nil {
		t.Fatal(err)
	}
	if source.count() != 2 {
		t.Fatal("当前版本已完整仍重复同步", source.count())
	}
	if status := auto.Status(); status.Stale != 0 || status.Running || status.LastError != "" {
		t.Fatalf("unexpected status: %+v", status)
	}
}

func TestOpenAIAutoSyncIncludesAPIKeyWithoutSubscriptionState(t *testing.T) {
	s, db, source := openAIAutoFixture(t)
	s.accounts = account.NewService(db, apiKeyStore{})
	var requests atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Error("API Key 目录请求的端点或认证错误")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-test-key"}]}`))
	}))
	defer upstream.Close()
	if _, err := db.Exec(`UPDATE providers SET base_url=? WHERE id='builtin_openai';
		UPDATE accounts SET auth_method='api_key',auth_state='' WHERE id='b'`, upstream.URL+"/v1"); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 4)
	auto := StartOpenAIAutoSync(context.Background(), s, nil, func(err error) { results <- err })
	t.Cleanup(auto.Close)
	if err := awaitAuto(t, results); err != nil {
		t.Fatal(err)
	}
	items, err := s.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range items {
		found = found || item.AccountID == "b" && item.Model == "gpt-test-key"
	}
	if !found || requests.Load() != 1 || source.count() != 1 {
		t.Fatalf("启动没有分别同步 API Key 与订阅目录: key=%t requests=%d subscription=%d", found, requests.Load(), source.count())
	}
	auto.Request()
	if err := awaitAuto(t, results); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 || auto.Status().Stale != 0 {
		t.Fatal("已同步的 API Key 目录被重复读取")
	}
	if _, err := db.Exec(`UPDATE accounts SET credential_ref='key-v2' WHERE id='b'`); err != nil {
		t.Fatal(err)
	}
	auto.Request()
	if err := awaitAuto(t, results); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 2 || source.count() != 1 {
		t.Fatal("更换 API Key 后没有重新同步独立目录")
	}
}

func TestOpenAIAutoSyncEmptyCatalogIsTerminalPerVersion(t *testing.T) {
	s, _, source := openAIAutoFixture(t)
	source.set(nil, nil)
	results := make(chan error, 4)
	auto := StartOpenAIAutoSync(context.Background(), s, nil, func(err error) { results <- err })
	t.Cleanup(auto.Close)
	if err := awaitAuto(t, results); err != nil {
		t.Fatal(err)
	}
	if source.count() != 2 {
		t.Fatal("空目录账号未被同步", source.count())
	}
	auto.Request()
	if err := awaitAuto(t, results); err != nil {
		t.Fatal(err)
	}
	if source.count() != 2 {
		t.Fatal("空目录在同一版本内被重复同步", source.count())
	}
	if status := auto.Status(); status.Stale != 2 {
		t.Fatalf("stale count = %d, want 2", status.Stale)
	}
}

func TestOpenAIAutoSyncFailurePreservesRowsAndReportsFixedCode(t *testing.T) {
	s, db, source := openAIAutoFixture(t)
	results := make(chan error, 4)
	auto := StartOpenAIAutoSync(context.Background(), s, nil, func(err error) { results <- err })
	t.Cleanup(auto.Close)
	if err := awaitAuto(t, results); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE accounts SET credential_ref='cred-a-v2' WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	source.set(nil, errors.New("private-upstream-detail"))
	auto.Request()
	if err := awaitAuto(t, results); !errors.Is(err, ErrUnavailable) {
		t.Fatal("未报告脱敏错误", err)
	}
	var rows int
	if err := db.QueryRow(`SELECT COUNT(*) FROM account_models WHERE account_id='a'`).Scan(&rows); err != nil || rows != 1 {
		t.Fatal("失败删除了旧目录", rows, err)
	}
	if status := auto.Status(); status.LastError != "openai_auto_sync_failed" {
		t.Fatalf("status = %+v", status)
	}
	if _, err := db.Exec(`UPDATE accounts SET credential_ref='cred-a-v3' WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	source.set([]codex.Model{{ID: "gpt-test", Name: "GPT Test"}}, nil)
	auto.Request()
	if err := awaitAuto(t, results); err != nil {
		t.Fatal(err)
	}
	items, err := s.List(context.Background())
	if err != nil || len(items) != 2 {
		t.Fatal("新版本恢复失败", len(items), err)
	}
}

func TestOpenAIAutoSyncSkipsDisabledLoggedOutAndOtherChannels(t *testing.T) {
	s, db, source := openAIAutoFixture(t)
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id='a';
		INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at,auth_method,auth_state)
		VALUES('out','builtin_openai','Out','cred','now','now','subscription_login','unknown'),
		('key','builtin_google','Key','cred','now','now','api_key','authenticated'),
		('google','builtin_google','Google','cred','now','now','subscription_login','authenticated')`); err != nil {
		t.Fatal(err)
	}
	results := make(chan error, 4)
	auto := StartOpenAIAutoSync(context.Background(), s, nil, func(err error) { results <- err })
	t.Cleanup(auto.Close)
	if err := awaitAuto(t, results); err != nil {
		t.Fatal(err)
	}
	if source.count() != 1 || source.calls[0] != "b" {
		t.Fatal("扫描了禁用、登出或其他提供商账号", source.calls)
	}
	if _, err := db.Exec(`UPDATE providers SET enabled=0 WHERE id='builtin_openai'`); err != nil {
		t.Fatal(err)
	}
	auto.Request()
	if err := awaitAuto(t, results); err != nil {
		t.Fatal(err)
	}
	if source.count() != 1 {
		t.Fatal("禁用提供商仍访问上游", source.calls)
	}
}

func TestOpenAIAutoSyncStatusAndCancel(t *testing.T) {
	s, _, source := openAIAutoFixture(t)
	source.entered = make(chan context.Context, 2)
	source.release = make(chan struct{})
	results := make(chan error, 4)
	auto := StartOpenAIAutoSync(context.Background(), s, nil, func(err error) { results <- err })
	ctx := awaitGoogleSync(t, source.entered)
	if !auto.Status().Running {
		t.Fatal("同步中状态未暴露")
	}
	if deadline, limited := ctx.Deadline(); !limited || time.Until(deadline) > autoSyncTimeout {
		t.Fatal("每账号请求没有独立超时")
	}
	close(source.release)
	if err := awaitGoogleSync(t, results); err != nil {
		t.Fatal(err)
	}
	if auto.Status().Running {
		t.Fatal("同步结束后仍标记运行中")
	}
	auto.Close()
}
