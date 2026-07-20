package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"omniproxy/internal/config"
	"omniproxy/internal/token"
	"testing"
)

func TestValidatorParsesSub2APIWalletUsage(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/usage" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"mode": "unrestricted",
			"isValid": true,
			"planName": "Wallet",
			"remaining": 23.4,
			"unit": "USD",
			"balance": 23.4
		}`))
	}))
	defer upstream.Close()

	validator, err := NewValidator(config.Config{
		Sub2APIBaseURL: upstream.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := validator.Validate(context.Background(), token.Token{
		Provider:       token.ProviderSub2API,
		CredentialType: token.CredentialTypeAPIKey,
		BaseURL:        upstream.URL,
		TokenValue:     "sub2api-api-key-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Usage == nil || result.Usage.BalanceRemaining != 23.4 || result.Usage.BalanceUnit != "USD" {
		t.Fatalf("unexpected sub2api wallet usage: %#v", result.Usage)
	}
	if result.Remaining == nil || *result.Remaining != 100 {
		t.Fatalf("expected wallet balance to map to remaining 100, got %#v", result.Remaining)
	}
}

func TestValidatorParsesCodexUsage(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer codex-access-token" {
			t.Fatalf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("ChatGPT-Account-Id") != "account-123" {
			t.Fatalf("unexpected account header: %q", r.Header.Get("ChatGPT-Account-Id"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"plan_type": "team",
			"rate_limit": {
				"limit_reached": false,
				"primary_window": {"used_percent": 27, "reset_at": 1760000000},
				"secondary_window": {"used_percent": 41, "reset_at": 1760500000}
			}
		}`))
	}))
	defer upstream.Close()

	validator, err := NewValidator(config.Config{
		ProxyPort:          3000,
		ControlPort:        3890,
		OpenAIBaseURL:      "https://api.openai.com",
		CodexUsageEndpoint: upstream.URL,
		SwitchThreshold:    15,
		MaxRetries:         1,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := validator.Validate(t.Context(), token.Token{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue: `{
			"tokens": {
				"access_token": "codex-access-token",
				"account_id": "account-123",
				"id_token": "codex-id-token"
			}
		}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Usage == nil {
		t.Fatal("expected usage details")
	}
	if result.Usage.PlanType != "team" {
		t.Fatalf("expected team plan, got %q", result.Usage.PlanType)
	}
	if result.Usage.PrimaryRemainingPercent != 73 {
		t.Fatalf("expected 73 primary remaining, got %d", result.Usage.PrimaryRemainingPercent)
	}
	if result.Usage.SecondaryRemainingPercent != 59 {
		t.Fatalf("expected 59 secondary remaining, got %d", result.Usage.SecondaryRemainingPercent)
	}
	if result.Remaining == nil || *result.Remaining != 73 {
		t.Fatalf("expected result remaining 73, got %#v", result.Remaining)
	}
}

func TestValidatorParsesCodexUsageByWindowMinutes(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"plan_type": "team",
			"rate_limit": {
				"limit_reached": false,
				"primary_window": {"used_percent": 34, "reset_at": 1760500000, "window_minutes": 10080},
				"secondary_window": {"used_percent": 12, "reset_at": 1760000000, "window_minutes": 300}
			}
		}`))
	}))
	defer upstream.Close()

	validator, err := NewValidator(config.Config{
		ProxyPort:          3000,
		ControlPort:        3890,
		OpenAIBaseURL:      "https://api.openai.com",
		CodexUsageEndpoint: upstream.URL,
		SwitchThreshold:    15,
		MaxRetries:         1,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := validator.Validate(t.Context(), token.Token{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     `{"tokens":{"access_token":"codex-access-token"}}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Usage == nil {
		t.Fatal("expected usage details")
	}
	if result.Usage.PrimaryRemainingPercent != 88 || result.Usage.PrimaryResetAt != 1760000000 {
		t.Fatalf("expected 5h window to map to primary quota, got %#v", result.Usage)
	}
	if result.Usage.SecondaryRemainingPercent != 66 || result.Usage.SecondaryResetAt != 1760500000 {
		t.Fatalf("expected 7d window to map to secondary quota, got %#v", result.Usage)
	}
	if result.Remaining == nil || *result.Remaining != 88 {
		t.Fatalf("expected result remaining 88 from 5h quota, got %#v", result.Remaining)
	}
}

func TestParseCodexUsageMapsWindowsByLimitSeconds(t *testing.T) {
	tests := []struct {
		name                   string
		body                   string
		wantPrimaryRemaining   int
		wantSecondaryRemaining int
		wantEffectiveRemaining int
	}{
		{
			name: "weekly window returned in primary slot",
			body: `{
				"plan_type": "plus",
				"rate_limit": {
					"primary_window": {
						"used_percent": 45,
						"reset_at": 1785000000,
						"limit_window_seconds": 604800
					}
				}
			}`,
			wantSecondaryRemaining: 55,
			wantEffectiveRemaining: 55,
		},
		{
			name: "five hour window restored in primary slot",
			body: `{
				"plan_type": "plus",
				"rate_limit": {
					"primary_window": {
						"used_percent": 30,
						"reset_at": 1785000000,
						"limit_window_seconds": 18000
					}
				}
			}`,
			wantPrimaryRemaining:   70,
			wantEffectiveRemaining: 70,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage, ok := parseCodexUsage([]byte(tt.body))
			if !ok || !usage.SubscriptionQuotaAvailable {
				t.Fatalf("expected subscription usage, got %#v", usage)
			}
			if usage.PrimaryRemainingPercent != tt.wantPrimaryRemaining {
				t.Fatalf("primary remaining = %d, want %d", usage.PrimaryRemainingPercent, tt.wantPrimaryRemaining)
			}
			if usage.SecondaryRemainingPercent != tt.wantSecondaryRemaining {
				t.Fatalf("secondary remaining = %d, want %d", usage.SecondaryRemainingPercent, tt.wantSecondaryRemaining)
			}
			if got := usage.EffectiveRemainingPercent(); got != tt.wantEffectiveRemaining {
				t.Fatalf("effective remaining = %d, want %d", got, tt.wantEffectiveRemaining)
			}
		})
	}
}

func TestValidatorParsesCodexFreeUsageWithoutPrimaryWindow(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer codex-access-token" {
			t.Fatalf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"plan_type": "free",
			"rate_limit": {
				"limit_reached": false,
				"secondary_window": {"used_percent": 18, "reset_at": 1760500000}
			}
		}`))
	}))
	defer upstream.Close()

	validator, err := NewValidator(config.Config{
		ProxyPort:          3000,
		ControlPort:        3890,
		OpenAIBaseURL:      "https://api.openai.com",
		CodexUsageEndpoint: upstream.URL,
		SwitchThreshold:    15,
		MaxRetries:         1,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := validator.Validate(t.Context(), token.Token{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue: `{
			"tokens": {
				"access_token": "codex-access-token",
				"id_token": "codex-id-token"
			}
		}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Usage == nil {
		t.Fatal("expected usage details")
	}
	if result.Usage.PrimaryRemainingPercent != 0 {
		t.Fatalf("expected no primary quota window, got %#v", result.Usage)
	}
	if result.Usage.SecondaryRemainingPercent != 82 {
		t.Fatalf("expected 82 secondary remaining, got %d", result.Usage.SecondaryRemainingPercent)
	}
	if result.Remaining == nil || *result.Remaining != 82 {
		t.Fatalf("expected result remaining 82 from weekly quota, got %#v", result.Remaining)
	}
}

func TestValidatorMapsCodexFreePrimaryWindowToWeeklyQuota(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"plan_type": "free",
			"rate_limit": {
				"limit_reached": false,
				"primary_window": {"used_percent": 3, "reset_at": 1779000000}
			}
		}`))
	}))
	defer upstream.Close()

	validator, err := NewValidator(config.Config{
		ProxyPort:          3000,
		ControlPort:        3890,
		OpenAIBaseURL:      "https://api.openai.com",
		CodexUsageEndpoint: upstream.URL,
		SwitchThreshold:    15,
		MaxRetries:         1,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := validator.Validate(t.Context(), token.Token{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     `{"tokens":{"access_token":"codex-access-token"}}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Usage == nil {
		t.Fatal("expected usage details")
	}
	if result.Usage.PrimaryRemainingPercent != 0 || result.Usage.PrimaryResetAt != 0 {
		t.Fatalf("expected free plan to avoid 5h quota fields, got %#v", result.Usage)
	}
	if result.Usage.SecondaryRemainingPercent != 97 || result.Usage.SecondaryResetAt != 1779000000 {
		t.Fatalf("expected free primary window to map to weekly quota, got %#v", result.Usage)
	}
	if result.Remaining == nil || *result.Remaining != 97 {
		t.Fatalf("expected result remaining 97 from weekly quota, got %#v", result.Remaining)
	}
}
