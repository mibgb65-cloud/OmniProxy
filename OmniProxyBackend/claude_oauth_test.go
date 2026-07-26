package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"omniproxy/internal/proxy"
	"omniproxy/internal/token"
)

func TestClaudeOAuthAuthJSONPreservesIdentityAndRefreshToken(t *testing.T) {
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	raw, err := claudeOAuthAuthJSON(proxy.ClaudeOAuthTokens{
		AccessToken:    "claude-access",
		RefreshToken:   "claude-refresh",
		ExpiresIn:      3600,
		Scope:          "user:profile user:inference",
		Email:          "claude@example.com",
		AccountID:      "account-1",
		OrganizationID: "organization-1",
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	fields, ok := token.ExtractClaudeOAuthFields(raw)
	if !ok {
		t.Fatal("generated Claude OAuth JSON could not be parsed")
	}
	if fields.Email != "claude@example.com" || fields.AccountID != "account-1" {
		t.Fatalf("unexpected identity: %+v", fields)
	}
	if fields.AccessToken != "claude-access" || fields.RefreshToken != "claude-refresh" {
		t.Fatalf("unexpected credentials: %+v", fields)
	}

	var saved map[string]any
	if err := json.Unmarshal([]byte(raw), &saved); err != nil {
		t.Fatal(err)
	}
	nested, ok := saved["claudeAiOauth"].(map[string]any)
	if !ok {
		t.Fatalf("missing Claude Code credential object: %#v", saved)
	}
	if nested["expiresAt"] != float64(now.Add(time.Hour).UnixMilli()) {
		t.Fatalf("unexpected expiry: %#v", nested)
	}
}

func TestClaudeOAuthCallbackAcceptsCode(t *testing.T) {
	session := &claudeOAuthSession{
		state:    "expected-state",
		callback: make(chan claudeOAuthCallbackResult, 1),
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://localhost:43123/callback?state=expected-state&code=authorization-code", nil)

	(&appServer{}).handleClaudeOAuthCallback(session, recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected callback status: %d", recorder.Code)
	}
	result := <-session.callback
	if result.err != nil || result.code != "authorization-code" {
		t.Fatalf("unexpected callback result: %+v", result)
	}
}

func TestClaudeOAuthCallbackRejectsStateMismatch(t *testing.T) {
	session := &claudeOAuthSession{
		state:    "expected-state",
		callback: make(chan claudeOAuthCallbackResult, 1),
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "http://localhost:43123/callback?state=wrong-state&code=authorization-code", nil)

	(&appServer{}).handleClaudeOAuthCallback(session, recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected callback status: %d", recorder.Code)
	}
	result := <-session.callback
	if result.err == nil || result.code != "" {
		t.Fatalf("expected state validation error, got %+v", result)
	}
}

func TestClaudeOAuthLoginStatusTracksCallback(t *testing.T) {
	session := &claudeOAuthSession{
		id:        "login-1",
		expiresAt: time.Now().Add(time.Minute),
		callback:  make(chan claudeOAuthCallbackResult, 1),
	}
	server := &appServer{claudeOAuthSession: session}

	status, err := server.claudeOAuthLoginStatus("login-1")
	if err != nil {
		t.Fatal(err)
	}
	if status.Ready {
		t.Fatal("login must remain pending before the callback arrives")
	}

	session.callback <- claudeOAuthCallbackResult{code: "authorization-code"}
	status, err = server.claudeOAuthLoginStatus("login-1")
	if err != nil {
		t.Fatal(err)
	}
	if !status.Ready {
		t.Fatal("login must become ready after the callback arrives")
	}
}
