package gateway

import (
	"errors"
	"testing"
	"time"

	"omniproxy/internal/core/proxy"
)

func TestRefusedBeforeRecognizesTheSkipSignal(t *testing.T) {
	if _, ok := refusedBefore(proxy.Failure("prompt_refused_before", 400)); !ok {
		t.Fatal("skip signal was not recognized")
	}
	if _, ok := refusedBefore(proxy.Failure("prompt_turned_away", 400)); ok {
		t.Fatal("a plain refusal was treated as a skip")
	}
	if _, ok := refusedBefore(errors.New("boom")); ok {
		t.Fatal("unrelated error was treated as a skip")
	}
}

func TestQuotaRetryDelayUsesTheUpstreamReset(t *testing.T) {
	now := time.Date(2026, 11, 12, 0, 0, 0, 0, time.UTC)
	failure := &proxy.Error{Code: "upstream_quota_exhausted", StatusCode: 429, RetryAfter: "3600"}
	if delay := quotaRetryDelay(failure, now); delay != time.Hour {
		t.Fatalf("delay = %s, want 1h", delay)
	}
	failure.RetryAfter = ""
	if delay := quotaRetryDelay(failure, now); delay != 30*time.Second {
		t.Fatalf("fallback delay = %s, want 30s", delay)
	}
	// 超出阻断窗口的重置时间不当作依据。
	failure.RetryAfter = "999999"
	if delay := quotaRetryDelay(failure, now); delay != 30*time.Second {
		t.Fatalf("out-of-window delay = %s, want 30s", delay)
	}
}

func TestHasLaterCandidateSkipsSeenAccounts(t *testing.T) {
	later := []proxy.AccountCandidate{{AccountID: "b"}, {AccountID: "c"}}
	if !hasLaterCandidate(later, map[string]bool{"a": true}) {
		t.Fatal("unseen candidate reported as seen")
	}
	if hasLaterCandidate(later, map[string]bool{"a": true, "b": true, "c": true}) {
		t.Fatal("seen candidates counted as later")
	}
}
