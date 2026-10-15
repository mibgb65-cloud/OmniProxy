package database

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestMiMoMigrationPreservesPaygCredentialsAndOtherCatalogVersions(t *testing.T) {
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
		if f.Name() >= "025" {
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
	for _, tc := range []struct{ id, base string }{{"ordinary", ""}, {"explicit", "https://api.xiaomimimo.com/v1"}, {"loopback", "http://127.0.0.1:12345/v1"}, {"unknown", "https://example.invalid/v1"}} {
		if _, err := db.Exec(`INSERT INTO providers(id,kind,name,base_url,created_at,updated_at)VALUES(?,'mimo','MiMo',?,'now','now')`, tc.id, tc.base); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at)VALUES(?,?,'legacy','protected-ref',0,'now','now')`, tc.id, tc.id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at)VALUES('other','builtin_kimi','Kimi','other-protected-ref','now','now');INSERT INTO kimi_account_options(account_id,region)VALUES('other','cn')`); err != nil {
		t.Fatal(err)
	}
	var oldVersion string
	if err := db.QueryRow(`SELECT version FROM account_catalog_versions WHERE id='other'`).Scan(&oldVersion); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"ordinary", "explicit", "loopback", "unknown"} {
		var region, mode, ref, version string
		var enabled bool
		if err := db.QueryRow(`SELECT m.region,m.billing_mode,a.credential_ref,a.enabled,v.version FROM accounts a JOIN mimo_account_options m ON m.account_id=a.id JOIN account_catalog_versions v ON v.id=a.id WHERE a.id=?`, id).Scan(&region, &mode, &ref, &enabled, &version); err != nil {
			t.Fatal(err)
		}
		expected := "payg"
		if id == "unknown" {
			expected = ""
		}
		if region != "" || mode != expected || ref != "protected-ref" || enabled || !strings.HasSuffix(version, ":mimo:1") {
			t.Fatal("legacy account changed or guessed", id)
		}
	}
	var version string
	if err := db.QueryRow(`SELECT version FROM account_catalog_versions WHERE id='other'`).Scan(&version); err != nil || version != oldVersion {
		t.Fatal("other provider catalog version changed")
	}
}
