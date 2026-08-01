package token

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

func TestExtractCodexAuthFieldsSupportsCLIProxyAPIFlatJSON(t *testing.T) {
	idToken := codexJWTForTest(t, "flat@example.com", "account-from-jwt")
	raw, err := json.Marshal(map[string]any{
		"type":          "codex",
		"email":         "flat@example.com",
		"id_token":      idToken,
		"access_token":  "flat-access-token",
		"refresh_token": "flat-refresh-token",
		"account_id":    "flat-account",
	})
	if err != nil {
		t.Fatal(err)
	}

	fields, ok := ExtractCodexAuthFields(string(raw))
	if !ok {
		t.Fatal("expected flat codex auth JSON to parse")
	}
	if fields.Type != "codex" || fields.Email != "flat@example.com" {
		t.Fatalf("unexpected identity fields: %#v", fields)
	}
	if fields.IDToken != idToken || fields.AccessToken != "flat-access-token" || fields.RefreshToken != "flat-refresh-token" {
		t.Fatalf("unexpected token fields: %#v", fields)
	}
	if fields.AccountID != "flat-account" {
		t.Fatalf("unexpected account id: %q", fields.AccountID)
	}
	if fields.PlanType != "plus" {
		t.Fatalf("unexpected plan type: %q", fields.PlanType)
	}

	email, ok := ExtractCodexEmail(string(raw))
	if !ok || email != "flat@example.com" {
		t.Fatalf("unexpected extracted email: %q ok=%v", email, ok)
	}
}

func TestExtractCodexAuthFieldsFallsBackToFlatIDTokenClaims(t *testing.T) {
	idToken := codexJWTForTest(t, "claims@example.com", "account-from-jwt")
	raw, err := json.Marshal(map[string]any{
		"type":         "codex",
		"id_token":     idToken,
		"access_token": "flat-access-token",
	})
	if err != nil {
		t.Fatal(err)
	}

	fields, ok := ExtractCodexAuthFields(string(raw))
	if !ok {
		t.Fatal("expected flat codex auth JSON to parse")
	}
	if fields.Email != "claims@example.com" {
		t.Fatalf("expected email from id_token, got %q", fields.Email)
	}
	if fields.AccountID != "account-from-jwt" {
		t.Fatalf("expected account id from id_token, got %q", fields.AccountID)
	}
	if fields.PlanType != "plus" {
		t.Fatalf("expected plan type from id_token, got %q", fields.PlanType)
	}
}

func TestDisplayNameAddsCodexAccountID(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"type":       "codex",
		"email":      "coder@example.com",
		"account_id": "1234567890abcdef",
		"tokens": map[string]any{
			"access_token": "codex-access-token",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	item := Token{
		Name:           "coder@example.com",
		Provider:       ProviderOpenAI,
		CredentialType: CredentialTypeCodexAuthJSON,
		TokenValue:     string(raw),
	}
	if got := DisplayName(item); got != "coder@example.com (account_id: 1234567890abcdef)" {
		t.Fatalf("unexpected display name: %q", got)
	}
}

func TestDisplayNameShortensLongCodexAccountID(t *testing.T) {
	raw, err := json.Marshal(map[string]any{
		"type":       "codex",
		"email":      "coder@example.com",
		"account_id": "1234567890abcdef1234567890abcdef",
		"tokens": map[string]any{
			"access_token": "codex-access-token",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	item := Token{
		Name:           "coder@example.com",
		Provider:       ProviderOpenAI,
		CredentialType: CredentialTypeCodexAuthJSON,
		TokenValue:     string(raw),
	}
	if got := DisplayName(item); got != "coder@example.com (account_id: 12345678...abcdef)" {
		t.Fatalf("unexpected display name: %q", got)
	}
}

func codexJWTForTest(t *testing.T, email string, accountID string) string {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"email": email,
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": accountID,
			"chatgpt_plan_type":  "plus",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}
