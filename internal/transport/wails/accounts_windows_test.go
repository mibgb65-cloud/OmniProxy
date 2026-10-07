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
	keys := []string{"batch-one-test-key", "batch-two-test-key"}
	batch, err := svc.CreateAccounts(account.BatchCreateInput{ProviderID: "builtin_google", APIKeys: keys})
	if err != nil || len(batch) != 2 {
		t.Fatalf("CreateAccounts count = %d, error = %v", len(batch), err)
	}
	for i, item := range batch {
		if item.Name != keys[i][:8] {
			t.Fatal("batch binding did not automatically name the account")
		}
		if err := db.QueryRow(`SELECT credential_ref FROM accounts WHERE id = ?`, item.ID).Scan(&ref); err != nil {
			t.Fatal(err)
		}
		stored, err := store.Get(ref)
		if err != nil || string(stored) != keys[i] {
			t.Fatal("batch credential did not survive protected storage")
		}
		clear(stored)
		preview, err := svc.AccountKeyPreview(item.ID)
		if err != nil || preview == keys[i] {
			t.Fatal("preview binding leaked or failed to read the protected key")
		}
		copied := ""
		svc.copyText = func(value string) bool { copied = value; return true }
		if err := svc.CopyAccountKey(item.ID); err != nil || copied != keys[i] {
			t.Fatal("copy binding did not use the protected key")
		}
		svc.copyText = func(string) bool { return false }
		if err := svc.CopyAccountKey(item.ID); err == nil {
			t.Fatal("clipboard failure was ignored")
		}
		if err := svc.DeleteAccount(item.ID); err != nil {
			t.Fatal(err)
		}
	}
}
