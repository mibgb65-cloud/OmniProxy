package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/application/provider"
	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/antigravity"
)

type googleAccess struct{ calls int }

func (a *googleAccess) Use(_ context.Context, id string, consume func(antigravity.Credentials) error) error {
	a.calls++
	if id != "google-sub" {
		return antigravity.Failure("invalid_input")
	}
	return consume(antigravity.Credentials{AccessToken: "fake-google-token", Project: "test-project"})
}

type googleTransport func(*http.Request) (*http.Response, error)

func (f googleTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGoogleSubscriptionRoutesWithoutAPIKeyAndRecordsUsage(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "google-sub.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at,auth_method,auth_state,auth_account_id,auth_email) VALUES('google-sub','builtin_google','Google','protected-ref','2026-10-16T00:00:00Z','v1','subscription_login','authenticated','1001','person@example.test')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version) VALUES('google-sub','gemini-test','Gemini','now','protected-ref:1001')`)
	if err != nil {
		t.Fatal(err)
	}
	providers, accounts := provider.NewService(db), account.NewService(db, &testStore{})
	service := New(proxy.Options{Observer: activity.New(db)}, providers, accounts, nil)
	access := &googleAccess{}
	service.SetGoogleSubscription(access)
	upstreamCalls := 0
	service.antigravity = antigravity.New(googleTransport(func(r *http.Request) (*http.Response, error) {
		upstreamCalls++
		if r.URL.Host != "daily-cloudcode-pa.googleapis.com" || r.Header.Get("Authorization") != "Bearer fake-google-token" {
			t.Error("subscription used an incorrect endpoint or credential")
		}
		status, body := 200, `{"response":{"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":4,"candidatesTokenCount":1,"totalTokenCount":5}}}`
		if upstreamCalls == 2 {
			status, body = 429, `{"error":"fake-sensitive-upstream"}`
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	router := routes.New(db, providers, accounts)
	service.SetRouter(router)
	if _, err := router.Create(ctx, routes.Input{Alias: "google-sub-alias", ProviderID: "builtin_google", AccountID: "google-sub", Model: "gemini-test", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Start(); err != nil {
		t.Fatal(err)
	}
	defer service.Stop(ctx)
	endpoint := fmt.Sprintf("http://127.0.0.1:%d/v1/chat/completions", service.Info().Runtime.Port)
	for _, expected := range []int{200, 429} {
		response, err := http.Post(endpoint, "application/json", strings.NewReader(`{"model":"google-sub-alias","messages":[{"role":"user","content":"hi"}]}`))
		if err != nil {
			t.Fatal(err)
		}
		data, _ := io.ReadAll(response.Body)
		response.Body.Close()
		if response.StatusCode != expected || strings.Contains(string(data), "fake-sensitive-upstream") {
			var failure struct {
				Error struct {
					Code string `json:"code"`
					Type string `json:"type"`
				} `json:"error"`
			}
			json.Unmarshal(data, &failure)
			t.Fatalf("gateway status or redaction failed: got %d, expected %d, code=%s type=%s", response.StatusCode, expected, failure.Error.Code, failure.Error.Type)
		}
	}
	if access.calls != 2 || upstreamCalls != 2 {
		t.Fatal("subscription retried or used an API Key")
	}
	var input, output int64
	if db.QueryRow(`SELECT input_tokens,output_tokens FROM request_records WHERE account_id='google-sub' AND input_tokens>0`).Scan(&input, &output) != nil || input != 4 || output != 1 {
		t.Fatal("usage was not attributed to the subscription account")
	}
	_, err = db.Exec(`UPDATE accounts SET auth_state='logged_out' WHERE id='google-sub'`)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := router.Resolve(ctx, "google-sub-alias")
	if err != nil || decision.Routable {
		t.Fatal("logged-out account remained routable")
	}
}
