package proxy

import (
	"net/http"
	"strings"
	"testing"

	"omniproxy/internal/token"
)

func TestApplyAuthUsesCodexAuthJSONAccessTokenAndAccountID(t *testing.T) {
	header := http.Header{}
	selected := token.Token{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue: `{
			"auth_mode": "chatgpt",
			"OPENAI_API_KEY": null,
			"tokens": {
				"access_token": "codex-access-token",
				"account_id": "account-123",
				"id_token": "codex-id-token"
			},
			"last_refresh": "2026-04-27T00:00:00Z"
		}`,
	}

	if err := applyAuth(header, selected); err != nil {
		t.Fatal(err)
	}
	if got := header.Get("Authorization"); got != "Bearer codex-access-token" {
		t.Fatalf("unexpected Authorization header: %q", got)
	}
	if got := header.Get("ChatGPT-Account-Id"); got != "account-123" {
		t.Fatalf("unexpected ChatGPT-Account-Id header: %q", got)
	}
}

func TestApplyAuthClearsIncomingCodexAccountID(t *testing.T) {
	header := http.Header{}
	header.Set("Authorization", "Bearer caller-token")
	header.Set("ChatGPT-Account-Id", "caller-account")
	selected := token.Token{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue: `{
			"auth_mode": "chatgpt",
			"tokens": {
				"access_token": "selected-access-token",
				"id_token": "selected-id-token"
			}
		}`,
	}

	if err := applyAuth(header, selected); err != nil {
		t.Fatal(err)
	}
	if got := header.Get("Authorization"); got != "Bearer selected-access-token" {
		t.Fatalf("unexpected Authorization header: %q", got)
	}
	if got := header.Get("ChatGPT-Account-Id"); got != "" {
		t.Fatalf("incoming ChatGPT-Account-Id should be cleared, got %q", got)
	}
}

func TestApplyAuthUsesTopLevelCodexAccountID(t *testing.T) {
	header := http.Header{}
	selected := token.Token{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue: `{
			"auth_mode": "chatgpt",
			"account_id": "top-level-account",
			"tokens": {
				"access_token": "selected-access-token",
				"id_token": "selected-id-token"
			}
		}`,
	}

	if err := applyAuth(header, selected); err != nil {
		t.Fatal(err)
	}
	if got := header.Get("ChatGPT-Account-Id"); got != "top-level-account" {
		t.Fatalf("unexpected ChatGPT-Account-Id header: %q", got)
	}
}

func TestApplyAuthUsesFlatCodexAuthJSON(t *testing.T) {
	header := http.Header{}
	selected := token.Token{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue: `{
			"type": "codex",
			"email": "coder@example.com",
			"access_token": "flat-access-token",
			"id_token": "flat-id-token",
			"account_id": "flat-account"
		}`,
	}

	if err := applyAuth(header, selected); err != nil {
		t.Fatal(err)
	}
	if got := header.Get("Authorization"); got != "Bearer flat-access-token" {
		t.Fatalf("unexpected Authorization header: %q", got)
	}
	if got := header.Get("ChatGPT-Account-Id"); got != "flat-account" {
		t.Fatalf("unexpected ChatGPT-Account-Id header: %q", got)
	}
}

func TestApplyAuthUsesGeminiAPIKeyHeader(t *testing.T) {
	header := http.Header{}
	header.Set("Authorization", "Bearer caller")
	header.Set("X-Goog-Api-Key", "caller-key")
	selected := token.Token{
		Provider:       token.ProviderGemini,
		CredentialType: token.CredentialTypeAPIKey,
		TokenValue:     "gemini-api-key-token",
	}

	if err := applyRouteAuth(header, selected, routeInfo{Protocol: "gemini"}); err != nil {
		t.Fatal(err)
	}
	if got := header.Get("Authorization"); got != "" {
		t.Fatalf("Authorization should be cleared, got %q", got)
	}
	if got := header.Get("x-goog-api-key"); got != "gemini-api-key-token" {
		t.Fatalf("unexpected x-goog-api-key header: %q", got)
	}
}

func TestApplyAuthUsesClaudeOAuthBearerHeaders(t *testing.T) {
	header := http.Header{}
	header.Set("x-api-key", "caller-key")
	selected := token.Token{
		Provider:       token.ProviderAnthropic,
		CredentialType: token.CredentialTypeClaudeOAuth,
		TokenValue:     `{"access_token":"claude-access-token","refresh_token":"claude-refresh-token"}`,
	}

	if err := applyRouteAuth(header, selected, routeInfo{Protocol: "anthropic"}); err != nil {
		t.Fatal(err)
	}
	if got := header.Get("Authorization"); got != "Bearer claude-access-token" {
		t.Fatalf("unexpected Authorization header: %q", got)
	}
	if got := header.Get("x-api-key"); got != "" {
		t.Fatalf("x-api-key should be cleared for OAuth, got %q", got)
	}
	if got := header.Get("anthropic-beta"); !strings.Contains(got, "oauth-2025-04-20") || !strings.Contains(got, "claude-code-20250219") {
		t.Fatalf("unexpected anthropic-beta header: %q", got)
	}
}

func TestApplyAuthUsesAnthropicHeadersForCompatibleProviders(t *testing.T) {
	for _, provider := range []string{token.ProviderZhipu, token.ProviderMiniMax, token.ProviderAnyRouter, token.ProviderCustom} {
		t.Run(provider, func(t *testing.T) {
			header := http.Header{}
			selected := token.Token{
				Provider:       provider,
				CredentialType: token.CredentialTypeAPIKey,
				TokenValue:     provider + "-api-key-token",
			}
			if err := applyRouteAuth(header, selected, routeInfo{Protocol: "anthropic"}); err != nil {
				t.Fatal(err)
			}
			if got := header.Get("x-api-key"); got != provider+"-api-key-token" {
				t.Fatalf("unexpected x-api-key header: %q", got)
			}
			if got := header.Get("anthropic-version"); got != "2023-06-01" {
				t.Fatalf("unexpected anthropic-version header: %q", got)
			}
		})
	}
}

func TestApplyAuthUsesFeatherlessBearerAndAttributionHeaders(t *testing.T) {
	header := http.Header{}
	selected := token.Token{
		Provider:       token.ProviderFeatherless,
		CredentialType: token.CredentialTypeAPIKey,
		TokenValue:     "featherless-api-key-token",
	}

	if err := applyRouteAuth(header, selected, routeInfo{Protocol: "openai"}); err != nil {
		t.Fatal(err)
	}
	if got := header.Get("Authorization"); got != "Bearer featherless-api-key-token" {
		t.Fatalf("unexpected Authorization header: %q", got)
	}
	if got := header.Get("HTTP-Referer"); got != "https://github.com/mibgb65-cloud/OmniProxy" {
		t.Fatalf("unexpected HTTP-Referer header: %q", got)
	}
	if got := header.Get("X-Title"); got != "OmniProxy" {
		t.Fatalf("unexpected X-Title header: %q", got)
	}
}
