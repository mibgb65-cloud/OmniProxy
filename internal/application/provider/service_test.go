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

func TestCatalogContainsTenProviderKinds(t *testing.T) {
	want := []string{"openai", "anthropic", "google", "deepseek", "zhipu", "kimi", "minimax", "mimo", "stepfun", "atria"}
	definitions := Catalog()
	if len(definitions) != len(want) {
		t.Fatalf("Catalog length = %d, want %d", len(definitions), len(want))
	}
	for i, definition := range definitions {
		subscription := definition.Kind == "openai" || definition.Kind == "google"
		if definition.Kind != want[i] || len(definition.AuthMethods) != map[bool]int{true: 2, false: 1}[subscription] || definition.AuthMethods[0] != AuthAPIKey {
			t.Errorf("definition %d = %+v", i, definition)
		}
		if subscription && (len(definition.PlannedAuthMethods) != 0 || definition.AuthMethods[1] != AuthSubscriptionLogin) {
			t.Errorf("provider subscription option = %+v", definition)
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

func TestBuiltInProvidersStartEnabledAndCanOnlyToggle(t *testing.T) {
	ctx := context.Background()
	service := testService(t)
	items, err := service.List(ctx)
	if err != nil || len(items) != 10 {
		t.Fatalf("List = (%v, %v), want ten built-ins", items, err)
	}
	for _, item := range items {
		if item.Source != SourceBuiltIn || !item.Enabled || item.ID != "builtin_"+item.Kind {
			t.Errorf("invalid built-in = %+v", item)
		}
	}
	updated, err := service.SetEnabled(ctx, "builtin_openai", false)
	if err != nil || updated.Enabled {
		t.Fatalf("SetEnabled = (%+v, %v), want disabled", updated, err)
	}
	if _, err := service.Update(ctx, "builtin_openai", UpdateInput{Name: "changed", BaseURL: "https://example.test"}); !errors.Is(err, ErrBuiltIn) {
		t.Fatalf("Update built-in = %v, want ErrBuiltIn", err)
	}
	if err := service.Delete(ctx, "builtin_openai"); !errors.Is(err, ErrBuiltIn) {
		t.Fatalf("Delete built-in = %v, want ErrBuiltIn", err)
	}
	stored, err := NewService(service.db).get(ctx, "builtin_openai")
	if err != nil || stored.Enabled || stored.Name != "OpenAI" || stored.BaseURL != "" {
		t.Fatalf("stored built-in = (%+v, %v)", stored, err)
	}
}

func TestCustomProviderLifecycle(t *testing.T) {
	ctx := context.Background()
	service := testService(t)
	item, err := service.Create(ctx, CreateInput{Name: " My endpoint ", BaseURL: "https://api.example.test/v1"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Kind != KindOpenAICompatible || item.Source != SourceCustom || !item.Enabled || item.Name != "My endpoint" {
		t.Fatalf("Create = %+v", item)
	}
	items, err := NewService(service.db).List(ctx)
	if err != nil || len(items) != 11 {
		t.Fatalf("persisted List = (%v, %v), want eleven", items, err)
	}
	updated, err := service.Update(ctx, item.ID, UpdateInput{Name: "Backup", BaseURL: "https://backup.example.test/v1"})
	if err != nil || updated.Name != "Backup" || !updated.Enabled {
		t.Fatalf("Update = (%+v, %v)", updated, err)
	}
	updated, err = service.SetEnabled(ctx, item.ID, false)
	if err != nil || updated.Enabled {
		t.Fatalf("SetEnabled custom = (%+v, %v)", updated, err)
	}
	if err := service.Delete(ctx, item.ID); err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(ctx, item.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("second Delete = %v, want ErrNotFound", err)
	}
	items, err = service.List(ctx)
	if err != nil || len(items) != 10 {
		t.Fatalf("List after Delete = (%v, %v)", items, err)
	}
}

func TestCustomProviderValidation(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	invalid := []CreateInput{
		{Name: " ", BaseURL: "https://example.test"},
		{Name: "bad\nname", BaseURL: "https://example.test"},
		{Name: strings.Repeat("x", maxNameRunes+1), BaseURL: "https://example.test"},
		{Name: "Provider", BaseURL: ""},
		{Name: "Provider", BaseURL: "http://remote.example/v1"},
		{Name: "Provider", BaseURL: "https://user:pass@example.com/v1"},
		{Name: "Provider", BaseURL: "https://example.com/v1?key=secret"},
		{Name: "Provider", BaseURL: "https://example.com/v1#fragment"},
		{Name: "Provider", BaseURL: "https://example.com:65536/v1"},
		{Name: "Provider", BaseURL: strings.Repeat("x", maxURLBytes+1)},
	}
	for i, input := range invalid {
		if _, err := service.Create(ctx, input); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("invalid case %d = %v, want ErrInvalidInput", i, err)
		}
	}
	if items, err := service.List(ctx); err != nil || len(items) != 10 {
		t.Fatalf("invalid inputs wrote rows: %v, %v", items, err)
	}
	if _, err := service.Create(ctx, CreateInput{Name: "Local proxy", BaseURL: "http://127.0.0.1:8080/v1"}); err != nil {
		t.Fatalf("loopback HTTP should be accepted: %v", err)
	}
	if _, err := service.SetEnabled(ctx, "../bad", false); !errors.Is(err, ErrInvalidID) {
		t.Fatalf("SetEnabled invalid ID = %v", err)
	}
}

func TestDeletePreservesAccountsAndCredentialReferences(t *testing.T) {
	service := testService(t)
	ctx := context.Background()
	item, err := service.Create(ctx, CreateInput{Name: "Endpoint", BaseURL: "https://example.test"})
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
