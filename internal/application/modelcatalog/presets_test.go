package modelcatalog

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/atria"
)

func TestAtriaCatalogDoesNotReadKeysOrPretendToDiscoverModels(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "presets.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES('a','builtin_atria','Atria','protected','now','now')`); err != nil {
		t.Fatal(err)
	}
	s := New(db, account.NewService(db, nil), provider.NewService(db), nil)
	for _, change := range []string{`SELECT 1`, `UPDATE accounts SET credential_ref='new-ref',enabled=0 WHERE id='a'`} {
		if _, err := db.Exec(change); err != nil {
			t.Fatal(err)
		}
		items, err := s.List(ctx)
		if err != nil || len(items) != 1 {
			t.Fatal("preset list failed", err)
		}
		item := items[0]
		if item.Model != atria.ModelID || item.Source != "preset" || item.FetchedAt != "" || item.Metadata.ContextWindow == nil || *item.Metadata.ContextWindow != 256000 {
			t.Fatal("preset origin lost", item)
		}
	}
	if _, err := s.Sync(ctx, "a"); err == nil {
		t.Fatal("undocumented discovery invoked")
	}
	if _, err := db.Exec(`DELETE FROM accounts WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	if items, err := s.List(ctx); err != nil || len(items) != 0 {
		t.Fatal("deleted account remains in catalog", err)
	}
}
