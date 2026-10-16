package antigravity

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

const quotaFixture = `{"groups":[{"displayName":"Gemini Models","buckets":[{"bucketId":"gemini-weekly","displayName":"Weekly Limit Remaining","window":"weekly","remainingFraction":0.956731,"resetTime":"2026-10-23T23:00:00+08:00"},{"bucketId":"gemini-5h","window":"5h","remainingFraction":0,"resetTime":"2026-10-17T04:00:00Z"}]},{"displayName":"Claude and GPT models","buckets":[{"bucketId":"3p-weekly","window":"weekly","remainingFraction":null},{"bucketId":"3p-5h","window":"5h","remainingFraction":0.00000001}]}],"access_token":"private-extra-token","email":"private@example.test"}`

func TestQuotaUsesSummaryAndPreservesUnknownZeroAndPrecision(t *testing.T) {
	client := New(transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "daily-cloudcode-pa.googleapis.com" || r.URL.Path != "/v1internal:retrieveUserQuotaSummary" || r.Method != http.MethodPost {
			t.Error("quota did not use the fixed summary endpoint")
		}
		if r.Header.Get("Authorization") != "Bearer test-access" || r.URL.RawQuery != "" {
			t.Error("credential was not isolated to the authorization header")
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"project":"test-project"}` {
			t.Error("quota query leaked or omitted project context")
		}
		return reply(http.StatusOK, quotaFixture), nil
	}))
	quota, err := client.ReadQuota(context.Background(), testCredentials())
	if err != nil {
		t.Fatal(err)
	}
	if len(quota.Groups) != 2 || len(quota.Groups[0].Buckets) != 2 || quota.FetchedAt == "" {
		t.Fatal("quota groups were changed or combined")
	}
	buckets := quota.Groups[0].Buckets
	if buckets[0].RemainingPercent == nil || *buckets[0].RemainingPercent != 95.6731 || buckets[0].ResetsAt != "2026-10-23T15:00:00Z" {
		t.Fatal("official percentage or reset time changed")
	}
	if buckets[1].RemainingPercent == nil || *buckets[1].RemainingPercent != 0 {
		t.Fatal("exhausted quota was treated as missing")
	}
	if quota.Groups[1].Buckets[0].RemainingPercent != nil || *quota.Groups[1].Buckets[1].RemainingPercent <= 0 {
		t.Fatal("unknown or tiny positive quota was rounded to zero")
	}
	data, _ := json.Marshal(quota)
	for _, private := range []string{"private-extra-token", "private@example.test", "test-project", "test-access"} {
		if strings.Contains(string(data), private) {
			t.Fatal("private upstream data escaped the quota DTO")
		}
	}
}

func TestQuotaRejectsMalformedBoundsAndRetainsEmptyAsUnknown(t *testing.T) {
	for _, body := range []string{
		`{}`, `{"groups":null}`,
		`{"groups":[{"displayName":"","buckets":[]}]}`,
		`{"groups":[{"displayName":"Gemini","buckets":[{"bucketId":"x","remainingFraction":-0.1}]}]}`,
		`{"groups":[{"displayName":"Gemini","buckets":[{"bucketId":"x","remainingFraction":1.01}]}]}`,
		`{"groups":[{"displayName":"Gemini","buckets":[{"bucketId":"x","remainingFraction":"1"}]}]}`,
		`{"groups":[{"displayName":"Gemini","buckets":[{"bucketId":"x","resetTime":"tomorrow"}]}]}`,
		`{"groups":[{"displayName":"Gemini","buckets":[{"bucketId":"x"},{"bucketId":"x"}]}]}`,
	} {
		client := New(transportFunc(func(*http.Request) (*http.Response, error) { return reply(200, body), nil }))
		if _, err := client.ReadQuota(context.Background(), testCredentials()); ErrorCode(err) != "invalid_response" {
			t.Fatal("malformed quota was accepted")
		}
	}
	client := New(transportFunc(func(*http.Request) (*http.Response, error) { return reply(200, `{"groups":[]}`), nil }))
	quota, err := client.ReadQuota(context.Background(), testCredentials())
	if err != nil || quota.Groups == nil || len(quota.Groups) != 0 {
		t.Fatal("empty official quota was replaced with invented windows")
	}
	client = New(transportFunc(func(*http.Request) (*http.Response, error) { return reply(429, `{"error":"private-quota-error"}`), nil }))
	if _, err := client.ReadQuota(context.Background(), testCredentials()); ErrorCode(err) != "rate_limited" || strings.Contains(err.Error(), "private-quota-error") {
		t.Fatal("quota errors lost their safe classification")
	}
}
