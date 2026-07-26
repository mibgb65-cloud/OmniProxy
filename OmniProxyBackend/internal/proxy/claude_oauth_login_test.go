package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestClaudeOAuthAuthorizationURLUsesPKCEAndLoopbackRedirect(t *testing.T) {
	redirectURI := "http://localhost:43123/callback"
	raw := ClaudeOAuthAuthorizationURL(redirectURI, "pkce-challenge", "oauth-state")
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme+"://"+parsed.Host+parsed.Path != "https://claude.com/cai/oauth/authorize" {
		t.Fatalf("unexpected authorize endpoint: %s", parsed)
	}
	query := parsed.Query()
	if query.Get("client_id") != claudeOAuthClientID ||
		query.Get("response_type") != "code" ||
		query.Get("redirect_uri") != redirectURI ||
		query.Get("code_challenge") != "pkce-challenge" ||
		query.Get("code_challenge_method") != "S256" ||
		query.Get("state") != "oauth-state" ||
		query.Get("code") != "true" {
		t.Fatalf("unexpected authorize query: %#v", query)
	}
	for _, scope := range strings.Fields(claudeOAuthLoginScope) {
		if !strings.Contains(" "+query.Get("scope")+" ", " "+scope+" ") {
			t.Fatalf("missing OAuth scope %q in %q", scope, query.Get("scope"))
		}
	}
}

func TestExchangeClaudeAuthorizationCodeUsesJSONPayload(t *testing.T) {
	var payload map[string]any
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "claude-access",
			"refresh_token": "claude-refresh",
			"expires_in": 3600,
			"scope": "user:profile user:inference",
			"account": {"uuid": "account-1", "email_address": "claude@example.com"},
			"organization": {"uuid": "organization-1"}
		}`))
	}))
	defer upstream.Close()

	restore := replaceHTTPPostJSONForTest(func(ctx context.Context, client *http.Client, endpoint string, body any) (*http.Response, error) {
		if endpoint != "https://platform.claude.com/v1/oauth/token" {
			t.Fatalf("unexpected endpoint: %s", endpoint)
		}
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstream.URL, strings.NewReader(string(raw)))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		return upstream.Client().Do(req)
	})
	defer restore()

	result, err := (&Validator{client: upstream.Client()}).ExchangeClaudeAuthorizationCode(
		context.Background(),
		"authorization-code",
		"oauth-state",
		"pkce-verifier",
		"http://localhost:43123/callback",
	)
	if err != nil {
		t.Fatal(err)
	}
	if payload["grant_type"] != "authorization_code" ||
		payload["code"] != "authorization-code" ||
		payload["state"] != "oauth-state" ||
		payload["code_verifier"] != "pkce-verifier" ||
		payload["redirect_uri"] != "http://localhost:43123/callback" {
		t.Fatalf("unexpected token exchange payload: %#v", payload)
	}
	if result.Email != "claude@example.com" ||
		result.AccountID != "account-1" ||
		result.OrganizationID != "organization-1" ||
		result.AccessToken != "claude-access" ||
		result.RefreshToken != "claude-refresh" {
		t.Fatalf("unexpected OAuth result: %+v", result)
	}
}
