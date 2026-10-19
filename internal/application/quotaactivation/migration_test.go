package quotaactivation

import (
	"context"
	"omniproxy/internal/infrastructure/database"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestObservationMigrationPreservesExistingClaims(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "upgrade.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	prior := fstest.MapFS{}
	entries, err := os.ReadDir("../../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() >= "034" {
			continue
		}
		body, e := os.ReadFile("../../../migrations/" + entry.Name())
		if e != nil {
			t.Fatal(e)
		}
		prior[entry.Name()] = &fstest.MapFile{Data: body}
	}
	if err = database.Migrate(ctx, db, prior); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES('qa','builtin_openai','Test','unused','now','now'); INSERT INTO codex_quota_activation_windows(account_id,window_kind,identity_version,observed_reset,attempted_cycle,attempted_at) VALUES('qa','five_hour',?,'2030-01-01T00:00:00Z','initial','2026-01-01T00:00:00Z')`, strings.Repeat("a", 64))
	if err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	var reset, cycle, attempted, observed, confirmed string
	if err = db.QueryRow(`SELECT observed_reset,attempted_cycle,attempted_at,observed_at,confirmed_reset FROM codex_quota_activation_windows WHERE account_id='qa'`).Scan(&reset, &cycle, &attempted, &observed, &confirmed); err != nil {
		t.Fatal(err)
	}
	if reset != "2030-01-01T00:00:00Z" || cycle != "initial" || attempted != "2026-01-01T00:00:00Z" || observed != "" || confirmed != "" {
		t.Fatal("migration changed registration or invented confirmation")
	}
}
