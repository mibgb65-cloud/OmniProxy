package gateway

import (
	"context"
	"io"
	"net/http"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/core/proxy"
	"strings"
	"testing"
)

func TestFailoverDuplicateKeysAreNotSentTwice(t *testing.T) {
	calls := []string{}
	s, db, observer, _ := failoverFixture(t, roundTripFunc(func(r *http.Request) (*http.Response, error) {
		key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer placeholder-failover-")
		calls = append(calls, key)
		if key == "a" {
			return upstreamReply(429, `{"error":{}}`, false), nil
		}
		return upstreamReply(200, `{"choices":[{"message":{"content":"OK"}}]}`, false), nil
	}))
	if _, err := db.Exec(`UPDATE accounts SET credential_ref='a' WHERE id='b'`); err != nil {
		t.Fatal(err)
	}
	status, _ := localGatewayRequest(t, s, "deepseek-flash", false)
	if status != 200 || strings.Join(calls, ",") != "a,c" {
		t.Fatal("duplicate secret resent", status, calls)
	}
	page, err := observer.Requests(context.Background(), activity.RequestQuery{Limit: 10})
	if err != nil || len(page.Items) != 1 {
		t.Fatal(err)
	}
	item, err := observer.Request(context.Background(), page.Items[0].ID)
	if err != nil || item.AccountID != "c" || len(item.Attempts) != 3 || item.Attempts[1].ErrorCode != "duplicate_credential" {
		t.Fatal("duplicate attempt detail lost", item, err)
	}
}

func TestFailoverRefusesCandidateVersionChange(t *testing.T) {
	s, db, _, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) { return upstreamReply(429, `{"error":{}}`, false), nil }))
	ctx := context.Background()
	request, err := proxy.ParseProtocolRequest([]byte(`{"model":"deepseek-flash","messages":[{"role":"user","content":"hi"}]}`), "/v1/chat/completions")
	if err != nil {
		t.Fatal(err)
	}
	request, err = s.RouteRequest(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`UPDATE accounts SET updated_at='2026-10-18T00:00:00Z' WHERE id='b'`); err != nil {
		t.Fatal(err)
	}
	err = s.Forward(ctx, request, func(proxy.Response) error { t.Fatal("stale plan consumed"); return nil })
	if err == nil || !strings.Contains(err.Error(), "route_configuration_changed") {
		t.Fatal("stale candidate not rejected", err)
	}
}

func TestFailoverCancellationNeverStartsNextAccount(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	s, _, _, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		cancel()
		return upstreamReply(429, `{"error":{}}`, false), nil
	}))
	request, err := proxy.ParseProtocolRequest([]byte(`{"model":"deepseek-flash","messages":[{"role":"user","content":"hi"}]}`), "/v1/chat/completions")
	if err != nil {
		t.Fatal(err)
	}
	request, err = s.RouteRequest(ctx, request)
	if err != nil {
		t.Fatal(err)
	}
	err = s.Forward(ctx, request, func(response proxy.Response) error { _, err := io.Copy(io.Discard, response.Body); return err })
	if calls != 1 || ctx.Err() == nil || err == nil {
		t.Fatal("cancelled request retried", calls, err)
	}
}
