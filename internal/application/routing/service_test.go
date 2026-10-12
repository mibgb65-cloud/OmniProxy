package routing

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/infrastructure/database"
)

type unreadStore struct{}

func (unreadStore) Put([]byte) (string, error) { panic("preview must not write keys") }
func (unreadStore) Get(string) ([]byte, error) { panic("preview must not read keys") }
func (unreadStore) Delete(string) error        { panic("preview must not delete keys") }

func fixture(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "routes.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at) VALUES('acct_old','builtin_deepseek','Old account','unused',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z'),('acct_new','builtin_deepseek','New account','unused',1,'2026-01-02T00:00:00Z','2026-01-02T00:00:00Z'),('acct_other','builtin_openai','Other account','unused',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	return New(db, provider.NewService(db), account.NewService(db, unreadStore{})), db
}

func input(alias string) Input {
	return Input{Alias: alias, ProviderID: "builtin_deepseek", Model: "deepseek-flash", Enabled: true}
}

func TestPersistenceUniqueAliasAndCRUD(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	rule, err := s.Create(ctx, input("coding-fast"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(ctx, input("coding-fast")); !errors.Is(err, ErrDuplicateAlias) {
		t.Fatalf("duplicate not rejected: %v", err)
	}
	if _, err := s.Create(ctx, input("Coding-fast")); err != nil {
		t.Fatal("case-sensitive alias rejected", err)
	}
	changed := input("coding-pro")
	changed.Model = "deepseek-v4-pro"
	changed.AccountID = "acct_new"
	updated, err := s.Update(ctx, rule.ID, changed)
	if err != nil || updated.ID != rule.ID || updated.CreatedAt != rule.CreatedAt {
		t.Fatalf("update: %+v %v", updated, err)
	}
	s = New(db, provider.NewService(db), account.NewService(db, unreadStore{}))
	preview, err := s.Resolve(ctx, "coding-pro")
	if err != nil || !preview.Routable || preview.AccountID != "acct_new" || preview.Model != "deepseek-v4-pro" || preview.CandidateCount != 1 {
		t.Fatalf("persisted preview: %+v %v", preview, err)
	}
	if _, err := s.SetEnabled(ctx, rule.ID, false); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "coding-pro")
	if err != nil || preview.Matched || preview.Routable {
		t.Fatal("disabled alias remained usable")
	}
	if err := s.Delete(ctx, rule.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, rule.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing rule not reported")
	}
}

func TestTargetValidationAndDeletedTargetDiagnostics(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	bad := input("bad")
	bad.AccountID = "acct_other"
	if _, err := s.Create(ctx, bad); err == nil {
		t.Fatal("cross-provider account accepted")
	}
	bad = input("bad")
	bad.ProviderID = "builtin_openai"
	if _, err := s.Create(ctx, bad); err == nil {
		t.Fatal("unsupported adapter accepted")
	}
	bad = input("bad")
	bad.Model = "display-name"
	if _, err := s.Create(ctx, bad); err == nil {
		t.Fatal("unknown target model accepted")
	}
	good := input("pinned")
	good.AccountID = "acct_new"
	rule, err := s.Create(ctx, good)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM accounts WHERE id='acct_new'`); err != nil {
		t.Fatal(err)
	}
	preview, err := s.Resolve(ctx, "pinned")
	if err != nil || !preview.Matched || preview.Routable || preview.ErrorCode != "account_missing" {
		t.Fatalf("deleted fixed account fallback: %+v %v", preview, err)
	}
	items, err := s.List(ctx)
	if err != nil || len(items) != 1 || items[0].ErrorCode != "account_missing" {
		t.Fatal("missing-target rule not retained")
	}
	if err := s.Delete(ctx, rule.ID); err != nil {
		t.Fatal(err)
	}
}

func TestNativeDirectOverrideAndDynamicAliases(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	preview, err := s.Resolve(ctx, "deepseek-flash")
	if err != nil || preview.Matched || !preview.Routable || preview.AccountID != "acct_old" {
		t.Fatalf("direct: %+v %v", preview, err)
	}
	value := input("deepseek-flash")
	value.Model = "deepseek-v4-pro"
	rule, err := s.Create(ctx, value)
	if err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "deepseek-flash")
	if err != nil || !preview.Matched || preview.Model != "deepseek-v4-pro" {
		t.Fatal("explicit rule did not override native model")
	}
	if _, err := db.Exec(`UPDATE providers SET enabled=0 WHERE id='builtin_deepseek'`); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "deepseek-flash")
	if err != nil || !preview.Matched || preview.Routable || preview.ErrorCode != "provider_disabled" {
		t.Fatal("unavailable matching rule silently fell back")
	}
	aliases, err := s.Aliases(ctx)
	if err != nil || len(aliases) != 0 {
		t.Fatal("invalid route advertised")
	}
	if _, err := s.SetEnabled(ctx, rule.ID, false); err != nil {
		t.Fatal(err)
	}
}
