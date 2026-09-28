package account

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"omniproxy/internal/infrastructure/database"
)

type fakeStore struct {
	secret  []byte
	deleted bool
	failDel bool
}

func (s *fakeStore) Put(secret []byte) (string, error) {
	s.secret = append([]byte(nil), secret...)
	return "cred-v1-test", nil
}

func (s *fakeStore) Delete(string) error {
	s.deleted = true
	if s.failDel {
		return errors.New("cleanup failed")
	}
	return nil
}

func testService(t *testing.T) (*Service, *fakeStore) {
	t.Helper()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "accounts.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	store := &fakeStore{}
	return NewService(db, store), store
}

func TestAccountLifecycleKeepsSecretOutOfMetadata(t *testing.T) {
	ctx := context.Background()
	service, store := testService(t)
	const secret = "sk-test-not-a-real-secret"
	item, err := service.Create(ctx, CreateInput{ProviderID: "builtin_openai", Name: "  主账号  ", APIKey: secret})
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "主账号" || !item.Enabled || item.ProviderID != "builtin_openai" {
		t.Fatalf("created account = %+v", item)
	}
	if string(store.secret) != secret {
		t.Fatal("credential store did not receive the key")
	}
	items, err := service.List(ctx)
	if err != nil || len(items) != 1 || items[0] != item {
		t.Fatalf("listed accounts = (%+v, %v)", items, err)
	}
	var ref string
	if err := service.db.QueryRowContext(ctx, `SELECT credential_ref FROM accounts WHERE id = ?`, item.ID).Scan(&ref); err != nil {
		t.Fatal(err)
	}
	if ref != "cred-v1-test" || strings.Contains(ref, secret) {
		t.Fatal("database stored a secret instead of its reference")
	}
	disabled, err := service.SetEnabled(ctx, item.ID, false)
	if err != nil || disabled.Enabled {
		t.Fatalf("disabled account = (%+v, %v)", disabled, err)
	}
	if err := service.Delete(ctx, item.ID); err != nil || !store.deleted {
		t.Fatalf("delete = %v, credential deleted = %v", err, store.deleted)
	}
	items, err = service.List(ctx)
	if err != nil || len(items) != 0 {
		t.Fatalf("accounts after delete = (%+v, %v)", items, err)
	}
}

func TestAccountValidationAndFailedInsertCleanup(t *testing.T) {
	ctx := context.Background()
	service, store := testService(t)
	for _, input := range []CreateInput{
		{ProviderID: "missing", Name: "one", APIKey: "key"},
		{ProviderID: "builtin_openai", Name: "", APIKey: "key"},
		{ProviderID: "builtin_openai", Name: "one", APIKey: "\n"},
		{ProviderID: "builtin_openai", Name: "one", APIKey: strings.Repeat("k", maxKeyBytes+1)},
	} {
		if _, err := service.Create(ctx, input); err == nil {
			t.Fatalf("Create(%q) accepted invalid input", input.ProviderID)
		}
	}
	if store.secret != nil {
		t.Fatal("invalid input reached credential store")
	}
	if _, err := service.db.ExecContext(ctx, `CREATE TRIGGER reject_account BEFORE INSERT ON accounts BEGIN SELECT RAISE(FAIL, 'blocked'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(ctx, CreateInput{ProviderID: "builtin_openai", Name: "one", APIKey: "secret"}); err == nil || !store.deleted {
		t.Fatalf("failed insert = %v, credential cleaned = %v", err, store.deleted)
	}
}

func TestDeleteReportsCredentialCleanupFailureAfterRemovingAccount(t *testing.T) {
	ctx := context.Background()
	service, store := testService(t)
	item, err := service.Create(ctx, CreateInput{ProviderID: "builtin_deepseek", Name: "one", APIKey: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	store.failDel = true
	if err := service.Delete(ctx, item.ID); err == nil {
		t.Fatal("expected credential cleanup error")
	}
	if _, _, err := service.get(ctx, item.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("account remains after cleanup failure: %v", err)
	}
}
