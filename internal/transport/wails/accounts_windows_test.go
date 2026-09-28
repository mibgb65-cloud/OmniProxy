//go:build windows

package wails

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/credential"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/platform/profile"
)

func TestAccountBindingsUseProtectedStore(t *testing.T) {
	prof := &profile.Profile{Kind: profile.Test, DataDir: t.TempDir()}
	db, err := database.Open(database.Config{Path: filepath.Join(prof.DataDir, "accounts.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	svc := &AppService{prof: prof, db: db}
	const secret = "test-only-key"
	item, err := svc.CreateAccount(account.CreateInput{ProviderID: "builtin_google", Name: "test", APIKey: secret})
	if err != nil {
		t.Fatal(err)
	}
	items, err := svc.ListAccounts()
	if err != nil || len(items) != 1 || items[0] != item {
		t.Fatalf("ListAccounts = (%+v, %v)", items, err)
	}
	var ref string
	if err := db.QueryRow(`SELECT credential_ref FROM accounts WHERE id = ?`, item.ID).Scan(&ref); err != nil {
		t.Fatal(err)
	}
	store, err := credential.New(prof)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(ref)
	if err != nil || string(got) != secret {
		t.Fatalf("stored key could not be read: %v", err)
	}
	clear(got)
	if err := svc.DeleteAccount(item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ref); !errors.Is(err, credential.ErrNotFound) {
		t.Fatalf("deleted credential still exists: %v", err)
	}
}
