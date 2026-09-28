package provider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"omniproxy/internal/infrastructure/database"
)

func testService(t *testing.T) *Service {
	t.Helper()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "test.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	return NewService(db)
}

func TestCatalogContainsNineProviderKinds(t *testing.T) {
	want := []string{"openai", "anthropic", "google", "deepseek", "zhipu", "kimi", "minimax", "mimo", "stepfun"}
	definitions := Catalog()
	if len(definitions) != len(want) {
		t.Fatalf("Catalog length = %d, want %d", len(definitions), len(want))
	}
	for i, definition := range definitions {
		if definition.Kind != want[i] || len(definition.AuthMethods) != 1 || definition.AuthMethods[0] != AuthAPIKey {
			t.Errorf("definition %d = %+v", i, definition)
		}
		if i == 0 && (len(definition.PlannedAuthMethods) != 1 || definition.PlannedAuthMethods[0] != AuthSubscriptionLogin) {
			t.Errorf("OpenAI subscription option = %+v", definition)
		}
		if i > 0 && len(definition.PlannedAuthMethods) != 0 {
			t.Errorf("unexpected planned auth for %s", definition.Kind)
		}
	}
	definitions[0].AuthMethods[0] = "changed"
	if Catalog()[0].AuthMethods[0] != AuthAPIKey {
		t.Fatal("Catalog returned mutable shared data")
	}
}

func TestProviderConfigurationLifecycle(t *testing.T) {
	ctx := context.Background()
	service := testService(t)
	items, err := service.List(ctx)
	if err != nil || items == nil || len(items) != 0 {
		t.Fatalf("initial List = (%v, %v), want empty array", items, err)
	}
	first, err := service.Create(ctx, CreateInput{Kind: "mimo", Name: " MiMo 普通 API ", BaseURL: "https://api.example.test/v1", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.Create(ctx, CreateInput{Kind: "mimo", Name: "MiMo Token Plan", BaseURL: "https://plan.example.test/v1", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if first.ID == second.ID || first.Name != "MiMo 普通 API" {
		t.Fatalf("created configurations = %+v, %+v", first, second)
	}
	items, err = NewService(service.db).List(ctx)
	if err != nil || len(items) != 2 {
		t.Fatalf("persisted List = (%v, %v), want two", items, err)
	}
	updated, err := service.Update(ctx, first.ID, UpdateInput{Name: "普通 API", BaseURL: "", Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Kind != "mimo" || updated.Name != "普通 API" || updated.BaseURL != "" || updated.Enabled {
		t.Fatalf("Update = %+v", updated)
	}
	if err := service.Delete(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(ctx, first.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second Delete = %v, want ErrNotFound", err)
	}
	items, err = service.List(ctx)
	if err != nil || len(items) != 1 || items[0].ID != second.ID {
		t.Fatalf("List after Delete = (%v, %v)", items, err)
	}
}

func TestProviderValidation(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	invalid := []CreateInput{
		{Kind: "unknown", Name: "Provider", Enabled: true},
		{Kind: "deepseek", Name: " ", Enabled: true},
		{Kind: "deepseek", Name: "bad\nname", Enabled: true},
		{Kind: "deepseek", Name: strings.Repeat("x", maxNameRunes+1), Enabled: true},
		{Kind: "deepseek", Name: "Provider", BaseURL: "http://remote.example/v1", Enabled: true},
		{Kind: "deepseek", Name: "Provider", BaseURL: "https://user:pass@example.com/v1", Enabled: true},
		{Kind: "deepseek", Name: "Provider", BaseURL: "https://example.com/v1?key=secret", Enabled: true},
		{Kind: "deepseek", Name: "Provider", BaseURL: "https://example.com/v1#fragment", Enabled: true},
		{Kind: "deepseek", Name: "Provider", BaseURL: "https://example.com:65536/v1", Enabled: true},
		{Kind: "deepseek", Name: "Provider", BaseURL: strings.Repeat("x", maxURLBytes+1), Enabled: true},
	}
	for i, input := range invalid {
		if _, err := service.Create(ctx, input); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("invalid case %d = %v, want ErrInvalidInput", i, err)
		}
	}
	if items, err := service.List(ctx); err != nil || len(items) != 0 {
		t.Fatalf("invalid inputs wrote rows: %v, %v", items, err)
	}
	if _, err := service.Create(ctx, CreateInput{Kind: "deepseek", Name: "Local proxy", BaseURL: "http://127.0.0.1:8080/v1", Enabled: true}); err != nil {
		t.Fatalf("loopback HTTP should be accepted: %v", err)
	}
	if _, err := service.Update(ctx, "../bad", UpdateInput{Name: "x"}); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("Update invalid ID = %v", err)
	}
}

func TestDeletePreservesAccountsAndCredentialReferences(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	item, err := service.Create(ctx, CreateInput{Kind: "openai", Name: "OpenAI", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.db.Exec(`
		INSERT INTO accounts (id, provider_id, name, credential_ref, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)`, "account-1", item.ID, "test", "opaque-test-reference", item.CreatedAt, item.UpdatedAt); err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(ctx, item.ID); !errors.Is(err, ErrHasAccounts) {
		t.Fatalf("Delete with account = %v, want ErrHasAccounts", err)
	}
	var ref string
	if err := service.db.QueryRow(`SELECT credential_ref FROM accounts WHERE id = ?`, "account-1").Scan(&ref); err != nil {
		t.Fatalf("account was deleted: %v", err)
	}
	if ref != "opaque-test-reference" {
		t.Fatalf("credential_ref changed to %q", ref)
	}
}
