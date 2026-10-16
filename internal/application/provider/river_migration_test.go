package provider

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"omniproxy/internal/infrastructure/database"
)

func TestRiverUpgradePreservesExistingStateAndRunsOnce(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "upgrade.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	files, err := os.ReadDir("../../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	before := fstest.MapFS{}
	for _, file := range files {
		if file.Name() == "029_builtin_river.sql" {
			continue
		}
		body, err := os.ReadFile("../../../migrations/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		before[file.Name()] = &fstest.MapFile{Data: body}
	}
	if err := database.Migrate(ctx, db, before); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE providers SET enabled=0 WHERE id='builtin_openai';
		INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES('old','builtin_openai','existing','opaque-reference','old','old')`); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	s := NewService(db)
	item, err := s.get(ctx, "builtin_river")
	if err != nil || !item.Enabled || item.Kind != "river" || item.Source != SourceBuiltIn {
		t.Fatal("missing preset", item, err)
	}
	if _, err := s.SetEnabled(ctx, "builtin_river", false); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"builtin_openai", "builtin_river"} {
		item, err := s.get(ctx, id)
		if err != nil || item.Enabled {
			t.Fatal("saved switch reset", id, err)
		}
	}
	var ref string
	if err := db.QueryRow(`SELECT credential_ref FROM accounts WHERE id='old'`).Scan(&ref); err != nil || ref != "opaque-reference" {
		t.Fatal("credential reference changed", err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM providers WHERE kind='river'`).Scan(&count); err != nil || count != 1 {
		t.Fatal("migration not idempotent", err)
	}
}
