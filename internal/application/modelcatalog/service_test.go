package modelcatalog

import (
	"context"
	"errors"
	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	core "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/infrastructure/codex"
	"omniproxy/internal/infrastructure/database"
	"os"
	"path/filepath"
	"testing"
)

type fakeSubscription struct {
	fail   bool
	change func()
}

func (s *fakeSubscription) Models(context.Context, string) ([]codex.Model, error) {
	if s.change != nil {
		s.change()
	}
	if s.fail {
		return nil, errors.New("upstream-private-data")
	}
	return []codex.Model{{ID: "gpt-test", Name: "GPT Test", Metadata: core.Metadata{Description: "Official model", IsDefault: true, DefaultEffort: "high", ReasoningEfforts: []core.Effort{{Value: "high"}}}}}, nil
}
func TestSnapshotPreservedOnFailureAndInvalidatedByCredentialsOnly(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "models.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_state,auth_account_id)VALUES('a','builtin_openai','A','protected-ref',1,'now','now','subscription_login','authenticated','workspace')`)
	if err != nil {
		t.Fatal(err)
	}
	upstream := &fakeSubscription{}
	s := New(db, account.NewService(db, nil), provider.NewService(db), upstream)
	models, err := s.Sync(ctx, "a")
	if err != nil || len(models) != 1 {
		t.Fatal("sync failed", err)
	}
	if models[0].Metadata.Description != "Official model" || len(models[0].Metadata.ReasoningEfforts) != 1 || !models[0].Metadata.IsDefault {
		t.Fatal("official metadata was not persisted")
	}
	upstream.fail = true
	if _, err = s.Sync(ctx, "a"); err == nil || err.Error() != "models: unavailable" {
		t.Fatal("failure leaked detail")
	}
	models, err = s.List(ctx)
	if err != nil || len(models) != 1 {
		t.Fatal("last snapshot lost")
	}
	db.Exec(`UPDATE accounts SET name='renamed',updated_at='later' WHERE id='a'`)
	models, err = s.List(ctx)
	if err != nil || len(models) != 1 {
		t.Fatal("metadata invalidated credentials")
	}
	db.Exec(`UPDATE accounts SET credential_ref='new-reference' WHERE id='a'`)
	models, err = s.List(ctx)
	if err != nil || len(models) != 0 {
		t.Fatal("stale credentials retained")
	}
	upstream.fail = false
	upstream.change = func() { db.Exec(`UPDATE accounts SET credential_ref='another-reference' WHERE id='a'`) }
	if _, err = s.Sync(ctx, "a"); err == nil {
		t.Fatal("concurrent credential replacement accepted")
	}
}
