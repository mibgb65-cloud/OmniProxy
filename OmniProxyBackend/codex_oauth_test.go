package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"omniproxy/internal/proxy"
	"omniproxy/internal/token"
)

func TestCodexOAuthAuthJSONPreservesIdentityAndRefreshToken(t *testing.T) {
	now := time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC)
	payload, err := json.Marshal(map[string]any{
		"https://api.openai.com/profile": map[string]string{"email": "browser-login@example.com"},
		"https://api.openai.com/auth":    map[string]string{"chatgpt_account_id": "account-browser-login"},
	})
	if err != nil {
		t.Fatal(err)
	}
	idToken := "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
	raw, err := codexOAuthAuthJSON(proxy.CodexOAuthTokens{
		AccessToken:  "access-token",
		IDToken:      idToken,
		RefreshToken: "refresh-token",
		ExpiresIn:    3600,
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	fields, ok := token.ExtractCodexAuthFields(raw)
	if !ok {
		t.Fatal("generated auth.json could not be parsed")
	}
	if fields.Email != "browser-login@example.com" || fields.AccountID != "account-browser-login" {
		t.Fatalf("unexpected identity: %+v", fields)
	}
	if fields.AccessToken != "access-token" || fields.RefreshToken != "refresh-token" {
		t.Fatalf("unexpected credentials: %+v", fields)
	}
	var saved map[string]any
	if err := json.Unmarshal([]byte(raw), &saved); err != nil {
		t.Fatal(err)
	}
	if _, ok := saved["tokens"]; ok {
		t.Fatalf("CPA Codex auth must keep token fields at the top level: %#v", saved)
	}
	if saved["type"] != "codex" || saved["email"] != "browser-login@example.com" {
		t.Fatalf("unexpected CPA identity fields: %#v", saved)
	}
	if saved["account_id"] != "account-browser-login" || saved["expired"] != now.Add(time.Hour).Format(time.RFC3339) {
		t.Fatalf("unexpected CPA account metadata: %#v", saved)
	}
	if saved["last_refresh"] != now.Format(time.RFC3339) {
		t.Fatalf("unexpected CPA last_refresh: %#v", saved)
	}
}

func TestCodexOAuthCallbackAcceptsCode(t *testing.T) {
	session := &codexOAuthSession{
		state:    "expected-state",
		callback: make(chan codexOAuthCallbackResult, 1),
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://localhost:1455/auth/callback?state=expected-state&code=authorization-code", nil)

	(&appServer{}).handleCodexOAuthCallback(session, recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected callback status: %d", recorder.Code)
	}
	result := <-session.callback
	if result.err != nil || result.code != "authorization-code" {
		t.Fatalf("unexpected callback result: %+v", result)
	}
}

func TestCodexOAuthCallbackRejectsStateMismatch(t *testing.T) {
	session := &codexOAuthSession{
		state:    "expected-state",
		callback: make(chan codexOAuthCallbackResult, 1),
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://localhost:1455/auth/callback?state=wrong-state&code=authorization-code", nil)

	(&appServer{}).handleCodexOAuthCallback(session, recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected callback status: %d", recorder.Code)
	}
	result := <-session.callback
	if result.err == nil || result.code != "" {
		t.Fatalf("expected state validation error, got %+v", result)
	}
}
