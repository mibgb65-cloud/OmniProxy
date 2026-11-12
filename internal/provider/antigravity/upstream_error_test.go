package antigravity

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/core/proxy"
)

const (
	refusalReply  = `{"error":{"code":429,"message":"Resource has been exhausted (e.g. check quota).","status":"RESOURCE_EXHAUSTED"}}`
	capacityReply = `{"error":{"code":429,"message":"You have exhausted your capacity on this model. Your quota will reset after 3h19m30s.","status":"RESOURCE_EXHAUSTED"}}`
	refusalSystem = "x-anthropic-billing-header: cc_version=2.1.299"
)

func googleRequest(system string) proxy.Request {
	return proxy.Request{Protocol: "chat", Model: "gemini-3.8-flash", Body: []byte(`{"messages":[{"role":"system","content":"` + system + `"},{"role":"user","content":"hi"}]}`)}
}

func forwardOnce(t *testing.T, client *Client, request proxy.Request) error {
	t.Helper()
	return client.Forward(context.Background(), testCredentials(), request, func(proxy.Response) error { return nil })
}

func proxyCode(t *testing.T, err error) *proxy.Error {
	t.Helper()
	var failure *proxy.Error
	if !errors.As(err, &failure) {
		t.Fatalf("expected a proxy error, got %v", err)
	}
	return failure
}

func TestPromptRefusalBecomesNonRetryable400(t *testing.T) {
	client := New(transportFunc(func(*http.Request) (*http.Response, error) {
		return reply(http.StatusTooManyRequests, refusalReply), nil
	}))
	failure := proxyCode(t, forwardOnce(t, client, googleRequest(refusalSystem)))
	if failure.Code != "prompt_turned_away" || failure.StatusCode != http.StatusBadRequest {
		t.Fatalf("refusal was not mapped to a 400: %s/%d", failure.Code, failure.StatusCode)
	}
	if ErrorCode(errors.New("google_subscription: "+failure.Code)) != "prompt_turned_away" {
		t.Fatal("refusal code is not part of the stable whitelist")
	}
}

func TestRefusalBodyWithoutRefusalPromptStaysRateLimited(t *testing.T) {
	client := New(transportFunc(func(*http.Request) (*http.Response, error) {
		return reply(http.StatusTooManyRequests, refusalReply), nil
	}))
	failure := proxyCode(t, forwardOnce(t, client, googleRequest("You are a coding agent.")))
	if failure.Code != "rate_limited" || failure.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("unexpected classification: %s/%d", failure.Code, failure.StatusCode)
	}
}

func TestCapacityQuotaCarriesResetSeconds(t *testing.T) {
	client := New(transportFunc(func(*http.Request) (*http.Response, error) {
		return reply(http.StatusTooManyRequests, capacityReply), nil
	}))
	failure := proxyCode(t, forwardOnce(t, client, googleRequest("You are a coding agent.")))
	if failure.Code != "upstream_quota_exhausted" || failure.RetryAfter != "11970" {
		t.Fatalf("capacity 429 lost its reset window: %s/%q", failure.Code, failure.RetryAfter)
	}
}

func TestClassifyFailureBoundaries(t *testing.T) {
	for _, test := range []struct {
		name    string
		status  int
		body    string
		refusal bool
		reset   time.Duration
	}{
		{"generic", 429, `{"error":{"message":"too many requests"}}`, false, 0},
		{"capacity", 429, capacityReply, false, 3*time.Hour + 19*time.Minute + 30*time.Second},
		{"capacity_beyond_limit", 429, `quota will reset after 999h`, false, 0},
		{"refusal", 429, refusalReply, true, 0},
		{"not_a_429", 500, refusalReply, false, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			classified := classifyFailure(test.status, []byte(test.body))
			if classified == nil {
				if test.status == 429 {
					t.Fatal("429 was not classified")
				}
				return
			}
			if classified.refusal != test.refusal || classified.reset != test.reset {
				t.Fatalf("classification mismatch: refusal=%t reset=%s", classified.refusal, classified.reset)
			}
			if strings.Contains(classified.Error(), "quota") {
				t.Fatal("error text carries upstream wording")
			}
		})
	}
}

func TestRefusedPromptIsSkippedUnlessItIsTheLastCandidate(t *testing.T) {
	calls := 0
	client := New(transportFunc(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return reply(http.StatusTooManyRequests, refusalReply), nil
		}
		return reply(http.StatusOK, `{"response":{"candidates":[{"content":{"parts":[{"text":"ok"}]},"finishReason":"STOP"}]}}`), nil
	}))
	if failure := proxyCode(t, forwardOnce(t, client, googleRequest(refusalSystem))); failure.Code != "prompt_turned_away" {
		t.Fatalf("first attempt not refused: %s", failure.Code)
	}
	if failure := proxyCode(t, forwardOnce(t, client, googleRequest(refusalSystem))); failure.Code != "prompt_refused_before" {
		t.Fatalf("known-refused prompt was sent again: %s", failure.Code)
	}
	if calls != 1 {
		t.Fatalf("upstream was called %d times, want 1", calls)
	}
	last := googleRequest(refusalSystem)
	last.LastCandidate = true
	if err := forwardOnce(t, client, last); err != nil {
		t.Fatalf("last candidate must still be asked: %v", err)
	}
	if calls != 2 {
		t.Fatalf("last candidate did not reach upstream: %d", calls)
	}
	// 另一个账号的同一 prompt 不受影响。
	other := testCredentials()
	other.Subject = "2002"
	request := googleRequest(refusalSystem)
	request.LastCandidate = true
	if err := client.Forward(context.Background(), other, request, func(proxy.Response) error { return nil }); err != nil {
		t.Fatalf("other account inherited the refusal: %v", err)
	}
}
