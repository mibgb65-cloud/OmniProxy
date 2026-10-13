package modelcatalog

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/infrastructure/codex"
	"omniproxy/internal/infrastructure/database"
)

type olderSubscription struct{ models []codex.Model }

func (s olderSubscription) Models(context.Context, string) ([]codex.Model, error) {
	return s.models, nil
}

func TestVerifiedNewModelSurvivesOlderOfficialCore(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "models.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_state,auth_account_id) VALUES
 ('a','builtin_openai','A','ref',1,'now','now','subscription_login','authenticated','workspace')`); err != nil {
		t.Fatal(err)
	}
	s := New(db, account.NewService(db, nil), provider.NewService(db), olderSubscription{})
	if err := s.ConfirmContext(ctx, "a", "gpt-6-sol", "ref:workspace", 900019); err == nil {
		t.Fatal("arbitrary missing model added")
	}
	if err := s.ConfirmContext(ctx, "a", "gpt-6.1-sol", "ref:workspace", 900019); err != nil {
		t.Fatal(err)
	}
	items, err := s.Sync(ctx, "a")
	if err != nil || len(items) != 1 || items[0].Model != "gpt-6.1-sol" || !items[0].Metadata.ContextVerified {
		t.Fatal("verified newer model removed by older core", err)
	}
	full := olderSubscription{models: make([]codex.Model, 1000)}
	for i := range full.models {
		full.models[i].ID = "model-" + strconv.Itoa(i)
	}
	s.subscription = full
	if _, err := s.Sync(ctx, "a"); err == nil {
		t.Fatal("supplement exceeded catalog bound")
	}
	items, err = s.List(ctx)
	if err != nil || len(items) != 1 || items[0].Model != "gpt-6.1-sol" {
		t.Fatal("failed oversized sync replaced the prior snapshot", err)
	}
	s.subscription = olderSubscription{}
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	if err := s.ConfirmContext(ctx, "a", "gpt-6.1-sol", "ref:workspace", 900019); err == nil {
		t.Fatal("disabled account accepted")
	}
	if _, err := db.Exec(`UPDATE accounts SET enabled=1,credential_ref='changed' WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	items, err = s.Sync(ctx, "a")
	if err != nil || len(items) != 0 {
		t.Fatal("new authorization inherited supplemental model", err)
	}
}
