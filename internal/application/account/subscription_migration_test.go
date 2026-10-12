package account

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"omniproxy/internal/infrastructure/database"
)

func TestSubscriptionAccountIDMigrationPreservesExistingAccounts(t *testing.T) {
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "upgrade.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	ctx := context.Background()
	oldMigrations := fstest.MapFS{}
	for _, name := range []string{"001_initial.sql", "002_builtin_providers.sql", "003_model_routes.sql", "004_subscription_accounts.sql"} {
		data, err := os.ReadFile(filepath.Join("../../../migrations", name))
		if err != nil {
			t.Fatal(err)
		}
		oldMigrations[name] = &fstest.MapFile{Data: data}
	}
	if err := database.Migrate(ctx, db, oldMigrations); err != nil {
		t.Fatal(err)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_email,auth_plan,auth_state)
		VALUES('acct_previous','builtin_openai','旧账号','test-ref',1,'2026-10-08','2026-10-08','subscription_login','previous@example.test','team','authenticated')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	item, err := NewService(db, &fakeStore{}).Metadata(ctx, "acct_previous")
	if err != nil || item.AuthAccountID != "" || item.AuthPlan != "team" || item.AuthState != "authenticated" || item.AuthEmail != "previous@example.test" {
		t.Fatal("upgrade changed previous identity or authorization")
	}
	var ref string
	if err := db.QueryRowContext(ctx, `SELECT credential_ref FROM accounts WHERE id='acct_previous'`).Scan(&ref); err != nil || ref != "test-ref" {
		t.Fatal("upgrade changed the credential reference")
	}
}
