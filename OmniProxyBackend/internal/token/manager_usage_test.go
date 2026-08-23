package token

import (
	"encoding/json"
	"omniproxy/internal/storage"
	"path/filepath"
	"testing"
	"time"
)

func TestManagerUpdatePreservesTokenValueWhenBlank(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(UpsertRequest{Name: "primary", Provider: ProviderOpenAI, TokenValue: "sk-primary-token"})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := manager.Update(item.ID, UpsertRequest{Name: "renamed", Provider: ProviderOpenAI, CredentialType: CredentialTypeAPIKey})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "renamed" {
		t.Fatalf("expected updated name, got %q", updated.Name)
	}
	if updated.TokenValue != "sk-primary-token" {
		t.Fatalf("expected token value to be preserved, got %q", updated.TokenValue)
	}
}

func TestManagerUpdateAllowsMimoTokenPlanRegionChangeWithoutTokenValue(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(UpsertRequest{
		Name:           "plan",
		Provider:       ProviderXiaomi,
		CredentialType: CredentialTypeMimoTokenPlan,
		Region:         MimoRegionCN,
		TokenValue:     "tp-xiaomi-token-plan",
	})
	if err != nil {
		t.Fatal(err)
	}

	updated, err := manager.Update(item.ID, UpsertRequest{
		Name:           "plan",
		Provider:       ProviderXiaomi,
		CredentialType: CredentialTypeMimoTokenPlan,
		Region:         MimoRegionAMS,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Region != MimoRegionAMS || updated.TokenValue != "tp-xiaomi-token-plan" {
		t.Fatalf("expected region update with preserved token, got %#v", updated)
	}
}

func TestManagerUpdateRequiresTokenValueWhenCredentialTypeChanges(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(UpsertRequest{Name: "primary", Provider: ProviderOpenAI, TokenValue: "sk-primary-token"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = manager.Update(item.ID, UpsertRequest{Name: "primary", Provider: ProviderOpenAI, CredentialType: CredentialTypeCodexAuthJSON})
	if err == nil {
		t.Fatal("expected credential type change without token value to fail")
	}
}

func TestManagerUsesCodexEmailAsName(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	authJSON := codexAuthJSONForTest(t, "coder@example.com")
	item, err := manager.Add(UpsertRequest{
		Name:           "",
		Provider:       ProviderOpenAI,
		CredentialType: CredentialTypeCodexAuthJSON,
		TokenValue:     authJSON,
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "coder@example.com" {
		t.Fatalf("expected email as name, got %q", item.Name)
	}
}

func TestManagerUsesFlatCodexEmailAsName(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	authJSON, err := json.Marshal(map[string]any{
		"type":         "codex",
		"email":        "flat@example.com",
		"access_token": "flat-access-token",
		"id_token":     codexJWTForTest(t, "flat@example.com", "flat-account"),
	})
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(UpsertRequest{
		Name:           "",
		Provider:       ProviderOpenAI,
		CredentialType: CredentialTypeCodexAuthJSON,
		TokenValue:     string(authJSON),
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "flat@example.com" {
		t.Fatalf("expected flat email as name, got %q", item.Name)
	}
}

func TestManagerAllowsCodexAuthSameEmailDifferentAccountIDs(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	firstAuth := codexAuthJSONForTestWithAccount(t, "coder@example.com", "account-a")
	secondAuth := codexAuthJSONForTestWithAccount(t, "coder@example.com", "account-b")

	if _, err := manager.Add(UpsertRequest{Provider: ProviderOpenAI, CredentialType: CredentialTypeCodexAuthJSON, TokenValue: firstAuth}); err != nil {
		t.Fatal(err)
	}
	second, err := manager.Add(UpsertRequest{Provider: ProviderOpenAI, CredentialType: CredentialTypeCodexAuthJSON, TokenValue: secondAuth})
	if err != nil {
		t.Fatal(err)
	}
	if second.Name != "coder@example.com" {
		t.Fatalf("expected email as shared display name, got %q", second.Name)
	}
	if items := manager.List(); len(items) != 2 {
		t.Fatalf("expected both Codex accounts to be stored, got %#v", items)
	}
	if _, err := manager.Add(UpsertRequest{Provider: ProviderOpenAI, CredentialType: CredentialTypeCodexAuthJSON, TokenValue: firstAuth}); err != ErrDuplicateName {
		t.Fatalf("expected same account_id to be treated as duplicate, got %v", err)
	}
}

func TestManagerUsesClaudeOAuthEmailAsName(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}

	item, err := manager.Add(UpsertRequest{
		Name:           "",
		Provider:       ProviderAnthropic,
		CredentialType: CredentialTypeClaudeOAuth,
		TokenValue:     `{"access_token":"claude-access-token","refresh_token":"claude-refresh-token","email":"claude@example.com","expired":"2026-05-01T08:09:54+08:00"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Name != "claude@example.com" {
		t.Fatalf("expected email as name, got %q", item.Name)
	}
}

func TestManagerRecordsProxyUsageTotalsAndDailyStats(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(UpsertRequest{Name: "metered", Provider: ProviderOpenAI, TokenValue: "sk-metered-token"})
	if err != nil {
		t.Fatal(err)
	}

	if err := manager.RecordProxyUsage(item.ID, TokenConsumption{InputTokens: 100, OutputTokens: 40, TotalTokens: 140, CacheCreationTokens: 7, CacheReadTokens: 80}); err != nil {
		t.Fatal(err)
	}
	if err := manager.RecordProxyUsage(item.ID, TokenConsumption{InputTokens: 10, OutputTokens: 5}); err != nil {
		t.Fatal(err)
	}

	updated, err := manager.Get(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Stats.RequestCount != 2 {
		t.Fatalf("expected 2 requests, got %d", updated.Stats.RequestCount)
	}
	if updated.Stats.TotalTokens != 155 {
		t.Fatalf("expected 155 total tokens, got %d", updated.Stats.TotalTokens)
	}
	if updated.Stats.InputTokens != 110 || updated.Stats.OutputTokens != 45 {
		t.Fatalf("unexpected input/output tokens: %#v", updated.Stats)
	}
	if updated.Stats.CacheCreationTokens != 7 || updated.Stats.CacheReadTokens != 80 {
		t.Fatalf("unexpected cache token stats: %#v", updated.Stats)
	}
	if len(updated.Stats.Daily) != 1 || updated.Stats.Daily[0].TotalTokens != 155 {
		t.Fatalf("unexpected daily stats: %#v", updated.Stats.Daily)
	}
	if updated.Stats.Daily[0].CacheCreationTokens != 7 || updated.Stats.Daily[0].CacheReadTokens != 80 {
		t.Fatalf("unexpected daily cache stats: %#v", updated.Stats.Daily)
	}
}

func TestManagerRecordsCodexWeeklyEstimateResetBaseline(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(UpsertRequest{
		Name:           "codex",
		Provider:       ProviderOpenAI,
		CredentialType: CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForTest(t, "codex@example.com"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.RecordProxyUsage(item.ID, TokenConsumption{InputTokens: 100, OutputTokens: 20, TotalTokens: 120, CacheReadTokens: 80}); err != nil {
		t.Fatal(err)
	}

	beforeReset, err := manager.Get(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	resetAt := time.Date(2026, 8, 23, 12, 0, 0, 0, time.UTC)
	if err := manager.RecordCodexWeeklyEstimateReset(item.ID, resetAt, beforeReset.Stats); err != nil {
		t.Fatal(err)
	}
	if err := manager.RecordProxyUsage(item.ID, TokenConsumption{InputTokens: 10, OutputTokens: 5, TotalTokens: 15}); err != nil {
		t.Fatal(err)
	}

	updated, err := manager.Get(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	baseline := updated.Stats.CodexWeeklyEstimateBaseline
	if updated.Stats.CodexWeeklyEstimateResetAt != resetAt.Unix() || baseline == nil {
		t.Fatalf("missing weekly estimate reset baseline: %#v", updated.Stats)
	}
	if baseline.InputTokens != 100 || baseline.OutputTokens != 20 || baseline.TotalTokens != 120 || baseline.CacheReadTokens != 80 {
		t.Fatalf("unexpected weekly estimate baseline: %#v", baseline)
	}
}

func TestManagerRecordsStreamingRequestAndConsumptionSeparately(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(UpsertRequest{Name: "streaming", Provider: ProviderOpenAI, TokenValue: "sk-streaming-token"})
	if err != nil {
		t.Fatal(err)
	}

	if err := manager.RecordProxyRequest(item.ID); err != nil {
		t.Fatal(err)
	}
	if err := manager.RecordProxyConsumption(item.ID, TokenConsumption{InputTokens: 100, OutputTokens: 5, TotalTokens: 105, CacheReadTokens: 80}); err != nil {
		t.Fatal(err)
	}
	if err := manager.RecordProxyConsumption(item.ID, TokenConsumption{InputTokens: 20, OutputTokens: 2, TotalTokens: 22}); err != nil {
		t.Fatal(err)
	}

	updated, err := manager.Get(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Stats.RequestCount != 1 || updated.Stats.TotalTokens != 127 {
		t.Fatalf("unexpected streaming stats: %#v", updated.Stats)
	}
	if len(updated.Stats.Daily) != 1 || updated.Stats.Daily[0].RequestCount != 1 || updated.Stats.Daily[0].TotalTokens != 127 {
		t.Fatalf("unexpected streaming daily stats: %#v", updated.Stats.Daily)
	}
	if updated.Stats.LastTotalTokens != 22 || updated.Stats.CacheReadTokens != 80 {
		t.Fatalf("unexpected streaming token details: %#v", updated.Stats)
	}
}

func TestManagerRecordsBalanceUsageStatus(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(UpsertRequest{Name: "deepseek", Provider: ProviderDeepSeek, TokenValue: "sk-deepseek-token"})
	if err != nil {
		t.Fatal(err)
	}

	if err := manager.RecordUsageInfo(item.ID, UsageInfo{
		Source:           ProviderDeepSeek,
		BalanceRemaining: 0,
		BalanceUnit:      "CNY",
		Message:          "DeepSeek balance",
	}); err != nil {
		t.Fatal(err)
	}
	updated, err := manager.Get(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != StatusExhausted || updated.Remaining != 0 {
		t.Fatalf("expected zero balance to exhaust token, got status=%s remaining=%d", updated.Status, updated.Remaining)
	}

	if err := manager.RecordUsageInfo(item.ID, UsageInfo{
		Source:           ProviderDeepSeek,
		BalanceRemaining: 12.5,
		BalanceUnit:      "CNY",
		Message:          "DeepSeek balance",
	}); err != nil {
		t.Fatal(err)
	}
	updated, err = manager.Get(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != StatusActive || updated.Remaining != 100 {
		t.Fatalf("expected positive balance to restore active token, got status=%s remaining=%d", updated.Status, updated.Remaining)
	}
}

func TestManagerRecordsSubscriptionUsageFallsBackToSecondaryWindow(t *testing.T) {
	manager, err := NewManager(storage.NewJSONStore[[]Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(UpsertRequest{
		Name:           "free-codex",
		Provider:       ProviderOpenAI,
		CredentialType: CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForTest(t, "free@example.com"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := manager.RecordUsageInfo(item.ID, UsageInfo{
		Source:                     "codex",
		PlanType:                   "free",
		SecondaryUsedPercent:       18,
		SecondaryRemainingPercent:  82,
		SecondaryResetAt:           1777798105,
		SubscriptionQuotaAvailable: true,
	}); err != nil {
		t.Fatal(err)
	}

	updated, err := manager.Get(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Remaining != 82 || updated.Status != StatusActive {
		t.Fatalf("expected secondary quota to keep token active, got status=%s remaining=%d usage=%#v", updated.Status, updated.Remaining, updated.Usage)
	}
}

func TestManagerBatchesUsagePersistenceUntilFlush(t *testing.T) {
	store := &countingTokenStore{}
	manager, err := NewManager(store, 15)
	if err != nil {
		t.Fatal(err)
	}
	manager.persistDelay = time.Hour

	item, err := manager.Add(UpsertRequest{Name: "metered", Provider: ProviderOpenAI, TokenValue: "sk-metered-token"})
	if err != nil {
		t.Fatal(err)
	}
	if store.saves != 1 {
		t.Fatalf("expected add to persist immediately, got %d saves", store.saves)
	}

	if err := manager.RecordProxyUsage(item.ID, TokenConsumption{TotalTokens: 10}); err != nil {
		t.Fatal(err)
	}
	if store.saves != 1 {
		t.Fatalf("expected usage update to be deferred, got %d saves", store.saves)
	}

	if err := manager.Flush(); err != nil {
		t.Fatal(err)
	}
	if store.saves != 2 {
		t.Fatalf("expected flush to persist deferred usage, got %d saves", store.saves)
	}
	if store.tokens[0].Stats.TotalTokens != 10 {
		t.Fatalf("expected flushed stats, got %#v", store.tokens[0].Stats)
	}
}

// List and Get return shallow Token copies, so their Stats.Daily slices share a
// backing array with the manager. Recording usage must not write into it.
func TestRecordDailyStatsLeavesPublishedSlicesAlone(t *testing.T) {
	day := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)

	published := recordDailyStats(nil, day, TokenConsumption{TotalTokens: 5, InputTokens: 2}, true, true)
	if len(published) != 1 {
		t.Fatalf("expected one day row, got %d", len(published))
	}

	updated := recordDailyStats(published, day, TokenConsumption{TotalTokens: 7, InputTokens: 3}, true, true)

	if published[0].RequestCount != 1 || published[0].TotalTokens != 5 || published[0].InputTokens != 2 {
		t.Fatalf("recording mutated an already-published row: %#v", published[0])
	}
	if updated[0].RequestCount != 2 || updated[0].TotalTokens != 12 || updated[0].InputTokens != 5 {
		t.Fatalf("unexpected accumulated row: %#v", updated[0])
	}
}

func TestRecordDailyStatsAppendsWithoutSharingStorage(t *testing.T) {
	first := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	second := first.AddDate(0, 0, 1)

	published := recordDailyStats(nil, first, TokenConsumption{TotalTokens: 5}, true, true)
	grown := recordDailyStats(published, second, TokenConsumption{TotalTokens: 9}, true, true)

	if len(published) != 1 {
		t.Fatalf("appending a new day changed the published slice: %#v", published)
	}
	if len(grown) != 2 || grown[1].TotalTokens != 9 {
		t.Fatalf("unexpected grown rows: %#v", grown)
	}
}
