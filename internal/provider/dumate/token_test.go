package dumate

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokenCacheMergesConcurrentRefreshes(t *testing.T) {
	cache := NewTokenCache()
	var calls int32
	fetch := func(context.Context) (Token, error) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(10 * time.Millisecond)
		return Token{Value: "shared", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}
	var wait sync.WaitGroup
	results := make(chan string, 8)
	for i := 0; i < 8; i++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			token, err := cache.Get(context.Background(), "key", fetch)
			if err != nil {
				t.Error(err)
				return
			}
			results <- token.Value
		}()
	}
	wait.Wait()
	close(results)
	for value := range results {
		if value != "shared" {
			t.Fatalf("token = %q", value)
		}
	}
	if calls != 1 {
		t.Fatalf("fetch calls = %d", calls)
	}
}

func TestTokenCacheRefreshesNearExpiryAndInvalidates(t *testing.T) {
	cache := NewTokenCache()
	now := time.Now()
	cache.now = func() time.Time { return now }
	var calls int32
	fetch := func(context.Context) (Token, error) {
		atomic.AddInt32(&calls, 1)
		return Token{Value: "token", ExpiresAt: now.Add(time.Hour)}, nil
	}
	if _, err := cache.Get(context.Background(), "key", fetch); err != nil {
		t.Fatal(err)
	}
	if _, err := cache.Get(context.Background(), "key", fetch); err != nil || calls != 1 {
		t.Fatalf("cached token = %v calls=%d", err, calls)
	}
	cache.now = func() time.Time { return now.Add(time.Hour - time.Minute) }
	if _, err := cache.Get(context.Background(), "key", fetch); err != nil || calls != 2 {
		t.Fatalf("near-expiry token not refreshed: calls=%d", calls)
	}
	cache.Invalidate("key")
	if _, err := cache.Get(context.Background(), "key", fetch); err != nil || calls != 3 {
		t.Fatalf("invalidate did not refresh: calls=%d", calls)
	}
}

func TestTokenCacheDropsFailedRefreshAndCapsEntries(t *testing.T) {
	cache := NewTokenCache()
	if _, err := cache.Get(context.Background(), "key", func(context.Context) (Token, error) {
		return Token{}, SignInFailure()
	}); !SignInExpired(err) {
		t.Fatalf("failure = %v", err)
	}
	if len(cache.entries) != 0 {
		t.Fatal("failed refresh kept an entry")
	}
	fetch := func(context.Context) (Token, error) {
		return Token{Value: "v", ExpiresAt: time.Now().Add(time.Hour)}, nil
	}
	for i := 0; i < maxTokenEntries+40; i++ {
		if _, err := cache.Get(context.Background(), string(rune('a'+i%26))+string(rune('a'+i/26))+"key", fetch); err != nil {
			t.Fatal(err)
		}
	}
	if len(cache.entries) > maxTokenEntries {
		t.Fatalf("cache entries = %d", len(cache.entries))
	}
}
