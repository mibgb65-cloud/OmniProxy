package gateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/chatgpt"
)

func requestForRotation() proxy.Request {
	return proxy.Request{Model: "gpt-test", Protocol: "responses", Stream: true, Body: []byte(`{"model":"gpt-test","stream":true,"input":[]}`)}
}

func TestOpenAIRotationDoesNotChangeOtherProviders(t *testing.T) {
	s := &Service{rotationMode: func(context.Context) (string, error) {
		t.Fatal("non-OpenAI request read OpenAI settings")
		return "", nil
	}}
	request := requestForRotation()
	request.Candidates = []proxy.AccountCandidate{{AccountID: "a"}, {AccountID: "b"}}
	candidates, err := s.rotateAccounts(context.Background(), target{kind: "deepseek"}, request)
	if err != nil || len(candidates) != 2 || candidates[0].AccountID != "a" {
		t.Fatal("another provider was rotated")
	}
}

func TestBalancedRotationPreservesRetryAndConfigurationBoundaries(t *testing.T) {
	for _, scenario := range []string{"quota-rejection", "already-started", "network-uncertain", "settings-failed", "other-provider", "stale-version", "all-exhausted", "all-exhausted-no-retry"} {
		t.Run(scenario, func(t *testing.T) {
			s, db, _, access := subscriptionQuotaFixture(t)
			settings := NewCodexSettingsService(db)
			if _, err := settings.SetAccountRotationMode(context.Background(), "balanced"); err != nil {
				t.Fatal(err)
			}
			s.SetAccountRotationMode(settings.AccountRotationMode)
			calls := []string{}
			s.chatgpt = chatgpt.New(roundTripFunc(func(r *http.Request) (*http.Response, error) {
				id := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer placeholder-subscription-")
				calls = append(calls, id)
				if scenario == "network-uncertain" {
					return nil, errors.New("private-network-detail")
				}
				body := missingTypeStream
				if scenario == "quota-rejection" && id == "a" {
					body = fmt.Sprintf(`{"error":{"type":"usage_limit_reached","resets_at":%d}}`, time.Now().Add(time.Hour).Unix())
				}
				if scenario == "already-started" {
					body = "data: {\"type\":\"response.created\",\"response\":{\"id\":\"started\"}}\n\ndata: {\"type\":\"error\",\"error\":{\"type\":\"usage_limit_reached\"}}\n\n"
				}
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
			}))
			if scenario == "settings-failed" {
				s.SetAccountRotationMode(func(context.Context) (string, error) { return "", errors.New("private-settings-detail") })
			}
			if strings.HasPrefix(scenario, "all-exhausted") {
				access.mu.Lock()
				access.blocked["a"], access.blocked["b"] = time.Now().Add(time.Hour), time.Now().Add(time.Hour)
				access.mu.Unlock()
				if scenario == "all-exhausted-no-retry" {
					if _, err := NewFailoverSettingsService(db).Set(context.Background(), false); err != nil {
						t.Fatal(err)
					}
				}
			}
			if scenario == "stale-version" || scenario == "other-provider" {
				request, err := s.RouteRequest(context.Background(), requestForRotation())
				if err != nil {
					t.Fatal(err)
				}
				initial, err := s.initialTarget(context.Background(), request)
				if err != nil {
					t.Fatal(err)
				}
				if scenario == "stale-version" {
					request.Candidates[1].UpdatedAt = "stale"
				} else {
					if _, err := db.Exec(`UPDATE accounts SET provider_id='builtin_deepseek' WHERE id='b'`); err != nil {
						t.Fatal(err)
					}
				}
				if _, err := s.rotateAccounts(context.Background(), initial, request); err == nil {
					t.Fatal("invalid candidate silently selected")
				}
				return
			}
			completed, status, raw := requestQuotaGateway(t, s, "http", "gpt-test")
			if strings.Contains(raw, "private-") {
				t.Fatal("private error leaked")
			}
			if scenario == "quota-rejection" {
				if !completed || strings.Join(calls, ",") != "a,b" {
					t.Fatal("confirmed pre-execution rejection did not switch once", calls)
				}
			} else if strings.HasPrefix(scenario, "all-exhausted") {
				if completed || status != 429 || len(calls) != 0 {
					t.Fatal("exhausted pool sent a probe")
				}
			} else if scenario == "settings-failed" {
				if completed || status != 503 || len(calls) != 0 {
					t.Fatal("invalid settings sent a request")
				}
			} else if completed || strings.Join(calls, ",") != "a" {
				t.Fatal("uncertain or started generation was replayed", calls)
			}
		})
	}
}

func TestBalancedRotationDoesNotMultiplySameSubscriptionIdentity(t *testing.T) {
	s, db, _, _ := subscriptionQuotaFixture(t)
	if _, err := db.Exec(`UPDATE accounts SET enabled=1,auth_email='b@example.invalid',auth_account_id='remote-b' WHERE id='c'`); err != nil {
		t.Fatal(err)
	}
	settings := NewCodexSettingsService(db)
	if _, err := settings.SetAccountRotationMode(context.Background(), "balanced"); err != nil {
		t.Fatal(err)
	}
	s.SetAccountRotationMode(settings.AccountRotationMode)
	for range 6 {
		request, err := s.RouteRequest(context.Background(), requestForRotation())
		if err != nil {
			t.Fatal(err)
		}
		initial, err := s.initialTarget(context.Background(), request)
		if err != nil {
			t.Fatal(err)
		}
		candidates, err := s.rotateAccounts(context.Background(), initial, request)
		if err != nil || len(candidates) != 2 {
			t.Fatal("same subscription was treated as separate quota", candidates, err)
		}
		for _, candidate := range candidates {
			if candidate.AccountID == "c" {
				t.Fatal("duplicate identity remained as a backup")
			}
		}
	}
}
