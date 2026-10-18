package gateway

import (
	"omniproxy/internal/core/proxy"
	"testing"
	"time"
)

func TestFailoverCoolingRecoveryAndConcurrentSuccess(t *testing.T) {
	s := newFailoverState()
	now := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	target := target{accountID: "a", updatedAt: "v1", snapshot: proxy.TargetSnapshot{AccountID: "a"}}
	first, ok, _ := s.acquire(target, "model")
	if !ok {
		t.Fatal("healthy rejected")
	}
	concurrent, ok, _ := s.acquire(target, "model")
	if !ok {
		t.Fatal("healthy serialized")
	}
	s.finish(first, &retryReason{code: "upstream_rate_limited", delay: 30 * time.Second}, false)
	s.finish(concurrent, nil, true)
	if _, ok, _ = s.acquire(target, "model"); ok {
		t.Fatal("stale success cleared new failure")
	}
	other, ok, _ := s.acquire(target, "other")
	if !ok {
		t.Fatal("unrelated model cooled")
	}
	s.finish(other, nil, true)
	now = now.Add(31 * time.Second)
	probe, ok, _ := s.acquire(target, "model")
	if !ok {
		t.Fatal("recovery blocked")
	}
	if _, ok, _ = s.acquire(target, "model"); ok {
		t.Fatal("multiple probes admitted")
	}
	s.finish(probe, nil, false)
	probe, ok, _ = s.acquire(target, "model")
	if !ok {
		t.Fatal("cancelled probe leaked lease")
	}
	s.finish(probe, nil, true)
	if len(s.entries) != 0 {
		t.Fatal("successful recovery not cleared")
	}
}

func TestFailoverRetryAfterValidation(t *testing.T) {
	now := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		raw     string
		seconds int
		ok      bool
	}{
		{"30", 30, true}, {"0", 0, true}, {"Sun, 18 Oct 2026 00:01:00 GMT", 60, true}, {"-1", 0, false}, {"+1", 0, false}, {"1.5", 0, false}, {"86401", 0, false}, {"99999999999999999999999999", 0, false}, {"Sat, 17 Oct 2026 00:00:00 GMT", 0, false}, {"invalid", 0, false},
	} {
		delay, ok := retryDelay(tc.raw, now)
		if ok != tc.ok || ok && int(delay.Seconds()) != tc.seconds {
			t.Errorf("retry-after %q: %v %v", tc.raw, delay, ok)
		}
	}
	google := target{kind: "google"}
	if rejectedAccount(google, proxy.Response{StatusCode: 429}, now) != nil {
		t.Fatal("project quota treated as independent keys")
	}
	for _, status := range []int{400, 403, 500, 502, 503, 504} {
		if rejectedAccount(target{kind: "deepseek"}, proxy.Response{StatusCode: status}, now) != nil {
			t.Fatal("unknown failure retried", status)
		}
	}
}
