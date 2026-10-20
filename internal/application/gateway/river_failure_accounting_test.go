package gateway

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/application/activity"
)

func TestRiverServerFailureRecordsFailedAttemptWithoutReplay(t *testing.T) {
	calls := 0
	s, db, _ := riverPoolFixture(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return upstreamReply(503, `{"error":{}}`, false), nil
	}))
	response, err := http.Post(s.Info().Runtime.Address+"/v1/chat/completions", "application/json",
		strings.NewReader(`{"model":"river-test","messages":[{"role":"user","content":"test"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		t.Fatal(err)
	}
	item, err := activity.New(db).Request(context.Background(), response.Header.Get("X-Request-ID"))
	if err != nil || calls != 1 || len(item.Attempts) != 1 || item.Attempts[0].StatusCode != 503 || item.Attempts[0].State != "failed" {
		t.Fatal("failed River attempt was marked successful or replayed", item, err)
	}
}

func TestRiverConsecutiveFailuresResetAfterSuccessAndIgnoreOldConcurrentSuccess(t *testing.T) {
	state := newFailoverState()
	now := time.Now()
	state.now = func() time.Time { return now }
	target := target{accountID: "a", updatedAt: "v1"}
	first, _, _ := state.acquire(target, "model")
	state.finishResult(first, nil, false, true)
	healthy, _, _ := state.acquire(target, "model")
	state.finishResult(healthy, nil, true, false)
	for range 2 {
		lease, _, _ := state.acquire(target, "model")
		state.finishResult(lease, nil, false, true)
	}
	if state.wait("a", "v1", "model") <= 0 {
		t.Fatal("two consecutive failures did not cool key")
	}
	now = now.Add(31 * time.Second)
	probe, available, _ := state.acquire(target, "model")
	if !available {
		t.Fatal("recovery probe unavailable")
	}
	state.finishResult(probe, nil, true, false)
	old, _, _ := state.acquire(target, "model")
	for range 2 {
		lease, _, _ := state.acquire(target, "model")
		state.finishResult(lease, nil, false, true)
	}
	state.finishResult(old, nil, true, false)
	if state.wait("a", "v1", "model") <= 0 {
		t.Fatal("an older concurrent success cleared a later cooldown")
	}
}
