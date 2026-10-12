package account

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestSubscriptionMetadataAndReferenceValidation(t *testing.T) {
	s, store := testService(t)
	ctx := context.Background()
	meta := SubscriptionMetadata{Email: "example@example.test", Plan: "plus", State: "authenticated", AccountID: "11111111-2222-3333-4444-555555555555"}
	for _, profile := range []string{"", "../../outside", strings.Repeat("z", 32), strings.Repeat("a", 33)} {
		if _, err := s.CreateSubscription(ctx, "builtin_openai", "", profile, meta); !errors.Is(err, ErrInvalidInput) {
			t.Fatal("invalid profile accepted")
		}
	}
	if _, err := s.CreateSubscription(ctx, "builtin_deepseek", "", strings.Repeat("a", 32), meta); !errors.Is(err, ErrInvalidInput) {
		t.Fatal("other provider accepted")
	}
	for _, invalid := range []SubscriptionMetadata{{Email: "bad\nidentity", State: "authenticated"}, {Plan: strings.Repeat("p", 81), State: "authenticated"}, {State: "invalid"}, {AccountID: strings.Repeat("a", 129), State: "authenticated"}, {AccountID: "unsafe/id", State: "authenticated"}, {AccountID: "bad\nidentity", State: "authenticated"}} {
		if validateSubscription(invalid) == nil {
			t.Fatal("invalid identity metadata accepted")
		}
	}
	if store.secret != nil {
		t.Fatal("invalid input reached protected store")
	}
	item, err := s.CreateSubscription(ctx, "builtin_openai", "", strings.Repeat("a", 32), meta)
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != meta.Email || item.AuthEmail != meta.Email || item.AuthPlan != "plus" || item.AuthAccountID != meta.AccountID {
		t.Fatal("identity metadata not saved")
	}
	var locator managedLocator
	if json.Unmarshal(store.secret, &locator) != nil || locator.Kind != "codex-keyring" || strings.Contains(string(store.secret), meta.Email) {
		t.Fatal("protected reference contains identity or invalid locator")
	}
	if _, err := s.Update(ctx, UpdateInput{ID: item.ID, Name: "订阅账号", APIKey: "replacement"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatal("subscription accepted API key replacement")
	}
	updated, err := s.Update(ctx, UpdateInput{ID: item.ID, Name: "订阅账号"})
	if err != nil || updated.Account.Name != "订阅账号" || updated.Account.AuthEmail != meta.Email || updated.Account.AuthAccountID != meta.AccountID {
		t.Fatal("metadata edit damaged subscription")
	}
}

func TestAPIKeyAccountsKeepCompatibleAuthentication(t *testing.T) {
	s, _ := testService(t)
	ctx := context.Background()
	items, err := s.CreateBatch(ctx, BatchCreateInput{ProviderID: "builtin_openai", APIKeys: []string{"test-key"}})
	if err != nil || len(items) != 1 || items[0].AuthMethod != AuthAPIKey {
		t.Fatal("batch lost API key authentication")
	}
	listed, err := s.List(ctx)
	if err != nil || len(listed) != 1 || listed[0] != items[0] {
		t.Fatal("batch and persisted metadata differ")
	}
	if err := s.UseAPIKey(ctx, items[0].ID, func(key []byte) error {
		if string(key) != "test-key" {
			t.Fatal("API key changed")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
