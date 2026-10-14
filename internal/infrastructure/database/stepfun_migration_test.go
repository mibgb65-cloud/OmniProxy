package database

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestStepMigrationPreservesLegacyRegionAndUnknownEndpoint(t *testing.T) {
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
	old := fstest.MapFS{}
	for _, file := range files {
		if file.Name() >= "021" {
			continue
		}
		body, err := os.ReadFile("../../../migrations/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		old[file.Name()] = &fstest.MapFile{Data: body}
	}
	if err := Migrate(ctx, db, old); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ id, url string }{{"cn", ""}, {"global", "https://api.stepfun.ai/v1"}, {"plan", "https://api.stepfun.com/step_plan/v1"}, {"unknown", "https://example.invalid/v1"}} {
		if _, err := db.Exec(`INSERT INTO providers(id,kind,name,base_url,created_at,updated_at)VALUES(?,'stepfun','Step',?,'now','now')`, tc.id, tc.url); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at)VALUES(?,?,'legacy','protected-ref',0,'now','now')`, tc.id, tc.id); err != nil {
			t.Fatal(err)
		}
	}
	if err := Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ id, region, mode string }{{"cn", "cn", "payg"}, {"global", "global", "payg"}, {"plan", "cn", "step_plan"}} {
		var region, mode, ref string
		var enabled bool
		if err := db.QueryRow(`SELECT s.region,s.billing_mode,a.credential_ref,a.enabled FROM stepfun_account_options s JOIN accounts a ON a.id=s.account_id WHERE a.id=?`, tc.id).Scan(&region, &mode, &ref, &enabled); err != nil {
			t.Fatal(err)
		}
		if region != tc.region || mode != tc.mode || ref != "protected-ref" || enabled {
			t.Fatal("legacy identity or endpoint changed")
		}
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM stepfun_account_options WHERE account_id='unknown'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("unknown URL was guessed")
	}
}
