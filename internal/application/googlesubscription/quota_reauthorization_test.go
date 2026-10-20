package googlesubscription

import (
	"context"
	"testing"
)

func TestGoogleNewAuthorizationCanReplaceOlderScopeDespiteClockChange(t *testing.T) {
	ctx := context.Background()
	s, item, version, value := cachedQuotaFixture(t)
	if _, err := s.quotaCache.Save(ctx, item, version, value); err != nil {
		t.Fatal(err)
	}
	current, err := s.accounts.SaveGoogleSubscription(ctx, item.ProviderID, item.ID, "", quotaCredential())
	if err != nil {
		t.Fatal(err)
	}
	nextVersion, err := s.accounts.CatalogVersion(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	// 防止旧授权的时间戳（包括系统时钟回拨）永久阻挡当前授权的合法快照。
	value.FetchedAt = "2026-10-20T11:00:00Z"
	if _, err := s.quotaCache.Save(ctx, current, nextVersion, value); err != nil {
		t.Fatal("new authorization blocked by old scope", err)
	}
	if _, err := s.quotaCache.Save(ctx, item, version, value); err == nil {
		t.Fatal("old authorization wrote after relogin")
	}
	got, err := s.CachedQuota(ctx, item.ID)
	if err != nil || got == nil {
		t.Fatal("new scope not restored", got, err)
	}
}
