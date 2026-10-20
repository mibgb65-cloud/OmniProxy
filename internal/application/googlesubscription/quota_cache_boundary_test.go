package googlesubscription

import (
	"context"
	"net/http"
	"testing"
	"time"

	"omniproxy/internal/provider/antigravity"
)

func TestGoogleCachedQuotaDoesNotWaitForNetworkRefreshLock(t *testing.T) {
	ctx := context.Background()
	s, item, version, value := cachedQuotaFixture(t)
	if _, err := s.quotaCache.Save(ctx, item, version, value); err != nil {
		t.Fatal(err)
	}
	s.ops.Lock()
	defer s.ops.Unlock()
	done := make(chan error, 1)
	go func() {
		result, err := s.CachedQuota(ctx, item.ID)
		if err == nil && result == nil {
			err = antigravity.Failure("quota_cache_unavailable")
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("local cache waited for a network operation")
	}
}

func TestGoogleFailedQuotaReadPreservesPersistedSnapshot(t *testing.T) {
	s, item, version, value := cachedQuotaFixture(t)
	if _, err := s.quotaCache.Save(context.Background(), item, version, value); err != nil {
		t.Fatal(err)
	}
	calls := 0
	s.client = antigravity.New(roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 503, Header: http.Header{}, Body: http.NoBody}, nil
	}))
	if _, err := s.ReadQuota(context.Background(), item.ID); err == nil || calls != 1 {
		t.Fatal("failed upstream read was hidden", err)
	}
	got, err := s.CachedQuota(context.Background(), item.ID)
	if err != nil || got == nil || *got.Groups[0].Buckets[0].RemainingPercent != 76.25 {
		t.Fatal("failed read erased cache", got, err)
	}
}
