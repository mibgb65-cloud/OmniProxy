package modelcatalog

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/antigravity"
)

type googleAutoSource func(context.Context, string) ([]antigravity.Model, string, error)

func (f googleAutoSource) Models(ctx context.Context, id string) ([]antigravity.Model, string, error) {
	return f(ctx, id)
}

func googleAutoFixture(t *testing.T) *Service {
	t.Helper()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "google-auto.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "z"} {
		if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_state,auth_account_id)
			VALUES(?,'builtin_google','Google','protected-ref',1,'now','now','subscription_login','authenticated',?)`, id, id); err != nil {
			t.Fatal(err)
		}
	}
	return New(db, account.NewService(db, nil), provider.NewService(db), nil)
}

func googleSnapshot(s *Service, ctx context.Context, id, model string) ([]antigravity.Model, string, error) {
	var version string
	err := s.db.QueryRowContext(ctx, `SELECT version FROM account_catalog_versions WHERE id=?`, id).Scan(&version)
	return []antigravity.Model{{ID: model, Name: model}}, version, err
}

func awaitGoogleSync[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("Google 后台同步未完成")
		var empty T
		return empty
	}
}

func TestGoogleAutoSyncStartupFailurePreservesSnapshotAndContinues(t *testing.T) {
	s := googleAutoFixture(t)
	var fail atomic.Bool
	s.SetGoogleSubscription(googleAutoSource(func(ctx context.Context, id string) ([]antigravity.Model, string, error) {
		if fail.Load() && id == "z" {
			return nil, "", errors.New("private-upstream-detail")
		}
		return googleSnapshot(s, ctx, id, "gemini-"+id)
	}))
	var refreshed atomic.Int32
	results := make(chan error, 4)
	auto := StartGoogleAutoSync(context.Background(), s, func(context.Context) error {
		refreshed.Add(1)
		return nil
	}, func(err error) { results <- err })
	t.Cleanup(auto.Close)
	if err := awaitGoogleSync(t, results); err != nil {
		t.Fatal(err)
	}
	items, err := s.List(context.Background())
	if err != nil || len(items) != 2 || refreshed.Load() != 1 {
		t.Fatal("启动未自动发现并刷新目录", err)
	}
	fail.Store(true)
	if _, err := s.db.Exec(`DELETE FROM account_models WHERE account_id='a'`); err != nil {
		t.Fatal(err)
	}
	auto.Request()
	if err := awaitGoogleSync(t, results); !errors.Is(err, ErrUnavailable) || err.Error() != ErrUnavailable.Error() {
		t.Fatal("未报告脱敏错误", err)
	}
	items, err = s.List(context.Background())
	if err != nil || len(items) != 2 || refreshed.Load() != 2 {
		t.Fatal("失败丢失旧快照或阻断其他账号", err)
	}
}

func TestGoogleAutoSyncSkipsDisabledAndOtherChannels(t *testing.T) {
	s := googleAutoFixture(t)
	if _, err := s.db.Exec(`UPDATE accounts SET enabled=0 WHERE id='z'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at,auth_method,auth_state)
		VALUES('key','builtin_google','API Key','protected-ref','now','now','api_key','authenticated'),
		('other','builtin_openai','Other','protected-ref','now','now','subscription_login','authenticated'),
		('logged-out','builtin_google','Logged out','protected-ref','now','now','subscription_login','unknown')`); err != nil {
		t.Fatal(err)
	}
	calls := make(chan string, 8)
	s.SetGoogleSubscription(googleAutoSource(func(ctx context.Context, id string) ([]antigravity.Model, string, error) {
		calls <- id
		return googleSnapshot(s, ctx, id, "gemini-test")
	}))
	results := make(chan error, 4)
	auto := StartGoogleAutoSync(context.Background(), s, nil, func(err error) { results <- err })
	t.Cleanup(auto.Close)
	if err := awaitGoogleSync(t, results); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 1 || awaitGoogleSync(t, calls) != "a" {
		t.Fatal("扫描了禁用账号、其他提供商或 API Key 通道")
	}
	if _, err := s.db.Exec(`UPDATE providers SET enabled=0 WHERE id='builtin_google'`); err != nil {
		t.Fatal(err)
	}
	auto.Request()
	if err := awaitGoogleSync(t, results); err != nil || len(calls) != 0 {
		t.Fatal("禁用提供商仍访问上游", err)
	}
}
