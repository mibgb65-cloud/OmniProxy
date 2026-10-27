package codex

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func resetCreditsServer(t *testing.T, status int, body string) *UsageClient {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rate-limit-reset-credits" ||
			r.Header.Get("Authorization") != "Bearer access-token" ||
			r.Header.Get("ChatGPT-Account-Id") != "account-one" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return &UsageClient{http: server.Client(), endpoint: server.URL + "/usage"}
}

func TestResetCreditsReadMapsOfficialSummary(t *testing.T) {
	client := resetCreditsServer(t, http.StatusOK, `{"available_count":2,"credits":[`+
		`{"id":"credit-one","title":"额度重置卡","description":"重置 5h 与一周窗口","reset_type":"codexRateLimits",`+
		`"status":"available","granted_at":1761346800,"expires_at":"2027-01-15T08:00:00Z"},`+
		`{"id":"credit-two","granted_at":"1761433200"}]}`)
	summary, err := client.ReadResetCredits(context.Background(), []byte("access-token"), "account-one")
	if err != nil {
		t.Fatal(err)
	}
	if summary.AvailableCount == nil || *summary.AvailableCount != 2 || len(summary.Credits) != 2 {
		t.Fatal("official reset credits were not mapped")
	}
	first, second := summary.Credits[0], summary.Credits[1]
	if first.ID != "credit-one" || first.Title != "额度重置卡" || first.ResetType != "codexRateLimits" || first.Status != "available" {
		t.Fatal("reset credit fields were not preserved")
	}
	if first.GrantedAt == nil || *first.GrantedAt != 1761346800 || first.ExpiresAt == nil || *first.ExpiresAt != 1800000000 {
		t.Fatal("reset credit times were not converted")
	}
	// 官方不返回的字段保持未知，不伪造标题、状态或时间。
	if second.Title != "" || second.Status != "" || second.ExpiresAt != nil || second.GrantedAt == nil || *second.GrantedAt != 1761433200 {
		t.Fatal("missing optional fields must stay unknown")
	}
}

func TestResetCreditsReadKeepsCountOnlyAndEmptyDetails(t *testing.T) {
	// 官方可能把计数序列化成字符串。
	client := resetCreditsServer(t, http.StatusOK, `{"available_count":"3","credits":null}`)
	summary, err := client.ReadResetCredits(context.Background(), []byte("access-token"), "account-one")
	if err != nil || summary.AvailableCount == nil || *summary.AvailableCount != 3 || summary.Credits != nil {
		t.Fatal("count-only summary must stay without details")
	}
	client = resetCreditsServer(t, http.StatusOK, `{"data":{"availableCount":0,"credits":[]}}`)
	summary, err = client.ReadResetCredits(context.Background(), []byte("access-token"), "account-one")
	if err != nil || summary.AvailableCount == nil || *summary.AvailableCount != 0 || summary.Credits == nil || len(summary.Credits) != 0 {
		t.Fatal("empty details must stay known and empty")
	}
}

func TestResetCreditsReadClassifiesFailures(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status     int
		want       error
	}{
		{name: "unauthorized", status: http.StatusForbidden, want: ErrUsageUnauthorized},
		{name: "rate limited", status: http.StatusTooManyRequests, want: ErrUsageTransient},
		{name: "server error", status: http.StatusBadGateway, want: ErrUsageTransient},
		{name: "malformed body", status: http.StatusOK, body: `{"available_count":`, want: ErrProtocol},
		{name: "missing fields", status: http.StatusOK, body: `{"items":[]}`, want: ErrProtocol},
		// 不按明细长度推算数量：官方没给出总数时保持未知。
		{name: "details without count", status: http.StatusOK, body: `{"credits":[{"id":"x","status":"available"}]}`, want: ErrProtocol},
		{name: "negative count", status: http.StatusOK, body: `{"available_count":-1}`, want: ErrProtocol},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := resetCreditsServer(t, test.status, test.body)
			if _, err := client.ReadResetCredits(context.Background(), []byte("access-token"), "account-one"); !errors.Is(err, test.want) {
				t.Fatalf("want %v, got %v", test.want, err)
			}
		})
	}
}

func TestResetCreditsTargetStaysOnOfficialPrefix(t *testing.T) {
	if target := (&UsageClient{}).resetCreditsTarget(); target != "https://chatgpt.com/backend-api/wham/rate-limit-reset-credits" {
		t.Fatalf("unexpected default target: %q", target)
	}
	client := &UsageClient{endpoint: "https://example.test/custom/usage/"}
	if target := client.resetCreditsTarget(); target != "https://example.test/custom/rate-limit-reset-credits" {
		t.Fatalf("reset credits path must derive from the usage endpoint: %q", target)
	}
}

func TestResetCreditsReadDoesNotSendWithoutIdentity(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer server.Close()
	client := &UsageClient{http: server.Client(), endpoint: server.URL + "/usage"}
	if _, err := client.ReadResetCredits(context.Background(), []byte("access-token"), ""); !errors.Is(err, ErrProtocol) {
		t.Fatal("missing account identity must not be queried")
	}
	if _, err := client.ReadResetCredits(context.Background(), nil, "account-one"); !errors.Is(err, ErrProtocol) {
		t.Fatal("missing access token must not be queried")
	}
	if called {
		t.Fatal("official endpoint was called without identity")
	}
}
