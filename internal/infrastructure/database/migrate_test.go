package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func testMigrations() fstest.MapFS {
	return fstest.MapFS{
		"001_initial.sql": &fstest.MapFile{Data: []byte(`
			CREATE TABLE providers (id TEXT PRIMARY KEY, name TEXT NOT NULL);
			CREATE TABLE accounts (id TEXT PRIMARY KEY, name TEXT NOT NULL);
		`)},
		"002_usage.sql": &fstest.MapFile{Data: []byte(`
			CREATE TABLE usage (id TEXT PRIMARY KEY, tokens INTEGER NOT NULL);
		`)},
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(Config{Path: filepath.Join(t.TempDir(), "test.db")})
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestMigrateAppliesAllMigrations(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if err := Migrate(ctx, db, testMigrations()); err != nil {
		t.Fatalf("Migrate error: %v", err)
	}

	for _, table := range []string{"schema_version", "providers", "accounts", "usage"} {
		var name string
		err := db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Errorf("table %s missing: %v", table, err)
		}
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_version`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("schema_version rows = %d, want 2", count)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	if err := Migrate(ctx, db, testMigrations()); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	if err := Migrate(ctx, db, testMigrations()); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_version`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("schema_version rows = %d after re-run, want 2", count)
	}
}

func TestMigrateAppliesOnlyNewMigrations(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	first := fstest.MapFS{
		"001_initial.sql": &fstest.MapFile{Data: []byte(`CREATE TABLE providers (id TEXT PRIMARY KEY);`)},
	}
	if err := Migrate(ctx, db, first); err != nil {
		t.Fatalf("Migrate(first): %v", err)
	}

	extended := fstest.MapFS{
		"001_initial.sql": &fstest.MapFile{Data: []byte(`CREATE TABLE providers (id TEXT PRIMARY KEY);`)},
		"002_usage.sql":   &fstest.MapFile{Data: []byte(`CREATE TABLE usage (id TEXT PRIMARY KEY);`)},
	}
	if err := Migrate(ctx, db, extended); err != nil {
		t.Fatalf("Migrate(extended): %v", err)
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_version`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("schema_version rows = %d, want 2", count)
	}
}

func TestMigrateRollsBackFailedMigration(t *testing.T) {
	db := openTestDB(t)
	ctx := context.Background()

	broken := fstest.MapFS{
		"001_initial.sql": &fstest.MapFile{Data: []byte(`CREATE TABLE ok (id INTEGER);`)},
		"002_broken.sql":  &fstest.MapFile{Data: []byte(`THIS IS NOT SQL;`)},
	}
	if err := Migrate(ctx, db, broken); err == nil {
		t.Fatal("expected error from broken migration")
	}

	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_version`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("schema_version rows = %d, want 1 (only the successful migration)", count)
	}
}

func TestParseVersion(t *testing.T) {
	v, err := parseVersion("001_initial.sql")
	if err != nil || v != 1 {
		t.Fatalf("parseVersion(001_initial.sql) = %d, %v; want 1, nil", v, err)
	}
	if _, err := parseVersion("initial.sql"); err == nil {
		t.Fatal("expected error for missing version prefix")
	}
	if _, err := parseVersion("abc_name.sql"); err == nil {
		t.Fatal("expected error for non-numeric version")
	}
}
