package codex

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func usageServer(t *testing.T, status int, body string) *UsageClient {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer access-token" || r.Header.Get("ChatGPT-Account-Id") != "account-one" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return &UsageClient{http: server.Client(), endpoint: server.URL}
}

func TestUsageReadMapsOfficialWindows(t *testing.T) {
	client := usageServer(t, http.StatusOK, `{"plan_type":"plus","rate_limit":{`+
		`"primary_window":{"used_percent":12.4,"limit_window_seconds":18000,"reset_at":1800000000},`+
		`"secondary_window":{"used_percent":40,"limit_window_seconds":604800,"reset_at":1800604800}}}`)
	result, err := client.Read(context.Background(), []byte("access-token"), "account-one")
	if err != nil {
		t.Fatal(err)
	}
	snapshot := result.Snapshot
	if snapshot == nil || snapshot.Plan != "plus" || snapshot.Primary == nil || snapshot.Secondary == nil {
		t.Fatal("official windows were not mapped")
	}
	if *snapshot.Primary.DurationMins != 300 || *snapshot.Primary.UsedPercent != 12 || *snapshot.Secondary.DurationMins != 10080 {
		t.Fatal("window duration or usage was not converted to the stdio contract")
	}
	if *snapshot.Primary.ResetsAt != 1800000000 {
		t.Fatal("reset time was not preserved")
	}
}

func TestUsageReadClassifiesFailures(t *testing.T) {
	for _, test := range []struct {
		name, body string
		status     int
		want       error
	}{
		{name: "unauthorized", status: http.StatusForbidden, want: ErrUsageUnauthorized},
		{name: "rate limited", status: http.StatusTooManyRequests, want: ErrUsageTransient},
		{name: "server error", status: http.StatusBadGateway, want: ErrUsageTransient},
		{name: "malformed body", status: http.StatusOK, body: `{"rate_limit":`, want: ErrProtocol},
		{name: "missing windows", status: http.StatusOK, body: `{"plan_type":"plus"}`, want: ErrProtocol},
		{name: "window without duration", status: http.StatusOK, body: `{"rate_limit":{"primary_window":{"used_percent":5}}}`, want: ErrProtocol},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := usageServer(t, test.status, test.body)
			if _, err := client.Read(context.Background(), []byte("access-token"), "account-one"); !errors.Is(err, test.want) {
				t.Fatalf("want %v, got %v", test.want, err)
			}
		})
	}
}

func TestUsageReadDoesNotSendWithoutIdentity(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer server.Close()
	client := &UsageClient{http: server.Client(), endpoint: server.URL}
	if _, err := client.Read(context.Background(), []byte("access-token"), ""); !errors.Is(err, ErrProtocol) {
		t.Fatal("missing account identity must not be queried")
	}
	if _, err := client.Read(context.Background(), nil, "account-one"); !errors.Is(err, ErrProtocol) {
		t.Fatal("missing access token must not be queried")
	}
	if called {
		t.Fatal("official endpoint was called without identity")
	}
}
