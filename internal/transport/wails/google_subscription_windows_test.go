//go:build windows

package wails

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/application/googlesubscription"
	"omniproxy/internal/infrastructure/credential"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/platform/profile"
	"omniproxy/internal/provider/antigravity"
)

type googleQuotaTransport func(*http.Request) (*http.Response, error)

func (f googleQuotaTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGoogleQuotaBindingOnlyReturnsSafeWindowData(t *testing.T) {
	ctx := context.Background()
	prof := &profile.Profile{Kind: profile.Test, DataDir: t.TempDir()}
	db, err := database.Open(database.Config{Path: filepath.Join(prof.DataDir, "quota.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	service := &AppService{prof: prof, db: db}
	defer service.shutdownGoogleSubscription()
	accounts, err := service.accountService()
	if err != nil {
		t.Fatal(err)
	}
	cred := antigravity.Credentials{OAuthConfig: antigravity.OAuthConfig{ClientID: "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com", ClientSecret: "GOCSPX-" + strings.Repeat("x", 28)}, AccessToken: "test-private-access", RefreshToken: "test-private-refresh", ExpiresAt: time.Now().Add(time.Hour), Subject: "1001", Email: "person@example.test", Project: "test-private-project"}
	item, err := accounts.SaveGoogleSubscription(ctx, "builtin_google", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	client := antigravity.New(googleQuotaTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1internal:retrieveUserQuotaSummary" || r.Header.Get("Authorization") != "Bearer test-private-access" {
			t.Error("binding used an unexpected identity or endpoint")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"groups":[{"displayName":"Gemini Models","buckets":[{"bucketId":"gemini-weekly","window":"weekly","remainingFraction":0.25}]}],"email":"person@example.test"}`))}, nil
	}))
	service.googleAuth.service = googlesubscription.New(accounts, client)
	quota, err := service.GoogleSubscriptionQuota(item.ID)
	if err != nil || len(quota.Groups) != 1 || *quota.Groups[0].Buckets[0].RemainingPercent != 25 {
		t.Fatal("safe Google quota binding failed")
	}
	data, _ := json.Marshal(quota)
	for _, private := range []string{cred.AccessToken, cred.RefreshToken, cred.Project, cred.Email} {
		if strings.Contains(string(data), private) {
			t.Fatal("binding returned private account information")
		}
	}
	if service.subscription.service != nil {
		t.Fatal("quota binding entered Codex subscription service")
	}
}

func TestGoogleSubscriptionBindingsKeepAuthorizationIsolated(t *testing.T) {
	ctx := context.Background()
	prof := &profile.Profile{Kind: profile.Test, DataDir: t.TempDir()}
	db, err := database.Open(database.Config{Path: filepath.Join(prof.DataDir, "test.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	service := &AppService{prof: prof, db: db}
	defer service.shutdownGoogleSubscription()
	view, err := service.SubscriptionCurrent("builtin_google", "")
	if err != nil || view.ID != "" {
		t.Fatal("Google binding entered Codex login")
	}
	accounts, err := service.accountService()
	if err != nil {
		t.Fatal(err)
	}
	cred := antigravity.Credentials{OAuthConfig: antigravity.OAuthConfig{ClientID: "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com", ClientSecret: "GOCSPX-" + strings.Repeat("x", 28)}, AccessToken: "fake-access", RefreshToken: "fake-refresh", ExpiresAt: time.Now().Add(time.Hour), Subject: "1001", Email: "person@example.test", Project: "test-project"}
	item, err := accounts.SaveGoogleSubscription(ctx, "builtin_google", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	var ref string
	if db.QueryRow("SELECT credential_ref FROM accounts WHERE id=?", item.ID).Scan(&ref) != nil {
		t.Fatal("protected reference missing")
	}
	store, err := credential.New(prof)
	if err != nil {
		t.Fatal(err)
	}
	data, err := store.Get(ref)
	if err != nil || !strings.Contains(string(data), "fake-access") {
		t.Fatal("DPAPI bundle unavailable")
	}
	clear(data)
	if _, err := service.SubscriptionLogout(item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(ref); err == nil {
		t.Fatal("logout kept a credential")
	}
	if err := service.DeleteAccount(item.ID); err != nil {
		t.Fatal(err)
	}
	if service.subscription.service != nil {
		t.Fatal("Google operation initialized a Codex process")
	}
}
