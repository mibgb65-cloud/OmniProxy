package quotaactivation

import (
	"context"
	"io"
	"net/http"
	"omniproxy/internal/application/subscription"
	"omniproxy/internal/provider/chatgpt"
	"strings"
	"testing"
	"time"
)

func TestSlidingWorkerActualSenderLedgerAndConfirmation(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	now := time.Now().UTC().Truncate(time.Second).Add(-5 * time.Minute)
	f.service.now = func() time.Time { return now }
	access := &testAccess{expectedID: f.accounts.item.ID, officialID: f.accounts.item.AuthAccountID, readHook: func() subscription.Quota { return floatingQuota(time.Now().UTC().Truncate(time.Second)) }}
	calls := 0
	client := chatgpt.New(roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		var cycle string
		if err := f.db.QueryRow(`SELECT attempted_cycle FROM codex_quota_activation_windows WHERE window_kind='five_hour'`).Scan(&cycle); err != nil || cycle != "initial" {
			t.Error("message sent before persistent claim", err)
		}
		if req.Header.Get("ChatGPT-Account-ID") != f.accounts.item.AuthAccountID {
			t.Error("worker selected another account")
		}
		fresh := activeQuota()
		fresh.FetchedAt = time.Now().UTC().Format(time.RFC3339)
		f.quotas.set(fresh)
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"qa\",\"status\":\"completed\",\"output\":[]}}\n\n"))}, nil
	}))
	f.service.sender = NewMessageSender(f.accounts, f.providers, access, client)
	f.quotas.set(floatingQuota(now))
	f.service.Check(context.Background())
	now = now.Add(5 * time.Minute)
	f.quotas.set(floatingQuota(now))
	if err := f.service.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	status := f.service.Status()
	if calls != 1 || status.Sent != 1 || status.Activated != 1 || status.Pending != 0 {
		t.Fatal("worker did not complete bounded activation", status, calls)
	}
	f.service.Check(context.Background())
	if calls != 1 {
		t.Fatal("worker resent active request")
	}
}
