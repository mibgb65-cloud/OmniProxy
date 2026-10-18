package googlesubscription

import (
	"context"
	"omniproxy/internal/provider/antigravity"
	"testing"
	"time"
)

func TestUseAtRejectsChangedAccountBeforeRefresh(t *testing.T) {
	manager, refreshes := setupService(t)
	cred := antigravity.Credentials{OAuthConfig: clientConfig(), AccessToken: "placeholder-access", RefreshToken: "placeholder-refresh", ExpiresAt: time.Now().Add(-time.Minute), Subject: "1001", Email: "person@example.test", Project: "test-project", Plan: "test-tier"}
	item, err := manager.accounts.SaveGoogleSubscription(context.Background(), "builtin_google", "", "", cred)
	if err != nil {
		t.Fatal(err)
	}
	called := false
	err = manager.UseAt(context.Background(), item.ID, "stale-version", func(antigravity.Credentials) error { called = true; return nil })
	if err == nil || called || refreshes.Load() != 0 {
		t.Fatal("stale account used or refreshed", err)
	}
}
