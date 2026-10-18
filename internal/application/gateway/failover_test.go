package gateway

import (
	"context"
	"io"
	"net/http"
	"omniproxy/internal/application/activity"
	routes "omniproxy/internal/application/routing"
	"strings"
	"sync"
	"testing"
)

func upstreamReply(status int, body string, stream bool) *http.Response {
	kind := "application/json"
	if stream {
		kind = "text/event-stream"
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {kind}, "Retry-After": {"60"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestFailoverRejectThenSuccessAndAccounting(t *testing.T) {
	for _, protocol := range []string{"chat/completions", "responses", "messages"} {
		for _, stream := range []bool{false, true} {
			t.Run(protocol+map[bool]string{false: "/json", true: "/sse"}[stream], func(t *testing.T) {
				var mu sync.Mutex
				calls := []string{}
				s, _, observer, _ := failoverFixture(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
					key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer placeholder-failover-")
					mu.Lock()
					calls = append(calls, key)
					mu.Unlock()
					if key == "a" {
						return upstreamReply(429, `{"error":{"code":"rate_limit"}}`, false), nil
					}
					if protocol == "chat/completions" && stream {
						return upstreamReply(200, "data: {\"choices\":[{\"delta\":{\"content\":\"OK\"}}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":4}}\n\ndata: [DONE]\n\n", true), nil
					}
					return upstreamReply(200, `{"choices":[{"message":{"content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":12,"completion_tokens":4}}`, false), nil
				}))
				body := `{"model":"deepseek-flash","stream":` + map[bool]string{true: "true", false: "false"}[stream] + `,"messages":[{"role":"user","content":"hi"}],"max_tokens":32}`
				if protocol == "responses" {
					body = `{"model":"deepseek-flash","stream":` + map[bool]string{true: "true", false: "false"}[stream] + `,"input":"hi"}`
				}
				for index := 0; index < 2; index++ {
					response, err := http.Post(s.Info().Runtime.Address+"/v1/"+protocol, "application/json", strings.NewReader(body))
					if err != nil {
						t.Fatal(err)
					}
					raw, err := io.ReadAll(response.Body)
					response.Body.Close()
					if err != nil || response.StatusCode != 200 {
						t.Fatalf("response %d: %s %v", response.StatusCode, raw, err)
					}
					item, err := observer.Request(context.Background(), response.Header.Get("X-Request-ID"))
					if err != nil || item.AccountID != "b" || item.AttemptCount != 2-index || item.InputTokens != "12" || item.OutputTokens != "4" {
						t.Fatalf("wrong accounting: %+v %v", item, err)
					}
					if index == 0 && (item.Attempts[0].State != "failed" || item.Attempts[0].InputKnown || item.Attempts[1].State != "success" || !item.Attempts[1].UsageComplete) {
						t.Fatalf("wrong attempts: %+v", item.Attempts)
					}
				}
				mu.Lock()
				defer mu.Unlock()
				if strings.Join(calls, ",") != "a,b,b" {
					t.Fatal("wrong attempts", calls)
				}
				usage, err := observer.Usage(context.Background())
				if err != nil || usage.AllTime.Requests != 2 || usage.AllTime.TotalTokens != "32" {
					t.Fatal("logical request duplicated", usage, err)
				}
			})
		}
	}
}

func TestFailoverFixedDisabledAndTerminalFailures(t *testing.T) {
	for _, name := range []string{"fixed", "disabled", "400", "403", "500", "503", "invalid-200", "broken-sse"} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			s, db, observer, router := failoverFixture(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
				calls++
				status, body, stream := 429, `{"error":{}}`, false
				switch name {
				case "400":
					status = 400
				case "403":
					status = 403
				case "500":
					status = 500
				case "503":
					status = 503
				case "invalid-200":
					status = 200
					body = `{"unexpected":true}`
				case "broken-sse":
					status = 200
					stream = true
					body = "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n"
				}
				return upstreamReply(status, body, stream), nil
			}))
			model := "deepseek-flash"
			if name == "fixed" {
				model = "fixed"
				if _, err := router.Create(context.Background(), routes.Input{Alias: model, Model: "deepseek-flash", ProviderID: "builtin_deepseek", AccountID: "a", Enabled: true}); err != nil {
					t.Fatal(err)
				}
			}
			if name == "disabled" {
				if _, err := NewFailoverSettingsService(db).Set(context.Background(), false); err != nil {
					t.Fatal(err)
				}
			}
			status := 0
			if name == "broken-sse" {
				response, err := http.Post(s.Info().Runtime.Address+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"deepseek-flash","stream":true,"messages":[{"role":"user","content":"hi"}]}`))
				if err != nil {
					t.Fatal(err)
				}
				_, readErr := io.ReadAll(response.Body)
				response.Body.Close()
				status = response.StatusCode
				if readErr == nil {
					t.Fatal("broken stream must be incomplete")
				}
			} else {
				status, _ = localGatewayRequest(t, s, model, false)
			}
			if calls != 1 {
				t.Fatal("terminal failure replayed", calls)
			}
			if name != "broken-sse" && status == 200 {
				t.Fatal("unexpected success")
			}
			page, err := observer.Requests(context.Background(), activity.RequestQuery{Limit: 10})
			if err != nil || len(page.Items) != 1 || page.Items[0].AttemptCount != 1 {
				t.Fatal("wrong attempt count", page, err)
			}
		})
	}
}

func TestFailoverAttemptLimitAndCoolingPool(t *testing.T) {
	calls := 0
	s, _, observer, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return upstreamReply(429, `{"error":{}}`, false), nil
	}))
	for _, expected := range []int{3, 4, 4} {
		status, _ := localGatewayRequest(t, s, "deepseek-flash", false)
		if status != 429 || calls != expected {
			t.Fatal("unbounded or repeated calls", calls, status)
		}
	}
	usage, _ := observer.Usage(context.Background())
	if usage.AllTime.Requests != 3 {
		t.Fatal("attempts counted as requests", usage)
	}
}
