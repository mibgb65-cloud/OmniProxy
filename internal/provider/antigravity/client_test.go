package antigravity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func reply(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}
func testConfig() OAuthConfig {
	return OAuthConfig{desktopClientID, "GOCSPX-" + strings.Repeat("x", 28)}
}
func testCredentials() Credentials {
	return Credentials{OAuthConfig: testConfig(), AccessToken: "test-access", RefreshToken: "test-refresh", Subject: "1001", Email: "person@example.test", Project: "test-project", Plan: "test-tier", ExpiresAt: time.Now().Add(time.Hour)}
}

func TestOAuthConfigBoundariesAndInstalledComponent(t *testing.T) {
	config := testConfig()
	fingerprint := sha256.Sum256([]byte(config.ClientSecret))
	for _, test := range []struct {
		name, source string
		valid        bool
	}{
		{"valid", strings.Repeat("x", 65520) + config.ClientID + config.ClientSecret, true},
		{"missing-id", config.ClientSecret, false},
		{"missing-secret", config.ClientID, false},
		{"other-client", config.ClientID + config.ClientSecret + "GOCSPX-" + strings.Repeat("y", 28), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			value, err := readOAuthConfig(strings.NewReader(test.source), hex.EncodeToString(fingerprint[:]))
			if (err == nil) != test.valid {
				t.Fatal("configuration validation mismatch")
			}
			if test.valid && (value != config) {
				t.Fatal("configuration changed")
			}
		})
	}
	if os.Getenv("OMNIPROXY_TEST_AGY_CONFIG") == "1" {
		value, err := InstalledOAuthConfig()
		if err != nil || value.ClientID != desktopClientID || len(value.ClientSecret) != 35 {
			t.Fatalf("installed official client configuration was not recognized: %s", ErrorCode(err))
		}
	}
}

func TestOAuthPKCEIdentityProjectAndRefresh(t *testing.T) {
	config := testConfig()
	authorization, err := url.Parse(AuthorizationURL(config, "state", "verifier", "http://127.0.0.1:1234/oauth-callback"))
	if err != nil {
		t.Fatal(err)
	}
	if authorization.Host != "accounts.google.com" || authorization.Query().Get("state") != "state" || authorization.Query().Get("code_challenge_method") != "S256" || authorization.Query().Get("code_challenge") == "verifier" {
		t.Fatal("OAuth challenge is missing")
	}
	client := New(transportFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Host + r.URL.Path {
		case "oauth2.googleapis.com/token":
			data, _ := io.ReadAll(r.Body)
			values, _ := url.ParseQuery(string(data))
			if values.Get("client_id") != config.ClientID || values.Get("client_secret") != config.ClientSecret {
				t.Error("client configuration was not sent to token endpoint")
			}
			if values.Get("grant_type") == "authorization_code" && values.Get("code_verifier") != "verifier" {
				t.Error("PKCE verifier missing")
			}
			return reply(200, `{"access_token":"test-access","refresh_token":"test-refresh","expires_in":3600}`), nil
		case "www.googleapis.com/oauth2/v2/userinfo":
			return reply(200, `{"id":"1001","email":"person@example.test","verified_email":true}`), nil
		case "cloudcode-pa.googleapis.com/v1internal:loadCodeAssist":
			return reply(200, `{"cloudaicompanionProject":{"id":"test-project"},"paidTier":{"id":"test-tier"}}`), nil
		default:
			t.Error("unexpected upstream endpoint")
			return reply(500, `{}`), nil
		}
	}))
	cred, err := client.Exchange(context.Background(), config, "code", "verifier", "http://127.0.0.1:1234/oauth-callback")
	if err != nil || !cred.Valid() || cred.Subject != "1001" || cred.Project != "test-project" {
		t.Fatal("authorization did not establish a valid identity and project")
	}
	if err := client.Refresh(context.Background(), &cred); err != nil {
		t.Fatal(err)
	}
	cred.Subject = "different"
	if ErrorCode(client.Identity(context.Background(), &cred)) != "identity_changed" {
		t.Fatal("identity change was accepted")
	}
}

func TestModelsRequireAuthoritativeCatalogAndErrorsAreRedacted(t *testing.T) {
	body := `{"models":{"gemini-test":{"displayName":"Gemini Test"},"claude-test":{},"gemini-image":{}}}`
	status := 200
	client := New(transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "daily-cloudcode-pa.googleapis.com" || r.URL.Path != "/v1internal:fetchAvailableModels" {
			t.Error("model catalog endpoint changed")
		}
		return reply(status, body), nil
	}))
	models, err := client.Models(context.Background(), testCredentials())
	if err != nil || len(models) != 1 || models[0].ID != "gemini-test" {
		t.Fatal("catalog did not filter unsupported models")
	}
	body = `{"models":null}`
	if _, err := client.Models(context.Background(), testCredentials()); err == nil {
		t.Fatal("missing catalog accepted")
	}
	status, body = 401, `{"error":"test-private-token"}`
	_, err = client.Models(context.Background(), testCredentials())
	if err == nil || strings.Contains(err.Error(), "test-private-token") || ErrorCode(err) != "reauth_required" {
		t.Fatal("upstream error was exposed")
	}
	data, _ := json.Marshal(models)
	if strings.Contains(string(data), "test-access") {
		t.Fatal("catalog contains credentials")
	}
}
