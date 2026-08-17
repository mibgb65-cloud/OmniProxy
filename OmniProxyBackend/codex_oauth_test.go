package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"omniproxy/internal/logs"
	"omniproxy/internal/proxy"
	"omniproxy/internal/storage"
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

func TestUpsertCodexOAuthTokenKeepsDifferentEmailsWithSharedTeamAccountID(t *testing.T) {
	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	firstAuth := codexAuthJSONForMainTestWithCredentials(t, "first@example.com", "shared-team-account", "first-access-token")
	secondAuth := codexAuthJSONForMainTestWithCredentials(t, "second@example.com", "shared-team-account", "second-access-token")
	first, err := manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     firstAuth,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	app := &appServer{tokens: manager, logs: logs.NewRecorder(10)}
	created, err := app.upsertCodexOAuthToken(ctx, secondAuth)
	if err != nil {
		t.Fatal(err)
	}

	if created.ID == first.ID {
		t.Fatal("different email was treated as the existing Team account")
	}
	items := manager.List()
	if len(items) != 2 {
		t.Fatalf("expected both email accounts to be retained, got %#v", items)
	}
	storedFirst, err := manager.Get(first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedFirst.TokenValue != firstAuth {
		t.Fatal("first email account credentials were replaced")
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

func TestCodexOAuthLoginStatusTracksCallback(t *testing.T) {
	session := &codexOAuthSession{
		id:        "login-1",
		expiresAt: time.Now().Add(time.Minute),
		callback:  make(chan codexOAuthCallbackResult, 1),
	}
	server := &appServer{codexOAuthSession: session}

	status, err := server.codexOAuthLoginStatus("login-1")
	if err != nil {
		t.Fatal(err)
	}
	if status.Ready {
		t.Fatal("login must remain pending before the callback arrives")
	}

	session.callback <- codexOAuthCallbackResult{code: "authorization-code"}
	status, err = server.codexOAuthLoginStatus("login-1")
	if err != nil {
		t.Fatal(err)
	}
	if !status.Ready {
		t.Fatal("login must become ready after the callback arrives")
	}
}

func TestCodexOAuthLoginStatusRejectsExpiredSession(t *testing.T) {
	server := &appServer{codexOAuthSession: &codexOAuthSession{
		id:        "expired-login",
		expiresAt: time.Now().Add(-time.Second),
		callback:  make(chan codexOAuthCallbackResult, 1),
	}}

	if _, err := server.codexOAuthLoginStatus("expired-login"); err == nil {
		t.Fatal("expected an expired login error")
	}
}
