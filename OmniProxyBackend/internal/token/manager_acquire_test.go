package token

import (
	"omniproxy/internal/storage"
	"path/filepath"
	"testing"
	"time"
)

func TestManagerAcquireBalancedRotatesAcrossAccounts(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	first, err := manager.Add(UpsertRequest{Name: "first", Provider: "openai", TokenValue: "sk-first-token"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := manager.Add(UpsertRequest{Name: "second", Provider: "openai", TokenValue: "sk-second-token"})
	if err != nil {
		t.Fatal(err)
	}

	sameCreatedAt := time.Unix(1000, 0)
	manager.mu.Lock()
	for i := range manager.tokens {
		manager.tokens[i].CreatedAt = sameCreatedAt
	}
	manager.mu.Unlock()

	selected, err := manager.AcquireBalancedMatching(ProviderOpenAI, CredentialTypeAPIKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != first.ID {
		t.Fatalf("expected older unused token first, got %s", selected.Name)
	}
	if err := manager.RecordProxyUsage(selected.ID, TokenConsumption{TotalTokens: 1}); err != nil {
		t.Fatal(err)
	}

	selected, err = manager.AcquireBalancedMatching(ProviderOpenAI, CredentialTypeAPIKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != second.ID {
		t.Fatalf("expected next unused token after one task, got %s", selected.Name)
	}
}

func TestManagerAcquireAvoidsInFlightQueueToken(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	backup, err := manager.Add(UpsertRequest{Name: "backup", Provider: "openai", TokenValue: "sk-backup-token"})
	if err != nil {
		t.Fatal(err)
	}
	primary, err := manager.Add(UpsertRequest{Name: "primary", Provider: "openai", TokenValue: "sk-primary-token"})
	if err != nil {
		t.Fatal(err)
	}

	selected, err := manager.Acquire(ProviderOpenAI, nil)
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != primary.ID {
		t.Fatalf("expected primary token first, got %s", selected.Name)
	}

	selected, err = manager.Acquire(ProviderOpenAI, nil)
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != backup.ID {
		t.Fatalf("expected busy primary to be skipped, got %s", selected.Name)
	}
}

func TestManagerAcquireBalancedPrefersHigherRemainingQuota(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	lower, err := manager.Add(UpsertRequest{Name: "lower", Provider: "openai", TokenValue: "sk-lower-token"})
	if err != nil {
		t.Fatal(err)
	}
	higher, err := manager.Add(UpsertRequest{Name: "higher", Provider: "openai", TokenValue: "sk-higher-token"})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.RecordUsage(lower.ID, 40); err != nil {
		t.Fatal(err)
	}
	if err := manager.RecordUsage(higher.ID, 80); err != nil {
		t.Fatal(err)
	}

	selected, err := manager.AcquireBalancedMatching(ProviderOpenAI, CredentialTypeAPIKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != higher.ID {
		t.Fatalf("expected higher remaining token, got %s", selected.Name)
	}
}

func TestManagerAcquireSkipsSelectedLowTokenWhenActiveTokenExists(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	active, err := manager.Add(UpsertRequest{Name: "active", Provider: ProviderOpenAI, TokenValue: "sk-active-token"})
	if err != nil {
		t.Fatal(err)
	}
	low, err := manager.Add(UpsertRequest{Name: "low-selected", Provider: ProviderOpenAI, TokenValue: "sk-low-token"})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.RecordUsage(low.ID, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.SetSelected(low.ID, true); err != nil {
		t.Fatal(err)
	}

	selected, err := manager.Acquire(ProviderOpenAI, nil)
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != active.ID {
		t.Fatalf("expected active token before selected low token, got %s", selected.Name)
	}
	manager.Release(selected.ID)

	if err := manager.RecordUsage(active.ID, 4); err != nil {
		t.Fatal(err)
	}
	selected, err = manager.Acquire(ProviderOpenAI, nil)
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != low.ID {
		t.Fatalf("expected selected low token when every token is low, got %s", selected.Name)
	}
}

func TestManagerAcquireBalancedSkipsSelectedLowTokenWhenActiveTokenExists(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	active, err := manager.Add(UpsertRequest{Name: "active", Provider: ProviderOpenAI, TokenValue: "sk-active-token"})
	if err != nil {
		t.Fatal(err)
	}
	low, err := manager.Add(UpsertRequest{Name: "low-selected", Provider: ProviderOpenAI, TokenValue: "sk-low-token"})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.RecordUsage(low.ID, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.SetSelected(low.ID, true); err != nil {
		t.Fatal(err)
	}

	selected, err := manager.AcquireBalancedMatching(ProviderOpenAI, CredentialTypeAPIKey, nil)
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != active.ID {
		t.Fatalf("expected active token before selected low token in balanced mode, got %s", selected.Name)
	}
}

func TestManagerAcquirePreferredSkipsPreferredLowTokenWhenActiveTokenExists(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	active, err := manager.Add(UpsertRequest{
		Name:           "paygo",
		Provider:       ProviderXiaomi,
		CredentialType: CredentialTypeAPIKey,
		TokenValue:     "sk-paygo-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	preferredLow, err := manager.Add(UpsertRequest{
		Name:           "token-plan",
		Provider:       ProviderXiaomi,
		CredentialType: CredentialTypeMimoTokenPlan,
		TokenValue:     "tp-token-plan-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.RecordUsage(preferredLow.ID, 5); err != nil {
		t.Fatal(err)
	}

	selected, err := manager.AcquirePreferredMatching(ProviderXiaomi, "", nil, func(item Token) bool {
		return item.CredentialType == CredentialTypeMimoTokenPlan
	})
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != active.ID {
		t.Fatalf("expected active non-preferred token before preferred low token, got %s", selected.Name)
	}
}

func TestManagerAcquireBalancedPreferredSkipsPreferredLowTokenWhenActiveTokenExists(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	active, err := manager.Add(UpsertRequest{Name: "api-key", Provider: ProviderOpenAI, TokenValue: "sk-api-key-token"})
	if err != nil {
		t.Fatal(err)
	}
	preferredLow, err := manager.Add(UpsertRequest{
		Name:           "codex-auth",
		Provider:       ProviderOpenAI,
		CredentialType: CredentialTypeCodexAuthJSON,
		TokenValue:     `{"type":"codex","email":"codex@example.com","access_token":"codex-access-token"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.RecordUsage(preferredLow.ID, 5); err != nil {
		t.Fatal(err)
	}

	selected, err := manager.AcquireBalancedPreferredMatching(ProviderOpenAI, "", nil, func(item Token) bool {
		return item.CredentialType == CredentialTypeCodexAuthJSON
	})
	if err != nil {
		t.Fatal(err)
	}
	if selected.ID != active.ID {
		t.Fatalf("expected active non-preferred token before preferred low token in balanced mode, got %s", selected.Name)
	}
}

func TestManagerAllowsSameNameAcrossProviders(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Add(UpsertRequest{Name: "work", Provider: ProviderOpenAI, TokenValue: "sk-openai-token"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(UpsertRequest{Name: "work", Provider: ProviderAnthropic, TokenValue: "sk-anthropic-token"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(UpsertRequest{Name: "work", Provider: ProviderKimi, TokenValue: "sk-kimi-token"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(UpsertRequest{Name: "work", Provider: ProviderZhipu, TokenValue: "zhipu-api-key-token"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(UpsertRequest{Name: "work", Provider: ProviderMiniMax, TokenValue: "minimax-api-key-token"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(UpsertRequest{Name: "work", Provider: ProviderGemini, TokenValue: "gemini-api-key-token"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(UpsertRequest{Name: "work", Provider: ProviderCustom, TokenValue: "custom-api-key-token"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(UpsertRequest{Name: "work", Provider: ProviderSub2API, BaseURL: "https://sub2api.example", TokenValue: "sub2api-api-key-token"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(UpsertRequest{Name: "work", Provider: ProviderNewAPI, BaseURL: "http://127.0.0.1:3000", TokenValue: "newapi-api-key-token"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(UpsertRequest{Name: "work", Provider: ProviderAnyRouter, BaseURL: "https://anyrouter.top", TokenValue: "anyrouter-api-key-token"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(UpsertRequest{Name: "work", Provider: ProviderPrem, TokenValue: "prem-api-key-token"}); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(UpsertRequest{Name: "work", Provider: ProviderAtria, TokenValue: "atr_test_token_value"}); err != nil {
		t.Fatal(err)
	}
}

func TestManagerRequiresSub2APIBaseURL(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Add(UpsertRequest{Name: "sub2api", Provider: ProviderSub2API, TokenValue: "sub2api-api-key-token"}); err == nil {
		t.Fatal("expected sub2api base url to be required")
	}
	item, err := manager.Add(UpsertRequest{Name: "sub2api", Provider: ProviderSub2API, BaseURL: "https://sub2api.example/v1/", TokenValue: "sub2api-api-key-token"})
	if err != nil {
		t.Fatal(err)
	}
	if item.BaseURL != "https://sub2api.example/v1" {
		t.Fatalf("expected normalized base url, got %q", item.BaseURL)
	}
	updated, err := manager.Update(item.ID, UpsertRequest{Name: "sub2api", Provider: ProviderSub2API, BaseURL: "https://other.example", TokenValue: ""})
	if err != nil {
		t.Fatal(err)
	}
	if updated.TokenValue != "sub2api-api-key-token" || updated.BaseURL != "https://other.example" {
		t.Fatalf("expected base url update without replacing key, got %#v", updated)
	}
}

func TestManagerRequiresNewAPIBaseURL(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Add(UpsertRequest{Name: "newapi", Provider: ProviderNewAPI, TokenValue: "newapi-api-key-token"}); err == nil {
		t.Fatal("expected new-api base url to be required")
	}
	item, err := manager.Add(UpsertRequest{Name: "newapi", Provider: ProviderNewAPI, BaseURL: "http://127.0.0.1:3000/v1/", TokenValue: "newapi-api-key-token"})
	if err != nil {
		t.Fatal(err)
	}
	if item.BaseURL != "http://127.0.0.1:3000/v1" {
		t.Fatalf("expected normalized base url, got %q", item.BaseURL)
	}
	updated, err := manager.Update(item.ID, UpsertRequest{Name: "newapi", Provider: ProviderNewAPI, BaseURL: "https://other.example", TokenValue: ""})
	if err != nil {
		t.Fatal(err)
	}
	if updated.TokenValue != "newapi-api-key-token" || updated.BaseURL != "https://other.example" {
		t.Fatalf("expected base url update without replacing key, got %#v", updated)
	}
}

func TestManagerRequiresAnyRouterBaseURL(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Add(UpsertRequest{Name: "anyrouter", Provider: ProviderAnyRouter, TokenValue: "anyrouter-api-key-token"}); err == nil {
		t.Fatal("expected anyrouter base url to be required")
	}
	item, err := manager.Add(UpsertRequest{Name: "anyrouter", Provider: ProviderAnyRouter, BaseURL: "https://anyrouter.top/v1/", TokenValue: "anyrouter-api-key-token"})
	if err != nil {
		t.Fatal(err)
	}
	if item.BaseURL != "https://anyrouter.top/v1" {
		t.Fatalf("expected normalized base url, got %q", item.BaseURL)
	}
	updated, err := manager.Update(item.ID, UpsertRequest{Name: "anyrouter", Provider: ProviderAnyRouter, BaseURL: "https://mirror.example", TokenValue: ""})
	if err != nil {
		t.Fatal(err)
	}
	if updated.TokenValue != "anyrouter-api-key-token" || updated.BaseURL != "https://mirror.example" {
		t.Fatalf("expected base url update without replacing key, got %#v", updated)
	}
}

func TestManagerAllowsPremAPIKeyWithoutBaseURL(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	item, err := manager.Add(UpsertRequest{Name: "prem", Provider: ProviderPrem, TokenValue: "prem-api-key-token"})
	if err != nil {
		t.Fatal(err)
	}
	if item.BaseURL != "" {
		t.Fatalf("expected prem account base url to stay empty, got %q", item.BaseURL)
	}
	updated, err := manager.Update(item.ID, UpsertRequest{Name: "prem", Provider: ProviderPrem, TokenValue: ""})
	if err != nil {
		t.Fatal(err)
	}
	if updated.TokenValue != "prem-api-key-token" || updated.BaseURL != "" {
		t.Fatalf("expected update without replacing key or setting base url, got %#v", updated)
	}
}

func TestManagerValidatesXiaomiCredentialFormats(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := manager.Add(UpsertRequest{Name: "paygo", Provider: ProviderXiaomi, TokenValue: "sk-xiaomi-paygo"}); err != nil {
		t.Fatal(err)
	}
	plan, err := manager.Add(UpsertRequest{
		Name:           "plan",
		Provider:       ProviderXiaomi,
		CredentialType: CredentialTypeMimoTokenPlan,
		TokenValue:     "tp-xiaomi-token-plan",
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Region != MimoRegionCN {
		t.Fatalf("expected default MiMo Token Plan region cn, got %q", plan.Region)
	}
	planSGP, err := manager.Add(UpsertRequest{
		Name:           "plan-sgp",
		Provider:       ProviderXiaomi,
		CredentialType: CredentialTypeMimoTokenPlan,
		Region:         MimoRegionSGP,
		TokenValue:     "tp-xiaomi-token-plan-sgp",
	})
	if err != nil {
		t.Fatal(err)
	}
	if planSGP.Region != MimoRegionSGP {
		t.Fatalf("expected MiMo Token Plan region sgp, got %q", planSGP.Region)
	}
	planAMS, err := manager.Add(UpsertRequest{
		Name:           "plan-ams",
		Provider:       ProviderXiaomi,
		CredentialType: CredentialTypeMimoTokenPlan,
		Region:         MimoRegionAMS,
		TokenValue:     "tp-xiaomi-token-plan-ams",
	})
	if err != nil {
		t.Fatal(err)
	}
	if planAMS.Region != MimoRegionAMS {
		t.Fatalf("expected MiMo Token Plan region ams, got %q", planAMS.Region)
	}
	if _, err := manager.Add(UpsertRequest{Name: "bad-paygo", Provider: ProviderXiaomi, TokenValue: "tp-wrong-kind"}); err == nil {
		t.Fatal("expected xiaomi pay-as-you-go key format error")
	}
	if _, err := manager.Add(UpsertRequest{
		Name:           "bad-plan",
		Provider:       ProviderXiaomi,
		CredentialType: CredentialTypeMimoTokenPlan,
		TokenValue:     "sk-wrong-kind",
	}); err == nil {
		t.Fatal("expected xiaomi token plan key format error")
	}
	if _, err := manager.Add(UpsertRequest{
		Name:           "bad-plan-region",
		Provider:       ProviderXiaomi,
		CredentialType: CredentialTypeMimoTokenPlan,
		Region:         "moon",
		TokenValue:     "tp-xiaomi-token-plan-region",
	}); err == nil {
		t.Fatal("expected xiaomi token plan region error")
	}
}

func TestManagerAllowsZhipuCodingPlanCredential(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(UpsertRequest{
		Name:           "glm-coding",
		Provider:       ProviderZhipu,
		CredentialType: CredentialTypeCodingPlan,
		TokenValue:     "zhipu-coding-plan-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.CredentialType != CredentialTypeCodingPlan {
		t.Fatalf("expected coding plan credential type, got %q", item.CredentialType)
	}
}
