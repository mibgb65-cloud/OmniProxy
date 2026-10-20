package googlesubscription

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/provider/antigravity"
)

func quotaCredential() antigravity.Credentials {
	return antigravity.Credentials{OAuthConfig: clientConfig(), AccessToken: "quota-access", RefreshToken: "quota-refresh", ExpiresAt: time.Now().Add(time.Hour), Subject: "1001", Email: "person@example.test", Project: "test-project"}
}

func TestQuotaRejectsLogoutOrNewAuthorizationDuringRequest(t *testing.T) {
	for _, change := range []string{"none", "logout", "relogin"} {
		t.Run(change, func(t *testing.T) {
			manager, refreshes := setupService(t)
			cred := quotaCredential()
			item, err := manager.accounts.SaveGoogleSubscription(context.Background(), "builtin_google", "", "", cred)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			manager.client = antigravity.New(roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Path != "/v1internal:retrieveUserQuotaSummary" {
					t.Error("quota query performed an unrelated request")
				}
				if change == "logout" {
					if _, err := manager.accounts.LogoutGoogleSubscription(context.Background(), item.ID); err != nil {
						t.Error(err)
					}
				}
				if change == "relogin" {
					if _, err := manager.accounts.SaveGoogleSubscription(context.Background(), "builtin_google", item.ID, "", cred); err != nil {
						t.Error(err)
					}
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"groups":[{"displayName":"Gemini Models","buckets":[{"bucketId":"gemini-5h","window":"5h","remainingFraction":0.75}]}]}`))}, nil
			}))
			quota, err := manager.ReadQuota(context.Background(), item.ID)
			if change == "none" {
				if err != nil || len(quota.Groups) != 1 || *quota.Groups[0].Buckets[0].RemainingPercent != 75 {
					t.Fatal("quota was not read")
				}
				if saved, err := manager.CachedQuota(context.Background(), item.ID); err != nil || saved == nil || !saved.FromCache {
					t.Fatal("successful read was not persisted", saved, err)
				}
			} else if antigravity.ErrorCode(err) != "quota_changed" {
				t.Fatal("old authorization published quota into a new scope")
			}
			if calls != 1 || refreshes.Load() != 0 {
				t.Fatal("quota query unexpectedly refreshed or generated content")
			}
			if change == "logout" {
				if _, err := manager.ReadQuota(context.Background(), item.ID); err == nil || calls != 1 {
					t.Fatal("logged-out quota performed an upstream request")
				}
			}
		})
	}
}

func TestQuotaRejectsNonGoogleAccountBeforeNetwork(t *testing.T) {
	manager, refreshes := setupService(t)
	if _, err := manager.ReadQuota(context.Background(), "missing-account"); err == nil || refreshes.Load() != 0 {
		t.Fatal("invalid quota account accepted")
	}
}

func TestAccountModelsRejectNewAuthorizationDuringRead(t *testing.T) {
	manager, _ := setupService(t)
	cred := quotaCredential()
	item, err := manager.accounts.SaveGoogleSubscription(context.Background(), "builtin_google", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	manager.client = antigravity.New(roundTrip(func(r *http.Request) (*http.Response, error) {
		if _, err := manager.accounts.SaveGoogleSubscription(context.Background(), "builtin_google", item.ID, "", cred); err != nil {
			t.Error(err)
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"models":{"gemini-test":{"displayName":"Gemini"}},"agentModelSorts":[{"groups":[{"modelIds":["gemini-test"]}]}]}`))}, nil
	}))
	if _, err := manager.ReadAccountModels(context.Background(), item.ID); antigravity.ErrorCode(err) != "quota_changed" {
		t.Fatal("models from an old authorization were published")
	}
}

func TestQuotaCancellationStopsTheReadWithoutPublishingWindows(t *testing.T) {
	manager, _ := setupService(t)
	item, err := manager.accounts.SaveGoogleSubscription(context.Background(), "builtin_google", "", "", quotaCredential())
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	manager.client = antigravity.New(roundTrip(func(r *http.Request) (*http.Response, error) {
		close(started)
		<-r.Context().Done()
		return nil, r.Context().Err()
	}))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		quota, err := manager.ReadQuota(ctx, item.ID)
		if len(quota.Groups) != 0 {
			t.Error("cancelled read published quota")
		}
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("quota request did not start")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled read succeeded")
		}
	case <-time.After(time.Second):
		t.Fatal("quota read ignored cancellation")
	}
}
