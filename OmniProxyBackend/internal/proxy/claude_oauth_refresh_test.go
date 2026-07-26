package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRefreshClaudeOAuthJSONUsesRefreshToken(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	var payload map[string]string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); !strings.Contains(got, "application/json") {
			t.Fatalf("unexpected content type: %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "new-claude-access",
			"refresh_token": "new-claude-refresh",
			"token_type": "Bearer",
			"expires_in": 3600
		}`))
	}))
	defer upstream.Close()

	restore := replaceHTTPPostJSONForTest(func(ctx context.Context, client *http.Client, endpoint string, body any) (*http.Response, error) {
		if endpoint != claudeOAuthTokenEndpoint {
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

	raw := `{"access_token":"old-access","refresh_token":"old-refresh","email":"claude@example.com","expired":"2026-05-01T00:10:00Z"}`
	updated, refreshed, err := RefreshClaudeOAuthJSON(context.Background(), upstream.Client(), raw, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if !refreshed {
		t.Fatal("expected refresh")
	}
	if payload["grant_type"] != "refresh_token" || payload["client_id"] != claudeOAuthClientID || payload["refresh_token"] != "old-refresh" {
		t.Fatalf("unexpected refresh payload: %#v", payload)
	}

	var saved struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		Email        string `json:"email"`
		Expired      string `json:"expired"`
		LastRefresh  string `json:"last_refresh"`
	}
	if err := json.Unmarshal([]byte(updated), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.AccessToken != "new-claude-access" || saved.RefreshToken != "new-claude-refresh" {
		t.Fatalf("unexpected refreshed token fields: %#v", saved)
	}
	if saved.Email != "claude@example.com" {
		t.Fatalf("email should be preserved, got %q", saved.Email)
	}
	if saved.LastRefresh != now.Format(time.RFC3339) {
		t.Fatalf("unexpected last_refresh: %q", saved.LastRefresh)
	}
}

func TestRefreshClaudeOAuthJSONSkipsFreshToken(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	restore := replaceHTTPPostJSONForTest(func(context.Context, *http.Client, string, any) (*http.Response, error) {
		t.Fatal("refresh endpoint should not be called")
		return nil, nil
	})
	defer restore()

	raw := `{"access_token":"fresh-access","refresh_token":"refresh","expired":"2026-05-01T02:00:00Z"}`
	updated, refreshed, err := RefreshClaudeOAuthJSON(context.Background(), nil, raw, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed {
		t.Fatal("expected no refresh")
	}
	if updated != raw {
		t.Fatalf("expected original JSON")
	}
}

func replaceHTTPPostJSONForTest(fn func(context.Context, *http.Client, string, any) (*http.Response, error)) func() {
	previous := httpPostJSON
	httpPostJSON = fn
	return func() {
		httpPostJSON = previous
	}
}

func TestClaudeOAuthNeedsRefreshMatchesRefreshDecision(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	fresh := `{"access_token":"fresh-access","refresh_token":"refresh","expired":"2026-05-01T02:00:00Z"}`
	if ClaudeOAuthNeedsRefresh(fresh, now) {
		t.Fatal("a token valid for another two hours must not need refreshing")
	}

	restore := replaceHTTPPostJSONForTest(func(context.Context, *http.Client, string, any) (*http.Response, error) {
		t.Fatal("refresh endpoint must not be called for a fresh token")
		return nil, nil
	})
	defer restore()
	if _, refreshed, err := RefreshClaudeOAuthJSON(context.Background(), nil, fresh, false, now); err != nil || refreshed {
		t.Fatalf("refresh path disagreed with the pre-check: refreshed=%v err=%v", refreshed, err)
	}

	expiring := `{"access_token":"stale","refresh_token":"refresh","expired":"2026-05-01T00:10:00Z"}`
	if !ClaudeOAuthNeedsRefresh(expiring, now) {
		t.Fatal("a token inside the refresh margin must need refreshing")
	}
	if !ClaudeOAuthNeedsRefresh("not json", now) {
		t.Fatal("an unparseable credential must fall through to the refresh path")
	}
}
