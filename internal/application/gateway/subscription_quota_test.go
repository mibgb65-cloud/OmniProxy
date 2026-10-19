package gateway

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/provider/chatgpt"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSubscriptionQuotaPreExecutionRejectionSwitchesBothTransports(t *testing.T) {
	for _, transport := range []string{"http", "websocket"} {
		for _, shape := range []string{"json", "sse", "flat", "explicit-sse", "large-first-event", "cached-five-hour", "cached-weekly"} {
			t.Run(transport+"/"+shape, func(t *testing.T) {
				s, db, observer, access := subscriptionQuotaFixture(t)
				_ = db
				reset := time.Now().Add(7 * 24 * time.Hour)
				if strings.HasPrefix(shape, "cached-") {
					access.blocked["a"] = reset
				}
				var mu sync.Mutex
				calls := []string{}
				s.chatgpt = chatgpt.New(roundTripFunc(func(r *http.Request) (*http.Response, error) {
					id := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer placeholder-subscription-")
					mu.Lock()
					calls = append(calls, id)
					mu.Unlock()
					if id == "a" && shape != "large-first-event" {
						raw := fmt.Sprintf(`{"error":{"type":"usage_limit_reached","message":"private-quota-marker","resets_at":%d}}`, reset.Unix())
						kind := ""
						if shape == "sse" || shape == "explicit-sse" {
							raw = "data: " + raw + "\n\n"
						}
						if shape == "flat" {
							raw = "data: {\"type\":\"error\",\"code\":\"usage_limit_reached\"}\n\n"
						}
						if shape == "explicit-sse" {
							kind = "text/event-stream"
						}
						return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {kind}}, Body: io.NopCloser(strings.NewReader(raw))}, nil
					}
					body := missingTypeStream
					if shape == "large-first-event" {
						body = "data: {\"type\":\"response.created\",\"response\":{\"id\":\"one\",\"tools\":[{\"description\":\"" + strings.Repeat("x", 600<<10) + "\"}]}}\n\n" + missingTypeStream
					}
					return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
				}))
				completed, _, _ := requestQuotaGateway(t, s, transport, "gpt-test")
				if !completed {
					t.Fatal("quota or large first event prevented completion")
				}
				page, err := observer.Requests(context.Background(), activity.RequestQuery{Limit: 10})
				if err != nil || len(page.Items) != 1 {
					t.Fatal("logical request lost", err)
				}
				item, err := observer.Request(context.Background(), page.Items[0].ID)
				want := 2
				if strings.HasPrefix(shape, "cached-") || shape == "large-first-event" {
					want = 1
				}
				if err != nil || item.AttemptCount != want || item.State != "success" || item.InputTokens != "12" || item.OutputTokens != "4" {
					t.Fatal("wrong logical accounting", item, err)
				}
				if want == 2 && (item.Attempts[0].StatusCode != 200 || item.Attempts[0].ErrorCode != "upstream_quota_exhausted" || item.AccountID != "b") {
					t.Fatal("wire rejection or successful account lost", item)
				}
				mu.Lock()
				defer mu.Unlock()
				expected := "a,b"
				if strings.HasPrefix(shape, "cached-") {
					expected = "b"
				}
				if shape == "large-first-event" {
					expected = "a"
				}
				if strings.Join(calls, ",") != expected {
					t.Fatal("wrong candidate sequence", calls)
				}
			})
		}
	}
}

func TestSubscriptionQuotaDenialsAndStartedStreamsKeepRetryBoundary(t *testing.T) {
	for _, transport := range []string{"http", "websocket"} {
		for _, scenario := range []string{"all-exhausted", "already-started", "fixed", "disabled"} {
			t.Run(transport+"/"+scenario, func(t *testing.T) {
				s, db, observer, _ := subscriptionQuotaFixture(t)
				model := "gpt-test"
				if scenario == "fixed" {
					if _, err := db.Exec(`INSERT INTO model_routes(id,alias,provider_id,model,account_id,enabled,created_at,updated_at) VALUES('quota-fixed','fixed','builtin_openai','gpt-test','a',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`); err != nil {
						t.Fatal(err)
					}
					model = "fixed"
				}
				if scenario == "disabled" {
					if _, err := NewFailoverSettingsService(db).Set(context.Background(), false); err != nil {
						t.Fatal(err)
					}
				}
				calls := 0
				s.chatgpt = chatgpt.New(roundTripFunc(func(*http.Request) (*http.Response, error) {
					calls++
					raw := "data: {\"type\":\"error\",\"error\":{\"type\":\"usage_limit_reached\",\"message\":\"private-quota-marker\"}}\n\n"
					if scenario == "already-started" {
						raw = "data: {\"type\":\"response.created\",\"response\":{\"id\":\"one\"}}\n\n" + raw
					}
					return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(raw))}, nil
				}))
				completed, status, raw := requestQuotaGateway(t, s, transport, model)
				want := 1
				if scenario == "all-exhausted" {
					want = 2
				}
				if completed || calls != want {
					t.Fatal("unsafe repeated execution", scenario, calls)
				}
				if scenario != "already-started" && (status != 429 || !strings.Contains(raw, "upstream_quota_exhausted")) {
					t.Fatal("quota denial not actionable", status, raw)
				}
				page, err := observer.Requests(context.Background(), activity.RequestQuery{Limit: 10})
				if err != nil || len(page.Items) != 1 || page.Items[0].State != "failed" {
					t.Fatal("failed request counted incorrectly", page, err)
				}
			})
		}
	}
}

func TestSubscriptionQuotaRejectionIsSharedAcrossModels(t *testing.T) {
	s, db, _, _ := subscriptionQuotaFixture(t)
	if _, err := db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version) SELECT id,'gpt-alt','Alternate','2026-01-01T00:00:00Z',version FROM account_catalog_versions WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	calls := []string{}
	s.chatgpt = chatgpt.New(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		id := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer placeholder-subscription-")
		mu.Lock()
		calls = append(calls, id)
		mu.Unlock()
		body := missingTypeStream
		if id == "a" {
			body = fmt.Sprintf(`{"error":{"type":"usage_limit_reached","resets_at":%d}}`, time.Now().Add(time.Hour).Unix())
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	for _, model := range []string{"gpt-test", "gpt-alt"} {
		completed, _, _ := requestQuotaGateway(t, s, "http", model)
		if !completed {
			t.Fatal("backup failed", model)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(calls, ",") != "a,b,b" {
		t.Fatal("shared quota retried on different model", calls)
	}
}

func TestSubscriptionFreshQuotaRecoveryDoesNotKeepOldModelCooldown(t *testing.T) {
	s, _, _, access := subscriptionQuotaFixture(t)
	calls := 0
	seen := []string{}
	s.chatgpt = chatgpt.New(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		seen = append(seen, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer placeholder-subscription-"))
		body := missingTypeStream
		if calls == 1 {
			body = fmt.Sprintf(`{"error":{"type":"usage_limit_reached","resets_at":%d}}`, time.Now().Add(7*24*time.Hour).Unix())
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	if ok, _, _ := requestQuotaGateway(t, s, "http", "gpt-test"); !ok {
		t.Fatal("initial backup failed")
	}
	access.mu.Lock()
	delete(access.blocked, "a")
	access.mu.Unlock()
	if ok, _, _ := requestQuotaGateway(t, s, "http", "gpt-test"); !ok {
		t.Fatal("reset quota was not usable")
	}
	if calls != 3 || strings.Join(seen, ",") != "a,b,a" {
		t.Fatal("fresh quota introduced extra retries", calls)
	}
	access.mu.Lock()
	defer access.mu.Unlock()
	if !access.blocked["a"].IsZero() {
		t.Fatal("healthy account was reblocked")
	}
}
