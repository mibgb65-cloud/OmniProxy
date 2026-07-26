package proxy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestRefreshCodexAuthJSONUsesRefreshToken(t *testing.T) {
	now := time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)
	var form url.Values
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); !strings.Contains(got, "application/x-www-form-urlencoded") {
			t.Fatalf("unexpected content type: %q", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		form = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "new-access-token",
			"id_token": "new-id-token",
			"refresh_token": "new-refresh-token",
			"token_type": "Bearer",
			"expires_in": 864000
		}`))
	}))
	defer upstream.Close()

	restore := replaceHTTPPostFormForTest(func(ctx context.Context, client *http.Client, endpoint string, values url.Values) (*http.Response, error) {
		if endpoint != codexOAuthTokenEndpoint {
			t.Fatalf("unexpected endpoint: %s", endpoint)
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstream.URL, strings.NewReader(values.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return upstream.Client().Do(req)
	})
	defer restore()

	raw := `{
		"auth_mode": "chatgpt",
		"tokens": {
			"access_token": "` + jwtWithExp(t, now.Add(time.Minute)) + `",
			"id_token": "` + jwtWithExp(t, now.Add(time.Hour)) + `",
			"refresh_token": "old-refresh-token",
			"account_id": "account-123"
		}
	}`
	updated, refreshed, err := RefreshCodexAuthJSON(context.Background(), upstream.Client(), raw, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if !refreshed {
		t.Fatal("expected refresh")
	}
	if form.Get("grant_type") != "refresh_token" {
		t.Fatalf("unexpected grant_type: %q", form.Get("grant_type"))
	}
	if form.Get("client_id") != codexOAuthClientID {
		t.Fatalf("unexpected client_id: %q", form.Get("client_id"))
	}
	if form.Get("refresh_token") != "old-refresh-token" {
		t.Fatalf("unexpected refresh_token: %q", form.Get("refresh_token"))
	}
	if form.Get("scope") != codexOAuthRefreshScope {
		t.Fatalf("unexpected scope: %q", form.Get("scope"))
	}

	var saved struct {
		Tokens struct {
			AccessToken  string `json:"access_token"`
			IDToken      string `json:"id_token"`
			RefreshToken string `json:"refresh_token"`
			AccountID    string `json:"account_id"`
		} `json:"tokens"`
		LastRefresh string `json:"last_refresh"`
	}
	if err := json.Unmarshal([]byte(updated), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Tokens.AccessToken != "new-access-token" || saved.Tokens.IDToken != "new-id-token" || saved.Tokens.RefreshToken != "new-refresh-token" {
		t.Fatalf("unexpected refreshed tokens: %#v", saved.Tokens)
	}
	if saved.Tokens.AccountID != "account-123" {
		t.Fatalf("account id should be preserved, got %q", saved.Tokens.AccountID)
	}
	if saved.LastRefresh != now.Format(time.RFC3339Nano) {
		t.Fatalf("unexpected last_refresh: %q", saved.LastRefresh)
	}
}

func TestRefreshCodexAuthJSONSkipsFreshAccessToken(t *testing.T) {
	now := time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)
	restore := replaceHTTPPostFormForTest(func(context.Context, *http.Client, string, url.Values) (*http.Response, error) {
		t.Fatal("refresh endpoint should not be called")
		return nil, nil
	})
	defer restore()

	raw := `{"tokens":{"access_token":"` + jwtWithExp(t, now.Add(2*time.Hour)) + `","refresh_token":"refresh-token"}}`
	updated, refreshed, err := RefreshCodexAuthJSON(context.Background(), nil, raw, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if refreshed {
		t.Fatal("expected no refresh")
	}
	if updated != raw {
		t.Fatalf("expected original auth JSON")
	}
}

func TestRefreshCodexAuthJSONSupportsFlatCLIProxyAPIFormat(t *testing.T) {
	now := time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)
	var form url.Values
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		form = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "new-flat-access-token",
			"id_token": "new-flat-id-token",
			"refresh_token": "new-flat-refresh-token",
			"expires_in": 3600
		}`))
	}))
	defer upstream.Close()

	restore := replaceHTTPPostFormForTest(func(ctx context.Context, client *http.Client, endpoint string, values url.Values) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstream.URL, strings.NewReader(values.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return upstream.Client().Do(req)
	})
	defer restore()

	raw := `{
		"type": "codex",
		"email": "coder@example.com",
		"access_token": "opaque-flat-access-token",
		"id_token": "old-flat-id-token",
		"refresh_token": "old-flat-refresh-token",
		"account_id": "flat-account",
		"expired": "` + now.Add(time.Minute).Format(time.RFC3339) + `"
	}`
	updated, refreshed, err := RefreshCodexAuthJSON(context.Background(), upstream.Client(), raw, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if !refreshed {
		t.Fatal("expected refresh")
	}
	if form.Get("refresh_token") != "old-flat-refresh-token" {
		t.Fatalf("unexpected refresh_token: %q", form.Get("refresh_token"))
	}

	var saved map[string]any
	if err := json.Unmarshal([]byte(updated), &saved); err != nil {
		t.Fatal(err)
	}
	if _, ok := saved["tokens"]; ok {
		t.Fatalf("flat auth JSON should stay flat, got %s", updated)
	}
	if saved["access_token"] != "new-flat-access-token" || saved["id_token"] != "new-flat-id-token" || saved["refresh_token"] != "new-flat-refresh-token" {
		t.Fatalf("unexpected refreshed flat tokens: %#v", saved)
	}
	if saved["type"] != "codex" || saved["email"] != "coder@example.com" || saved["account_id"] != "flat-account" {
		t.Fatalf("flat metadata should be preserved, got %#v", saved)
	}
	if saved["last_refresh"] != now.Format(time.RFC3339Nano) {
		t.Fatalf("unexpected last_refresh: %q", saved["last_refresh"])
	}
	if saved["expired"] != now.Add(time.Hour).Format(time.RFC3339) {
		t.Fatalf("unexpected expired timestamp: %q", saved["expired"])
	}
}

func TestRefreshCodexAuthJSONUsesStoredClientIDAndTokenExpiry(t *testing.T) {
	now := time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)
	var form url.Values
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		form = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"access_token": "new-custom-access-token",
			"id_token": "new-custom-id-token",
			"expires_in": 7200
		}`))
	}))
	defer upstream.Close()

	restore := replaceHTTPPostFormForTest(func(ctx context.Context, client *http.Client, endpoint string, values url.Values) (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, upstream.URL, strings.NewReader(values.Encode()))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return upstream.Client().Do(req)
	})
	defer restore()

	raw := `{
		"tokens": {
			"access_token": "opaque-access-token",
			"refresh_token": "old-custom-refresh-token",
			"client_id": "custom-client-id",
			"expires_at": "` + now.Add(time.Minute).Format(time.RFC3339) + `"
		}
	}`
	updated, refreshed, err := RefreshCodexAuthJSON(context.Background(), upstream.Client(), raw, false, now)
	if err != nil {
		t.Fatal(err)
	}
	if !refreshed {
		t.Fatal("expected refresh")
	}
	if form.Get("client_id") != "custom-client-id" {
		t.Fatalf("unexpected client_id: %q", form.Get("client_id"))
	}
	if form.Get("scope") != codexOAuthRefreshScope {
		t.Fatalf("unexpected scope: %q", form.Get("scope"))
	}

	var saved struct {
		Tokens struct {
			AccessToken  string `json:"access_token"`
			IDToken      string `json:"id_token"`
			RefreshToken string `json:"refresh_token"`
			ClientID     string `json:"client_id"`
			ExpiresAt    string `json:"expires_at"`
		} `json:"tokens"`
	}
	if err := json.Unmarshal([]byte(updated), &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Tokens.AccessToken != "new-custom-access-token" || saved.Tokens.IDToken != "new-custom-id-token" {
		t.Fatalf("unexpected refreshed tokens: %#v", saved.Tokens)
	}
	if saved.Tokens.RefreshToken != "old-custom-refresh-token" {
		t.Fatalf("refresh_token should be preserved when response omits it, got %q", saved.Tokens.RefreshToken)
	}
	if saved.Tokens.ClientID != "custom-client-id" {
		t.Fatalf("client_id should be preserved, got %q", saved.Tokens.ClientID)
	}
	if saved.Tokens.ExpiresAt != now.Add(2*time.Hour).Format(time.RFC3339) {
		t.Fatalf("unexpected expires_at: %q", saved.Tokens.ExpiresAt)
	}
}

func TestRefreshCodexAuthJSONFailsWhenExpiredAndRefreshTokenMissing(t *testing.T) {
	now := time.Date(2026, 4, 29, 10, 0, 0, 0, time.UTC)
	raw := `{"tokens":{"access_token":"` + jwtWithExp(t, now.Add(-time.Minute)) + `"}}`
	if _, _, err := RefreshCodexAuthJSON(context.Background(), nil, raw, false, now); err == nil {
		t.Fatal("expected missing refresh token error")
	}
}

func TestCodexOAuthAuthorizationURLUsesPKCEAndState(t *testing.T) {
	redirectURI := "http://localhost:1455/auth/callback"
	authURL := CodexOAuthAuthorizationURL(redirectURI, "challenge-value", "state-value")
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme+"://"+parsed.Host+parsed.Path != codexOAuthAuthorizeEndpoint {
		t.Fatalf("unexpected authorize endpoint: %s", parsed.String())
	}
	query := parsed.Query()
	expected := map[string]string{
		"response_type":         "code",
		"client_id":             codexOAuthClientID,
		"redirect_uri":          redirectURI,
		"scope":                 codexOAuthLoginScope,
		"code_challenge":        "challenge-value",
		"code_challenge_method": "S256",
		"state":                 "state-value",
		"originator":            codexOAuthLoginOriginator,
	}
	for key, want := range expected {
		if got := query.Get(key); got != want {
			t.Fatalf("unexpected %s: got %q want %q", key, got, want)
		}
	}
	if query.Has("code_verifier") {
		t.Fatal("authorization URL must not expose the PKCE verifier")
	}
}

func TestExchangeCodexAuthorizationCode(t *testing.T) {
	var posted url.Values
	restore := replaceHTTPPostFormForTest(func(_ context.Context, _ *http.Client, endpoint string, values url.Values) (*http.Response, error) {
		if endpoint != codexOAuthTokenEndpoint {
			t.Fatalf("unexpected token endpoint: %s", endpoint)
		}
		posted = values
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(`{"access_token":"access","id_token":"id","refresh_token":"refresh","expires_in":3600}`)),
		}, nil
	})
	defer restore()

	validator := &Validator{client: http.DefaultClient}
	tokens, err := validator.ExchangeCodexAuthorizationCode(context.Background(), "authorization-code", "pkce-verifier", "http://localhost:1455/auth/callback")
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "access" || tokens.IDToken != "id" || tokens.RefreshToken != "refresh" || tokens.ExpiresIn != 3600 {
		t.Fatalf("unexpected OAuth tokens: %+v", tokens)
	}
	if posted.Get("grant_type") != "authorization_code" || posted.Get("code") != "authorization-code" || posted.Get("code_verifier") != "pkce-verifier" {
		t.Fatalf("unexpected token form: %v", posted)
	}
}

func replaceHTTPPostFormForTest(fn func(context.Context, *http.Client, string, url.Values) (*http.Response, error)) func() {
	previous := httpPostForm
	httpPostForm = fn
	return func() {
		httpPostForm = previous
	}
}

func jwtWithExp(t *testing.T, exp time.Time) string {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"exp": exp.Unix(),
		"https://api.openai.com/profile": map[string]string{
			"email": "coder@example.com",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return "header." + base64.RawURLEncoding.EncodeToString(payload) + ".signature"
}

// The pre-check runs before the refresh lock, so it has to agree with the
// refresh path. If it under-reports, tokens silently stop being refreshed.
func TestCodexAuthNeedsRefreshMatchesRefreshDecision(t *testing.T) {
	now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	fresh := `{"tokens":{"access_token":"` + jwtWithExp(t, now.Add(2*time.Hour)) + `","refresh_token":"refresh-token"}}`
	if CodexAuthNeedsRefresh(fresh, now) {
		t.Fatal("a token valid for another two hours must not need refreshing")
	}
	updated, refreshed, err := RefreshCodexAuthJSON(context.Background(), nil, fresh, false, now)
	if err != nil || refreshed || updated != fresh {
		t.Fatalf("refresh path disagreed with the pre-check: refreshed=%v err=%v", refreshed, err)
	}

	expiring := `{"tokens":{"access_token":"` + jwtWithExp(t, now.Add(time.Minute)) + `","refresh_token":"refresh-token"}}`
	if !CodexAuthNeedsRefresh(expiring, now) {
		t.Fatal("a token inside the refresh margin must need refreshing")
	}

	if !CodexAuthNeedsRefresh("not json", now) {
		t.Fatal("an unparseable credential must fall through to the refresh path")
	}
}
