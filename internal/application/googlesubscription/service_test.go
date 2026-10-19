package googlesubscription

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/subscription"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/antigravity"
)

type memoryStore struct {
	values map[string][]byte
	next   int
}

func (s *memoryStore) Put(value []byte) (string, error) {
	s.next++
	key := fmt.Sprintf("test-ref-%d", s.next)
	s.values[key] = append([]byte(nil), value...)
	return key, nil
}
func (s *memoryStore) Get(key string) ([]byte, error) {
	return append([]byte(nil), s.values[key]...), nil
}
func (s *memoryStore) Delete(key string) error { delete(s.values, key); return nil }

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func clientConfig() antigravity.OAuthConfig {
	return antigravity.OAuthConfig{ClientID: "1071006060591-tmhssin2h21lcre235vtolojh4g403ep.apps.googleusercontent.com", ClientSecret: "GOCSPX-" + strings.Repeat("x", 28)}
}

func setupService(t *testing.T) (*Service, *atomic.Int32) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "google.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	refreshes := &atomic.Int32{}
	client := antigravity.New(roundTrip(func(r *http.Request) (*http.Response, error) {
		body := ""
		switch r.URL.Path {
		case "/token":
			refreshes.Add(1)
			body = `{"access_token":"test-access","refresh_token":"test-refresh","expires_in":3600}`
		case "/oauth2/v2/userinfo":
			body = `{"id":"1001","email":"person@example.test","verified_email":true}`
		case "/v1internal:loadCodeAssist":
			body = `{"cloudaicompanionProject":"test-project","paidTier":{"id":"test-tier"}}`
		case "/v1internal:fetchAvailableModels":
			body = `{"models":{"gemini-test":{"displayName":"Gemini Test"}}}`
		default:
			t.Error("unexpected endpoint")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	manager := New(account.NewService(db, &memoryStore{values: map[string][]byte{}}), client)
	manager.config = func() (antigravity.OAuthConfig, error) { return clientConfig(), nil }
	t.Cleanup(manager.Close)
	return manager, refreshes
}

func TestBrowserLoginRejectsInvalidStateAndPreservesSession(t *testing.T) {
	manager, _ := setupService(t)
	stages := make(chan string, 4)
	manager.OnChange(func() {
		manager.mu.Lock()
		state := manager.current.view.State
		manager.mu.Unlock()
		stages <- state
	})
	view, err := manager.Begin(context.Background(), subscription.Input{ProviderID: "builtin_google", Mode: "browser"})
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(view.URL)
	callback := u.Query().Get("redirect_uri")
	if !strings.HasPrefix(callback, "http://127.0.0.1:") {
		t.Fatal("callback is not loopback")
	}
	response, err := http.Get(callback + "?state=wrong&code=test-code")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 400 {
		t.Fatal("invalid state was accepted")
	}
	resumed, err := manager.Current("builtin_google", "")
	if err != nil || resumed.ID != view.ID {
		t.Fatal("pending authorization could not resume")
	}
	response, err = http.Get(callback + "?state=" + url.QueryEscape(u.Query().Get("state")) + "&code=test-code")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	select {
	case <-manager.current.done:
	case <-time.After(3 * time.Second):
		t.Fatal("login did not complete")
	}
	completed, err := manager.Read(view.ID)
	if err != nil || completed.State != "completed" || completed.URL != "" || completed.Account == nil || completed.Account.AuthAccountID != "1001" {
		t.Fatal("successful login was not saved safely")
	}
	var observed []string
	for len(stages) > 0 {
		observed = append(observed, <-stages)
	}
	if strings.Join(observed, ",") != "exchanging,saving,completed" {
		t.Fatalf("missing real Google authorization stages: %v", observed)
	}
	models, version, err := manager.Models(context.Background(), completed.Account.ID)
	if err != nil || len(models) != 1 || version == "" {
		t.Fatal("account catalog could not be discovered")
	}
}

func TestConcurrentCredentialRefreshIsDeduplicated(t *testing.T) {
	manager, refreshes := setupService(t)
	cred := antigravity.Credentials{OAuthConfig: clientConfig(), AccessToken: "test-old-access", RefreshToken: "test-refresh", ExpiresAt: time.Now().Add(-time.Minute), Subject: "1001", Email: "person@example.test", Project: "test-project", Plan: "test-tier"}
	item, err := manager.accounts.SaveGoogleSubscription(context.Background(), "builtin_google", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for range 10 {
		group.Go(func() {
			if err := manager.Use(context.Background(), item.ID, func(current antigravity.Credentials) error {
				if current.AccessToken != "test-access" {
					t.Error("expired token consumed")
				}
				return nil
			}); err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if refreshes.Load() != 1 {
		t.Fatal("concurrent calls refreshed repeatedly")
	}
	if _, err := manager.Logout(context.Background(), item.ID); err != nil {
		t.Fatal(err)
	}
	if err := manager.Use(context.Background(), item.ID, func(antigravity.Credentials) error { return nil }); err == nil {
		t.Fatal("logged-out account was used")
	}
}

func TestCancellationAndProviderIsolation(t *testing.T) {
	manager, _ := setupService(t)
	if _, err := manager.Begin(context.Background(), subscription.Input{ProviderID: "builtin_openai", Mode: "browser"}); err == nil {
		t.Fatal("Google login accepted another provider")
	}
	view, err := manager.Begin(context.Background(), subscription.Input{ProviderID: "builtin_google", Mode: "browser"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Begin(context.Background(), subscription.Input{ProviderID: "builtin_google", Mode: "browser"}); antigravity.ErrorCode(err) != "busy" {
		t.Fatal("overlapping login was accepted")
	}
	result, err := manager.Cancel(view.ID)
	if err != nil || result.State != "failed" || result.ErrorCode != "cancelled" || result.URL != "" {
		t.Fatal("cancellation did not stop authorization")
	}
}
