package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"omniproxy/internal/config"
	"omniproxy/internal/token"
	"testing"
)

func TestValidatorQueriesOpenRouterKeyUsage(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/key" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer openrouter-api-key" {
			t.Fatalf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"usage":12.5,"limit":50,"limit_remaining":37.5}}`))
	}))
	defer upstream.Close()

	validator, err := NewValidator(config.Config{
		OpenRouterBaseURL: upstream.URL + "/api/v1",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := validator.Validate(context.Background(), token.Token{
		Provider:       token.ProviderOpenRouter,
		CredentialType: token.CredentialTypeAPIKey,
		TokenValue:     "openrouter-api-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Usage == nil || result.Remaining == nil {
		t.Fatalf("expected openrouter validation usage, got %#v", result)
	}
	if *result.Remaining != 75 {
		t.Fatalf("expected 75 percent remaining, got %d", *result.Remaining)
	}
	if result.Usage.BalanceRemaining != 37.5 || result.Usage.BalanceTotal != 50 || result.Usage.BalanceUsed != 12.5 || result.Usage.BalanceUnit != "USD" {
		t.Fatalf("unexpected openrouter usage: %#v", result.Usage)
	}
}

func TestValidatorQueriesOpenRouterUnlimitedKeyUsage(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/key" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"usage":0.017,"limit":null,"limit_remaining":null,"is_free_tier":true}}`))
	}))
	defer upstream.Close()

	validator, err := NewValidator(config.Config{
		OpenRouterBaseURL: upstream.URL + "/api/v1",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := validator.Validate(context.Background(), token.Token{
		Provider:       token.ProviderOpenRouter,
		CredentialType: token.CredentialTypeAPIKey,
		TokenValue:     "openrouter-api-key",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK || result.Usage == nil || result.Remaining == nil {
		t.Fatalf("expected openrouter validation usage, got %#v", result)
	}
	if *result.Remaining != 100 {
		t.Fatalf("expected unlimited key to remain active, got %d", *result.Remaining)
	}
	if !result.Usage.BalanceUnlimited || result.Usage.BalanceUnit != "USD" || result.Usage.BalanceUsed != 0.017 {
		t.Fatalf("unexpected unlimited openrouter usage: %#v", result.Usage)
	}
	if result.Usage.PlanType != "OpenRouter Free Tier" {
		t.Fatalf("unexpected plan type: %q", result.Usage.PlanType)
	}
}

func TestValidatorUsesTokenRouterRoutingRules(t *testing.T) {
	tests := []struct {
		name    string
		baseURL func(string) string
	}{
		{
			name:    "base without version",
			baseURL: func(serverURL string) string { return serverURL },
		},
		{
			name:    "base with version",
			baseURL: func(serverURL string) string { return serverURL + "/v1" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/routing-rules" {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer tr_test_token" {
					t.Fatalf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"rules":[]}`))
			}))
			defer upstream.Close()

			validator, err := NewValidator(config.Config{
				TokenRouterBaseURL: tt.baseURL(upstream.URL),
			})
			if err != nil {
				t.Fatal(err)
			}

			result, err := validator.Validate(context.Background(), token.Token{
				Provider:       token.ProviderTokenRouter,
				CredentialType: token.CredentialTypeAPIKey,
				TokenValue:     "tr_test_token",
			})
			if err != nil {
				t.Fatal(err)
			}
			if !result.OK {
				t.Fatalf("expected tokenrouter validation to pass: %#v", result)
			}
		})
	}
}

func TestValidatorUsesPremModelsEndpoint(t *testing.T) {
	var gotPath string
	var authorization string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		authorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer upstream.Close()

	validator, err := NewValidator(config.Config{
		PremBaseURL: upstream.URL + "/v1",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := validator.Validate(context.Background(), token.Token{
		Provider:       token.ProviderPrem,
		CredentialType: token.CredentialTypeAPIKey,
		BaseURL:        upstream.URL + "/v1",
		TokenValue:     "prem-api-key-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("expected validation ok, got %#v", result)
	}
	if gotPath != "/openai/v1/models" || authorization != "Bearer prem-api-key-token" {
		t.Fatalf("unexpected Prem validation path=%q authorization=%q", gotPath, authorization)
	}
}

func TestValidatorUsesZoModelsAvailable(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models/available" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer zo_sk_test_token" {
			t.Fatalf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"models":[]}`))
	}))
	defer upstream.Close()

	validator, err := NewValidator(config.Config{
		ZoBaseURL: upstream.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := validator.Validate(context.Background(), token.Token{
		Provider:       token.ProviderZo,
		CredentialType: token.CredentialTypeAPIKey,
		TokenValue:     "zo_sk_test_token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("expected zo validation to pass: %#v", result)
	}
}

func TestValidatorUsesSub2APIUsageEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		baseURL func(string) string
	}{
		{
			name:    "base without version",
			baseURL: func(serverURL string) string { return serverURL },
		},
		{
			name:    "base with version",
			baseURL: func(serverURL string) string { return serverURL + "/v1" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/usage" {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer sub2api-api-key-token" {
					t.Fatalf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"mode": "quota_limited",
					"isValid": true,
					"status": "active",
					"quota": {"limit": 100, "used": 12.5, "remaining": 87.5, "unit": "USD"},
					"usage": {
						"today": {"requests": 2, "total_tokens": 10, "actual_cost": 0.12},
						"total": {"requests": 9, "total_tokens": 80, "actual_cost": 1.25}
					},
					"model_stats": []
				}`))
			}))
			defer upstream.Close()

			validator, err := NewValidator(config.Config{
				Sub2APIBaseURL: tt.baseURL(upstream.URL),
			})
			if err != nil {
				t.Fatal(err)
			}

			result, err := validator.Validate(context.Background(), token.Token{
				Provider:       token.ProviderSub2API,
				CredentialType: token.CredentialTypeAPIKey,
				BaseURL:        tt.baseURL(upstream.URL),
				TokenValue:     "sub2api-api-key-token",
			})
			if err != nil {
				t.Fatal(err)
			}
			if !result.OK {
				t.Fatalf("expected sub2api validation to pass: %#v", result)
			}
			if result.Usage == nil {
				t.Fatal("expected sub2api usage details")
			}
			if result.Usage.Source != token.ProviderSub2API || result.Usage.BalanceUnit != "USD" {
				t.Fatalf("unexpected sub2api usage source/unit: %#v", result.Usage)
			}
			if result.Usage.BalanceTotal != 100 || result.Usage.BalanceUsed != 12.5 || result.Usage.BalanceRemaining != 87.5 {
				t.Fatalf("unexpected sub2api quota usage: %#v", result.Usage)
			}
			if result.Remaining == nil || *result.Remaining != 88 {
				t.Fatalf("expected result remaining 88, got %#v", result.Remaining)
			}
		})
	}
}

func TestValidatorUsesNewAPIUsageEndpoint(t *testing.T) {
	tests := []struct {
		name    string
		baseURL func(string) string
	}{
		{
			name:    "base without version",
			baseURL: func(serverURL string) string { return serverURL },
		},
		{
			name:    "base with version",
			baseURL: func(serverURL string) string { return serverURL + "/v1" },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/usage/token/" {
					t.Fatalf("unexpected path: %s", r.URL.Path)
				}
				if r.Header.Get("Authorization") != "Bearer newapi-api-key-token" {
					t.Fatalf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{
					"code": true,
					"message": "ok",
					"data": {
						"object": "token_usage",
						"name": "work",
						"total_granted": 1000,
						"total_used": 250,
						"total_available": 750,
						"unlimited_quota": false
					}
				}`))
			}))
			defer upstream.Close()

			validator, err := NewValidator(config.Config{
				NewAPIBaseURL: tt.baseURL(upstream.URL),
			})
			if err != nil {
				t.Fatal(err)
			}

			result, err := validator.Validate(context.Background(), token.Token{
				Provider:       token.ProviderNewAPI,
				CredentialType: token.CredentialTypeAPIKey,
				BaseURL:        tt.baseURL(upstream.URL),
				TokenValue:     "newapi-api-key-token",
			})
			if err != nil {
				t.Fatal(err)
			}
			if !result.OK {
				t.Fatalf("expected new-api validation to pass: %#v", result)
			}
			if result.Usage == nil {
				t.Fatal("expected new-api usage details")
			}
			if result.Usage.Source != token.ProviderNewAPI || result.Usage.BalanceUnit != "Quota" {
				t.Fatalf("unexpected new-api usage source/unit: %#v", result.Usage)
			}
			if result.Usage.BalanceTotal != 1000 || result.Usage.BalanceUsed != 250 || result.Usage.BalanceRemaining != 750 {
				t.Fatalf("unexpected new-api quota usage: %#v", result.Usage)
			}
			if result.Remaining == nil || *result.Remaining != 75 {
				t.Fatalf("expected result remaining 75, got %#v", result.Remaining)
			}
		})
	}
}

func TestValidatorUsesAtriaModelsEndpoint(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer atr_test_token" {
			t.Fatalf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"object":"list","data":[{"id":"Atria-Dawn-Preview"}]}`))
	}))
	defer upstream.Close()

	validator, err := NewValidator(config.Config{
		AtriaBaseURL: upstream.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := validator.Validate(context.Background(), token.Token{
		Provider:       token.ProviderAtria,
		CredentialType: token.CredentialTypeAPIKey,
		TokenValue:     "atr_test_token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.OK {
		t.Fatalf("expected atria validation to pass: %#v", result)
	}
	if result.CheckedPath == "" || result.Status != http.StatusOK {
		t.Fatalf("expected atria validation to hit /v1/models, got %#v", result)
	}
}
