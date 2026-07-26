package proxy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/config"
	"omniproxy/internal/token"
)

func claudeOAuthTestToken() token.Token {
	return token.Token{
		ID:             "claude-1",
		Provider:       token.ProviderAnthropic,
		CredentialType: token.CredentialTypeClaudeOAuth,
		TokenValue:     `{"access_token":"test-access-token"}`,
	}
}

func useClaudeUsageEndpoint(t *testing.T, endpoint string) {
	t.Helper()
	original := claudeSubscriptionUsageEndpoint
	claudeSubscriptionUsageEndpoint = endpoint
	t.Cleanup(func() {
		claudeSubscriptionUsageEndpoint = original
	})
}

func TestParseClaudeSubscriptionUsageMapsWindows(t *testing.T) {
	usage, ok := parseClaudeSubscriptionUsage([]byte(`{
		"five_hour": {"utilization": 42.4, "resets_at": "2026-07-26T18:00:00Z"},
		"seven_day": {"utilization": 13, "resets_at": "2026-08-01T00:00:00Z"},
		"seven_day_opus": null,
		"extra_usage": {"is_enabled": false}
	}`))
	if !ok {
		t.Fatal("expected usage to parse")
	}
	if !usage.SubscriptionQuotaAvailable {
		t.Fatal("expected subscription quota to be available")
	}
	if usage.PrimaryUsedPercent != 42 || usage.PrimaryRemainingPercent != 58 {
		t.Fatalf("unexpected five-hour window: %#v", usage)
	}
	if usage.SecondaryUsedPercent != 13 || usage.SecondaryRemainingPercent != 87 {
		t.Fatalf("unexpected seven-day window: %#v", usage)
	}
	if want := time.Date(2026, 7, 26, 18, 0, 0, 0, time.UTC).Unix(); usage.PrimaryResetAt != want {
		t.Fatalf("expected primary reset %d, got %d", want, usage.PrimaryResetAt)
	}
	if want := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC).Unix(); usage.SecondaryResetAt != want {
		t.Fatalf("expected secondary reset %d, got %d", want, usage.SecondaryResetAt)
	}
}

func TestParseClaudeSubscriptionUsageKeepsUnusedWindowVisible(t *testing.T) {
	usage, ok := parseClaudeSubscriptionUsage([]byte(`{"five_hour": {"utilization": 0, "resets_at": ""}}`))
	if !ok {
		t.Fatal("expected usage to parse")
	}
	// The UI treats a window as present when any of used/remaining/reset is
	// non-zero, so a fully unused window must still report 100% remaining.
	if usage.PrimaryRemainingPercent != 100 || usage.PrimaryUsedPercent != 0 {
		t.Fatalf("unexpected idle window: %#v", usage)
	}
}

func TestParseClaudeSubscriptionUsageRejectsUnusablePayloads(t *testing.T) {
	for _, body := range []string{
		`not json`,
		`{}`,
		`{"five_hour": null, "seven_day": null}`,
		`{"five_hour": {"resets_at": "2026-07-26T18:00:00Z"}}`,
	} {
		if _, ok := parseClaudeSubscriptionUsage([]byte(body)); ok {
			t.Fatalf("expected %q to be rejected", body)
		}
	}
}

func TestValidatorSkipsClaudeUsageWhenDisabled(t *testing.T) {
	called := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	useClaudeUsageEndpoint(t, upstream.URL)

	validator, err := NewValidator(config.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, ok := validator.queryClaudeSubscriptionUsage(context.Background(), claudeOAuthTestToken()); ok {
		t.Fatal("expected no usage while the toggle is off")
	}
	if called {
		t.Fatal("the usage endpoint must not be contacted while the toggle is off")
	}
}

func TestValidatorFetchesClaudeSubscriptionUsage(t *testing.T) {
	var gotAuth, gotAgent, gotBeta string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAgent = r.Header.Get("User-Agent")
		gotBeta = strings.Join(r.Header.Values("anthropic-beta"), ",")
		_, _ = w.Write([]byte(`{"five_hour": {"utilization": 50, "resets_at": "2026-07-26T18:00:00Z"}}`))
	}))
	defer upstream.Close()
	useClaudeUsageEndpoint(t, upstream.URL)

	validator, err := NewValidator(config.Config{ClaudeSubscriptionUsageEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	usage, remaining, ok := validator.queryClaudeSubscriptionUsage(context.Background(), claudeOAuthTestToken())
	if !ok {
		t.Fatal("expected usage to be fetched")
	}
	if gotAuth != "Bearer test-access-token" {
		t.Fatalf("unexpected authorization header: %q", gotAuth)
	}
	if gotAgent != claudeSubscriptionUsageUserAgent {
		t.Fatalf("unexpected user agent: %q", gotAgent)
	}
	if !strings.Contains(gotBeta, "oauth-2025-04-20") {
		t.Fatalf("expected the oauth beta header, got %q", gotBeta)
	}
	if usage.PrimaryUsedPercent != 50 || usage.Source != "claude" {
		t.Fatalf("unexpected usage: %#v", usage)
	}
	if remaining == nil || *remaining != 50 {
		t.Fatalf("unexpected remaining: %v", remaining)
	}
}
