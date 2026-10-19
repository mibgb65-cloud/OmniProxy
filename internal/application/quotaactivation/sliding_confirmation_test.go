package quotaactivation

import (
	"context"
	"omniproxy/internal/application/account"
	"testing"
	"time"
)

func TestSuccessfulMessageWithoutQuotaChangeIsNeverCalledActivated(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	now := time.Now().UTC().Truncate(time.Second)
	f.service.now = func() time.Time { return now }
	f.service.sender = sendFunc(func(context.Context, account.Item, SendPlan) error { f.sends.Add(1); return nil })
	for range 2 {
		f.quotas.set(floatingQuota(now))
		f.service.Check(context.Background())
		now = now.Add(5 * time.Minute)
	}
	status := f.service.Status()
	if f.sends.Load() != 1 || status.Sent != 1 || status.Activated != 0 || status.Pending != 1 {
		t.Fatal("message acknowledgement falsely confirmed timer", status)
	}
	f.quotas.set(floatingQuota(now))
	f.service.Check(context.Background())
	if f.sends.Load() != 1 {
		t.Fatal("unchanged quota retried")
	}
}

func TestStableZeroRoundedUsageConfirmsLaterWithoutAnotherMessage(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	now := time.Now().UTC().Truncate(time.Second)
	f.service.now = func() time.Time { return now }
	fixed := ""
	f.service.sender = sendFunc(func(context.Context, account.Item, SendPlan) error {
		f.sends.Add(1)
		fresh := floatingQuota(now)
		fixed = fresh.FiveHour.ResetsAt
		f.quotas.set(fresh)
		return nil
	})
	f.quotas.set(floatingQuota(now))
	f.service.Check(context.Background())
	now = now.Add(5 * time.Minute)
	f.quotas.set(floatingQuota(now))
	f.service.Check(context.Background())
	if f.service.Status().Activated != 0 {
		t.Fatal("same-second zero usage confirmed too early")
	}
	now = now.Add(40 * time.Second)
	fresh := floatingQuota(now)
	fresh.FiveHour.ResetsAt = fixed
	f.quotas.set(fresh)
	f.service.Check(context.Background())
	if f.sends.Load() != 1 || f.service.Status().Activated != 1 {
		t.Fatal("stable rounded timer not confirmed without resend", f.service.Status())
	}
}
