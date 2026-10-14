package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestKimiMigrationPreservesExistingGlobalDefaultAndExplicitChina(t *testing.T) {
	ctx := context.Background()
	db, err := Open(Config{Path: filepath.Join(t.TempDir(), "legacy.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	files, err := os.ReadDir("../../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	before := fstest.MapFS{}
	for _, f := range files {
		if f.Name() >= "023" {
			continue
		}
		body, err := os.ReadFile("../../../migrations/" + f.Name())
		if err != nil {
			t.Fatal(err)
		}
		before[f.Name()] = &fstest.MapFile{Data: body}
	}
	if err := Migrate(ctx, db, before); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ id, base string }{{"global", ""}, {"cn", "https://api.moonshot.cn/v1"}, {"unknown", "https://example.invalid/v1"}} {
		if _, err := db.Exec(`INSERT INTO providers(id,kind,name,base_url,created_at,updated_at)VALUES(?,'kimi','Kimi',?,'now','now')`, tc.id, tc.base); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at)VALUES(?,?,'legacy','protected-ref',0,'now','now')`, tc.id, tc.id); err != nil {
			t.Fatal(err)
		}
	}
	if err := Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"global", "cn"} {
		var region, ref string
		var enabled bool
		if err := db.QueryRow(`SELECT k.region,a.credential_ref,a.enabled FROM kimi_account_options k JOIN accounts a ON a.id=k.account_id WHERE a.id=?`, id).Scan(&region, &ref, &enabled); err != nil || region != id || ref != "protected-ref" || enabled {
			t.Fatalf("legacy account changed: %s %v", id, err)
		}
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM kimi_account_options WHERE account_id='unknown'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("unknown endpoint guessed")
	}
}
