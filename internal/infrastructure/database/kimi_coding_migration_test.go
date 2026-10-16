package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestKimiCodingMigrationPreservesLegacyCredentialsAndCatalogIdentity(t *testing.T) {
	ctx := context.Background()
	db, err := Open(Config{Path: filepath.Join(t.TempDir(), "old.db")})
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
		if f.Name() >= "028" {
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
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at)VALUES('legacy','builtin_kimi','Legacy','opaque-protected-ref',0,'now','now');INSERT INTO kimi_account_options(account_id,region,config_version)VALUES('legacy','global',3)`); err != nil {
		t.Fatal(err)
	}
	var oldVersion string
	if err := db.QueryRow(`SELECT version FROM account_catalog_versions WHERE id='legacy'`).Scan(&oldVersion); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version)VALUES('legacy','kimi-k3','K3','now',?)`, oldVersion); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	var ref, region, mode, version string
	var enabled bool
	var n int
	if err := db.QueryRow(`SELECT a.credential_ref,a.enabled,k.region,k.billing_mode,v.version FROM accounts a JOIN kimi_account_options k ON k.account_id=a.id JOIN account_catalog_versions v ON v.id=a.id WHERE a.id='legacy'`).Scan(&ref, &enabled, &region, &mode, &version); err != nil {
		t.Fatal(err)
	}
	if ref != "opaque-protected-ref" || enabled || region != "global" || mode != "payg" || version != oldVersion {
		t.Fatal("migration changed legacy identity")
	}
	if err := db.QueryRow(`SELECT count(*) FROM account_models WHERE account_id='legacy' AND account_version=?`, oldVersion).Scan(&n); err != nil || n != 1 {
		t.Fatal("migration invalidated unchanged API catalog")
	}
}
