package googlesubscription

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/core/netproxy"
	"omniproxy/internal/provider/antigravity"
)

// 额度、可选模型与令牌续期都必须带上账号所属提供商，否则「只给这个提供商配置代理」会静默直连。
func TestQuotaAndModelsCarryProviderScope(t *testing.T) {
	manager, _ := setupService(t)
	cred := quotaCredential()
	item, err := manager.accounts.SaveGoogleSubscription(context.Background(), "builtin_google", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	var scopes []netproxy.Scope
	manager.client = antigravity.New(roundTrip(func(r *http.Request) (*http.Response, error) {
		scope, ok := netproxy.ScopeFrom(r.Context())
		if !ok || scope.ProviderID != item.ProviderID {
			t.Errorf("%s scope = %+v, ok = %v", r.URL.Path, scope, ok)
		}
		scopes = append(scopes, scope)
		body := `{"groups":[{"displayName":"Gemini Models","buckets":[{"bucketId":"gemini-5h","window":"5h","remainingFraction":0.5}]}]}`
		if r.URL.Path == "/v1internal:fetchAvailableModels" {
			body = `{"models":{"gemini-test":{"displayName":"Gemini"}},"agentModelSorts":[{"groups":[{"modelIds":["gemini-test"]}]}]}`
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	if _, err := manager.ReadQuota(context.Background(), item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ReadAccountModels(context.Background(), item.ID); err != nil {
		t.Fatal(err)
	}
	if len(scopes) != 2 {
		t.Fatalf("scoped requests = %d, want 2", len(scopes))
	}
}

// 令牌续期同样属于该提供商的出站流量。
func TestTokenRefreshCarriesProviderScope(t *testing.T) {
	manager, _ := setupService(t)
	cred := quotaCredential()
	cred.ExpiresAt = time.Now().Add(-time.Minute)
	item, err := manager.accounts.SaveGoogleSubscription(context.Background(), "builtin_google", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	var scope netproxy.Scope
	manager.client = antigravity.New(roundTrip(func(r *http.Request) (*http.Response, error) {
		scope, _ = netproxy.ScopeFrom(r.Context())
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"access_token":"new-access","refresh_token":"new-refresh","expires_in":3600}`))}, nil
	}))
	if err := manager.Use(context.Background(), item.ID, func(antigravity.Credentials) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if scope.ProviderID != item.ProviderID {
		t.Fatalf("refresh scope = %+v, want provider %q", scope, item.ProviderID)
	}
}
