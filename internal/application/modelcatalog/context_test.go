package modelcatalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	core "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/infrastructure/codex"
	"omniproxy/internal/infrastructure/database"
)

type contextSubscription struct{}

func (contextSubscription) Models(context.Context, string) ([]codex.Model, error) {
	return []codex.Model{{ID: "gpt-6.1-sol", Name: "GPT-6.1-Sol", Metadata: core.Metadata{Description: "official"}}}, nil
}

func TestContextConfirmationIsScopedAndSurvivesModelSync(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "models.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_state,auth_account_id) VALUES
 ('a','builtin_openai','A','ref-a',1,'now','now','subscription_login','authenticated','workspace'),
 ('b','builtin_openai','B','ref-b',1,'now','now','subscription_login','authenticated','workspace')`)
	if err != nil {
		t.Fatal(err)
	}
	s := New(db, account.NewService(db, nil), provider.NewService(db), contextSubscription{})
	for _, id := range []string{"a", "b"} {
		if _, err := s.Sync(ctx, id); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		version, model string
		tokens         int64
	}{
		{"ref-a:workspace", "gpt-6.1-sol", 899999},
		{"stale", "gpt-6.1-sol", 900019},
		{"ref-a:workspace", "unknown", 900019},
	} {
		if s.ConfirmContext(ctx, "a", test.model, test.version, test.tokens) == nil {
			t.Fatal("invalid or unverified context accepted")
		}
	}
	if err := s.ConfirmContext(ctx, "a", "gpt-6.1-sol", "ref-a:workspace", 900019); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Sync(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	items, err := s.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.AccountID == "a" && (!item.Metadata.ContextVerified || item.Metadata.ContextWindow == nil || *item.Metadata.ContextWindow != 1000000) {
			t.Fatal("confirmed capacity lost after sync")
		}
		if item.AccountID == "b" && (item.Metadata.ContextVerified || item.Metadata.ContextWindow != nil) {
			t.Fatal("confirmation leaked to another account")
		}
	}
	if _, err := db.Exec(`UPDATE accounts SET credential_ref='new-ref' WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Sync(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	items, err = s.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.AccountID == "a" && item.Metadata.ContextVerified {
			t.Fatal("old authorization retained confirmed capacity")
		}
	}
}
