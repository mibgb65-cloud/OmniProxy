package gateway

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/chatgpt"
)

type balanceSubscription struct {
	*quotaAwareSubscription
	weights map[string]int
}

func (s balanceSubscription) QuotaBalanceWeight(id, version string, now time.Time) (int, bool) {
	weight, ok := s.weights[id]
	return weight, ok
}

func TestAccountRotationChangesActualHTTPAndWebSocketRequests(t *testing.T) {
	for _, transport := range []string{"http", "websocket"} {
		t.Run(transport, func(t *testing.T) {
			s, db, _, access := subscriptionQuotaFixture(t)
			settings := NewCodexSettingsService(db)
			s.SetAccountRotationMode(settings.AccountRotationMode)
			var seen []string
			s.chatgpt = chatgpt.New(roundTripFunc(func(r *http.Request) (*http.Response, error) {
				seen = append(seen, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer placeholder-subscription-"))
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(missingTypeStream))}, nil
			}))
			send := func(model string) {
				t.Helper()
				if ok, _, _ := requestQuotaGateway(t, s, transport, model); !ok {
					t.Fatal("request failed")
				}
			}
			send("gpt-test")
			send("gpt-test")
			if strings.Join(seen, ",") != "a,a" {
				t.Fatal("default queue behavior changed", seen)
			}
			if _, err := settings.SetAccountRotationMode(context.Background(), "balanced"); err != nil {
				t.Fatal(err)
			}
			seen = nil
			for range 4 {
				send("gpt-test")
			}
			if strings.Join(seen, ",") != "a,b,a,b" {
				t.Fatal("balanced mode did not distribute real requests", seen)
			}
			access.mu.Lock()
			access.blocked["a"] = time.Now().Add(time.Hour)
			access.mu.Unlock()
			seen = nil
			send("gpt-test")
			send("gpt-test")
			if strings.Join(seen, ",") != "b,b" {
				t.Fatal("balanced mode used exhausted account", seen)
			}
			if _, err := db.Exec(`INSERT INTO model_routes(id,alias,provider_id,model,account_id,enabled,created_at,updated_at) VALUES('balance-fixed','fixed','builtin_openai','gpt-test','a',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`); err != nil {
				t.Fatal(err)
			}
			seen = nil
			send("fixed")
			if strings.Join(seen, ",") != "a" {
				t.Fatal("balanced mode bypassed fixed account")
			}
		})
	}
}

func TestBalancedRotationUsesFreshWeightsAndFallsBackForUnknownQuota(t *testing.T) {
	for _, scenario := range []struct {
		name    string
		weights map[string]int
		wantA   int
	}{
		{"weighted", map[string]int{"a": 80, "b": 20}, 8},
		{"one-unknown", map[string]int{"a": 80}, 5},
		{"invalid-weight", map[string]int{"a": 200, "b": 20}, 5},
		{"zero-weight", map[string]int{"a": 0, "b": 80}, 0},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			s, db, _, access := subscriptionQuotaFixture(t)
			settings := NewCodexSettingsService(db)
			if _, err := settings.SetAccountRotationMode(context.Background(), "balanced"); err != nil {
				t.Fatal(err)
			}
			s.SetAccountRotationMode(settings.AccountRotationMode)
			s.SetSubscription(balanceSubscription{access, scenario.weights})
			countA, countB := 0, 0
			s.chatgpt = chatgpt.New(roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if strings.HasSuffix(r.Header.Get("Authorization"), "-a") {
					countA++
				} else {
					countB++
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(missingTypeStream))}, nil
			}))
			for range 10 {
				if ok, _, _ := requestQuotaGateway(t, s, "http", "gpt-test"); !ok {
					t.Fatal("weighted request failed")
				}
			}
			if countA != scenario.wantA || countA+countB != 10 {
				t.Fatalf("unexpected dispatch distribution: %d/%d", countA, countB)
			}
		})
	}
}

func TestRotationReservationsAreBalancedUnderConcurrency(t *testing.T) {
	var state rotationState
	candidates := []proxy.AccountCandidate{{AccountID: "a", UpdatedAt: "v1"}, {AccountID: "b", UpdatedAt: "v1"}}
	counts := [2]int{}
	var mu sync.Mutex
	var group sync.WaitGroup
	for range 100 {
		group.Go(func() {
			offset := state.choose(candidates, []int{0, 1}, []int{1, 1})
			mu.Lock()
			counts[offset]++
			mu.Unlock()
		})
	}
	group.Wait()
	if counts != [2]int{50, 50} {
		t.Fatal("concurrent selection concentrated on one account", counts)
	}
}

func TestRotationRetainsLargePoolScoresAndBoundsRetiredVersions(t *testing.T) {
	var state rotationState
	candidates := make([]proxy.AccountCandidate, 3000)
	eligible, weights := make([]int, len(candidates)), make([]int, len(candidates))
	for index := range candidates {
		candidates[index] = proxy.AccountCandidate{AccountID: fmt.Sprintf("a-%d", index), UpdatedAt: "v1"}
		eligible[index], weights[index] = index, 1
	}
	for want := range 3 {
		if got := state.choose(candidates, eligible, weights); got != want {
			t.Fatalf("large existing pool reset its rotation: got %d want %d", got, want)
		}
	}
	for index := range candidates {
		candidates[index].UpdatedAt = "v2"
	}
	state.choose(candidates, eligible, weights)
	if len(state.scores) != len(candidates) {
		t.Fatal("retired versions exceeded bounded rotation state")
	}
}
