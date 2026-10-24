package main

import (
	"context"
	"path/filepath"
	"testing"

	"omniproxy/internal/infrastructure/database"
)

func TestEmbeddedMigrationsCreateProviderAndAccountTables(t *testing.T) {
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "test.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	files, err := embeddedMigrations()
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(context.Background(), db, files); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"providers", "accounts"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Errorf("embedded migrations did not create %s", table)
		}
	}
	var builtins int
	if err := db.QueryRow(`SELECT COUNT(*) FROM providers WHERE source = 'builtin' AND enabled = 1`).Scan(&builtins); err != nil {
		t.Fatal(err)
	}
	if builtins != 12 {
		t.Fatalf("embedded migrations created %d enabled providers, want twelve", builtins)
	}
}
