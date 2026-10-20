package gateway

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/river"
)

func riverPoolFixture(t *testing.T, transport http.RoundTripper) (*Service, *sql.DB, *routes.Service) {
	t.Helper()
	s, db, _, router := failoverFixture(t, transport)
	if _, err := db.Exec(`DELETE FROM accounts WHERE id IN ('c','d'); UPDATE accounts SET provider_id='builtin_river'`); err != nil {
		t.Fatal(err)
	}
	if _, err := router.Create(context.Background(), routes.Input{Alias: "river-test", ProviderID: "builtin_river", Model: river.DeepSeekFlash, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	s.river = river.New(transport)
	s.SetRiverAccountRotationMode(NewRiverSettingsService(db).AccountRotationMode)
	return s, db, router
}

func riverRequest(t *testing.T, s *Service, model string) int {
	t.Helper()
	response, err := http.Post(s.Info().Runtime.Address+"/v1/chat/completions", "application/json",
		strings.NewReader(`{"model":"`+model+`","messages":[{"role":"user","content":"test"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		t.Fatal(err)
	}
	return response.StatusCode
}

func TestRiverQueueAndBalancedPersistAndDistributeActualRequests(t *testing.T) {
	var calls []string
	s, db, _ := riverPoolFixture(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls = append(calls, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer placeholder-failover-"))
		return upstreamReply(200, atriaJSON("chat"), false), nil
	}))
	for range 2 {
		if riverRequest(t, s, "river-test") != 200 {
			t.Fatal("queue request failed")
		}
	}
	if strings.Join(calls, ",") != "a,a" {
		t.Fatal("queue changed", calls)
	}
	settings := NewRiverSettingsService(db)
	if _, err := settings.SetAccountRotationMode(context.Background(), "balanced"); err != nil {
		t.Fatal(err)
	}
	if mode, err := NewRiverSettingsService(db).AccountRotationMode(context.Background()); err != nil || mode != "balanced" {
		t.Fatal("mode not persisted", mode, err)
	}
	calls = nil
	for range 4 {
		if riverRequest(t, s, "river-test") != 200 {
			t.Fatal("balanced request failed")
		}
	}
	if strings.Join(calls, ",") != "a,b,a,b" {
		t.Fatal("not balanced", calls)
	}
	for _, mode := range []string{"", "BALANCED", "balanced\n", strings.Repeat("x", 1024)} {
		if _, err := settings.SetAccountRotationMode(context.Background(), mode); err == nil {
			t.Fatal("invalid mode accepted")
		}
	}
	if _, err := db.Exec(`DELETE FROM river_settings`); err != nil {
		t.Fatal(err)
	}
	before := len(calls)
	if riverRequest(t, s, "river-test") != 503 || len(calls) != before {
		t.Fatal("missing settings sent a request")
	}
}

func TestRiverRepeatedFailuresOnlySwitchSubsequentRequests(t *testing.T) {
	for _, mode := range []string{"network", "500", "503"} {
		t.Run(mode, func(t *testing.T) {
			var calls []string
			failed := 0
			s, _, _ := riverPoolFixture(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer placeholder-failover-")
				calls = append(calls, key)
				if key == "a" && failed < 2 {
					failed++
					if mode == "network" {
						return nil, errors.New("private-network-marker")
					}
					status := 500
					if mode == "503" {
						status = 503
					}
					return upstreamReply(status, `{"error":{}}`, false), nil
				}
				return upstreamReply(200, atriaJSON("chat"), false), nil
			}))
			now := time.Now()
			s.failover.now = func() time.Time { return now }
			for range 2 {
				if riverRequest(t, s, "river-test") < 500 {
					t.Fatal("uncertain request was replayed")
				}
			}
			if len(calls) != 2 || len(s.FailoverHealth()) != 1 {
				t.Fatal("key not cooled after repeated failure", calls)
			}
			if riverRequest(t, s, "river-test") != 200 || strings.Join(calls, ",") != "a,a,b" {
				t.Fatal("failed key remained sticky", calls)
			}
			now = now.Add(31 * time.Second)
			if riverRequest(t, s, "river-test") != 200 || strings.Join(calls, ",") != "a,a,b,a" || len(s.FailoverHealth()) != 0 {
				t.Fatal("recovery did not clear cooldown", calls)
			}
		})
	}
}

func TestRiverExplicitRejectionFixedAndDisabledBoundaries(t *testing.T) {
	for _, mode := range []string{"401", "402", "429", "fixed", "disabled", "400", "403"} {
		t.Run(mode, func(t *testing.T) {
			var calls []string
			s, db, router := riverPoolFixture(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer placeholder-failover-")
				calls = append(calls, key)
				status := 401
				switch mode {
				case "402":
					status = 402
				case "429":
					status = 429
				case "400":
					status = 400
				case "403":
					status = 403
				}
				if key == "a" {
					return upstreamReply(status, `{"error":{}}`, false), nil
				}
				return upstreamReply(200, atriaJSON("chat"), false), nil
			}))
			if mode == "fixed" {
				if _, err := router.Create(context.Background(), routes.Input{Alias: "river-fixed", ProviderID: "builtin_river", Model: river.DeepSeekFlash, AccountID: "a", Enabled: true}); err != nil {
					t.Fatal(err)
				}
				if riverRequest(t, s, "river-fixed") != 401 || strings.Join(calls, ",") != "a" {
					t.Fatal("fixed account switched", calls)
				}
				return
			}
			if mode == "disabled" {
				if _, err := NewFailoverSettingsService(db).Set(context.Background(), false); err != nil {
					t.Fatal(err)
				}
			}
			status := riverRequest(t, s, "river-test")
			shouldSwitch := mode == "401" || mode == "402" || mode == "429"
			if shouldSwitch && (status != 200 || strings.Join(calls, ",") != "a,b") || !shouldSwitch && len(calls) != 1 {
				t.Fatal("unsafe switching boundary", status, calls)
			}
		})
	}
}

func TestRiverTransientFailureDoesNotCountCancellationOrClientErrors(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if riverTransientFailure(ctx, 503, proxy.Failure("upstream_unavailable", 502)) || riverTransientFailure(context.Background(), 400, nil) {
		t.Fatal("client cancellation or bad request counted as key failure")
	}
}
