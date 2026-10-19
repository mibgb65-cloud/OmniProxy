package quotaactivation

import (
	"context"
	"errors"
	"io"
	"net/http"
	"omniproxy/internal/application/subscription"
	"omniproxy/internal/provider/chatgpt"
	"strings"
	"testing"
	"time"
)

func TestFloatingPreflightRequiresFreshSupportingEvidence(t *testing.T) {
	base := time.Now().UTC().Truncate(time.Second).Add(-35 * time.Second)
	for _, name := range []string{"moving", "fixed", "used", "missing", "too-soon", "invalid-kind", "stale", "irregular"} {
		t.Run(name, func(t *testing.T) {
			before := floatingQuota(base)
			fresh := floatingQuota(base.Add(35 * time.Second))
			plan := SendPlan{Before: before, Claimed: []string{"five_hour"}, Floating: []string{"five_hour"}}
			switch name {
			case "fixed":
				fresh.FiveHour.ResetsAt = before.FiveHour.ResetsAt
			case "used":
				fresh.FiveHour.RemainingPercent = 99
			case "missing":
				fresh.FiveHour = nil
			case "too-soon":
				fresh = floatingQuota(base.Add(time.Second))
			case "invalid-kind":
				plan.Floating = []string{"other"}
			case "stale":
				fresh = floatingQuota(base.Add(2 * time.Minute))
			case "irregular":
				fresh.FiveHour.ResetsAt = base.Add(5*time.Hour + 55*time.Second).Format(time.RFC3339)
			}
			needed, err := plannedNeed(fresh, plan)
			if name == "moving" {
				if err != nil || !needed {
					t.Fatal("moving evidence rejected", err)
				}
			} else if needed {
				t.Fatal("unsafe preflight accepted", name)
			}
		})
	}
}

func TestFloatingMessageUsesActualSenderAndKeepsAccountIdentity(t *testing.T) {
	f := newFixture(t)
	before := floatingQuota(time.Now().UTC().Truncate(time.Second).Add(-35 * time.Second))
	fresh := floatingQuota(time.Now().UTC().Truncate(time.Second))
	access := &testAccess{quota: fresh, expectedID: f.accounts.item.ID, officialID: f.accounts.item.AuthAccountID}
	calls := 0
	client := chatgpt.New(roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if request.Header.Get("ChatGPT-Account-ID") != f.accounts.item.AuthAccountID {
			t.Error("wrong activation account")
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"activation-test\",\"status\":\"completed\",\"output\":[]}}\n\n"))}, nil
	}))
	sender := NewMessageSender(f.accounts, f.providers, access, client)
	if err := sender.Send(context.Background(), f.accounts.item, SendPlan{Before: before, Claimed: []string{"five_hour"}, Floating: []string{"five_hour"}}); err != nil || calls != 1 {
		t.Fatal("floating sender did not send once", err, calls)
	}
	access.quota = floatingQuota(time.Now().UTC().Truncate(time.Second))
	access.quota.FiveHour.ResetsAt = before.FiveHour.ResetsAt
	if err := sender.Send(context.Background(), f.accounts.item, SendPlan{Before: before, Claimed: []string{"five_hour"}, Floating: []string{"five_hour"}}); !errors.Is(err, ErrAlreadyActive) || calls != 1 {
		t.Fatal("preflight started timer resent", err)
	}
}

func TestImmediateFloatingPreflightWaitsAndCanStillSend(t *testing.T) {
	f := newFixture(t)
	before := floatingQuota(time.Now().UTC().Truncate(time.Second))
	observedGap := time.Duration(0)
	access := &testAccess{expectedID: f.accounts.item.ID, officialID: f.accounts.item.AuthAccountID}
	access.readHook = func() subscription.Quota {
		now := time.Now().UTC().Truncate(time.Second)
		old, _ := time.Parse(time.RFC3339, before.FetchedAt)
		observedGap = now.Sub(old)
		return floatingQuota(now)
	}
	calls := 0
	client := chatgpt.New(roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"id\":\"activation-test\",\"status\":\"completed\",\"output\":[]}}\n\n"))}, nil
	}))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := NewMessageSender(f.accounts, f.providers, access, client).Send(ctx, f.accounts.item, SendPlan{Before: before, Claimed: []string{"five_hour"}, Floating: []string{"five_hour"}}); err != nil || calls != 1 || observedGap < 2*time.Second {
		t.Fatal("same-second activation could never progress", err, calls, observedGap)
	}
}

func TestFloatingPreflightCancellationNeverReadsOrSends(t *testing.T) {
	f := newFixture(t)
	reads := 0
	access := &testAccess{readHook: func() subscription.Quota { reads++; return unstartedQuota() }}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := NewMessageSender(f.accounts, f.providers, access, nil).Send(ctx, f.accounts.item, SendPlan{Before: floatingQuota(time.Now().UTC().Truncate(time.Second)), Claimed: []string{"five_hour"}, Floating: []string{"five_hour"}})
	if !errors.Is(err, ErrNotSent) || reads != 0 {
		t.Fatal("cancelled wait performed quota or model requests", err, reads)
	}
}

func TestPreflightNeverUsesAnUnclaimedWindowAsSendPermission(t *testing.T) {
	fresh := unstartedQuota()
	fresh.Weekly = activeQuota().Weekly
	needed, err := plannedNeed(fresh, SendPlan{Claimed: []string{"weekly"}})
	if err != nil || needed {
		t.Fatal("unclaimed five-hour window authorized a message", err)
	}
	plan := buildSendPlan(floatingQuota(time.Now().UTC()), []windowClaim{{kind: "weekly", cycle: "initial"}}, []string{"five_hour"})
	if len(plan.Floating) != 0 || len(plan.Claimed) != 1 {
		t.Fatal("plan included an unclaimed floating window")
	}
	_, err = plannedNeed(fresh, SendPlan{Claimed: []string{"weekly"}, Floating: []string{"five_hour"}})
	if !errors.Is(err, ErrNotSent) {
		t.Fatal("invalid floating permission accepted")
	}
}
