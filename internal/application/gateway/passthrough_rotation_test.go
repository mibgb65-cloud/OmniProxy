package gateway

import (
	"context"
	"errors"
	"testing"
	"time"

	"omniproxy/internal/core/proxy"
)

// fallbackFirst 返回兜底会话实际排在第一位的账号，避免测试依赖内部轮询状态。
func fallbackFirst(t *testing.T, s *Service) string {
	t.Helper()
	session, err := s.openFallbackSession(context.Background(), "", "", "https://example.invalid")
	if err != nil {
		t.Fatal(err)
	}
	state, ok := session.(*passthroughSession)
	if !ok || len(state.request.Candidates) == 0 {
		t.Fatal("fallback session did not retain candidates")
	}
	return state.request.Candidates[0].AccountID
}

func TestFallbackSessionHonorsRotationMode(t *testing.T) {
	s, db, _, access := subscriptionQuotaFixture(t)
	settings := NewCodexSettingsService(db)
	s.SetAccountRotationMode(settings.AccountRotationMode)

	// 默认 queue：保持 accounts.List 的稳定顺序（created_at DESC, id DESC）。
	for range 4 {
		if got := fallbackFirst(t, s); got != "b" {
			t.Fatalf("queue fallback changed order: %s", got)
		}
	}

	if _, err := settings.SetAccountRotationMode(context.Background(), "balanced"); err != nil {
		t.Fatal(err)
	}
	seen := make([]string, 0, 4)
	for range 4 {
		seen = append(seen, fallbackFirst(t, s))
	}
	if want := []string{"b", "a", "b", "a"}; !equalStrings(seen, want) {
		t.Fatalf("balanced fallback did not distribute: %v", seen)
	}

	access.mu.Lock()
	access.blocked["b"] = time.Now().Add(time.Hour)
	access.mu.Unlock()
	if got := fallbackFirst(t, s); got != "a" {
		t.Fatalf("fallback used quota-blocked account: %s", got)
	}
}

func TestFallbackSessionKeepsOrderWhenRotationUnavailable(t *testing.T) {
	s, db, _, _ := subscriptionQuotaFixture(t)
	s.SetAccountRotationMode(func(context.Context) (string, error) {
		return "", errors.New("private-database-detail")
	})
	for range 3 {
		if got := fallbackFirst(t, s); got != "b" {
			t.Fatalf("fallback did not preserve original order: %s", got)
		}
	}
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	if got := fallbackFirst(t, s); got != "b" {
		t.Fatalf("fallback lost its only enabled account: %s", got)
	}
}

func TestFallbackSessionRejectsCoolingPool(t *testing.T) {
	s, db, _, access := subscriptionQuotaFixture(t)
	settings := NewCodexSettingsService(db)
	if _, err := settings.SetAccountRotationMode(context.Background(), "balanced"); err != nil {
		t.Fatal(err)
	}
	s.SetAccountRotationMode(settings.AccountRotationMode)
	access.mu.Lock()
	access.blocked["a"] = time.Now().Add(time.Hour)
	access.blocked["b"] = time.Now().Add(time.Hour)
	access.mu.Unlock()
	if _, err := s.openFallbackSession(context.Background(), "", "", "https://example.invalid"); err == nil {
		t.Fatal("cooling pool was accepted")
	} else {
		var failure *proxy.Error
		if !errors.As(err, &failure) || failure.Code != "account_pool_cooling" {
			t.Fatalf("unexpected fallback error: %v", err)
		}
	}
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
