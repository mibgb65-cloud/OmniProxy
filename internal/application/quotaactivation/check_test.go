package quotaactivation

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"omniproxy/internal/application/account"
)

func TestDefaultOffAndSingleMessageActivatesBothWindows(t *testing.T) {
	f := newFixture(t)
	ctx := context.Background()
	if err := f.service.Check(ctx); err != nil {
		t.Fatal(err)
	}
	if f.quotas.reads.Load() != 0 || f.sends.Load() != 0 {
		t.Fatal("default-off feature made requests")
	}
	f.enable(t)
	if err := f.service.Check(ctx); err != nil {
		t.Fatal(err)
	}
	status := f.service.Status()
	if f.sends.Load() != 1 || status.Activated != 1 || status.Pending != 0 {
		t.Fatalf("not activated: %+v", status)
	}
	if err := f.service.Check(ctx); err != nil {
		t.Fatal(err)
	}
	if f.sends.Load() != 1 {
		t.Fatal("active account resent")
	}
	var count int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM codex_quota_activation_windows WHERE attempted_cycle='initial'`).Scan(&count); err != nil || count != 2 {
		t.Fatal("two windows were not claimed before one message", err)
	}
}

func TestUnknownExpiredOrExhaustedQuotaAndDisabledAccountsAreSkipped(t *testing.T) {
	for name, quota := range map[string]struct {
		five, week *string
		remaining  int
	}{
		"unknown": {nil, nil, 100}, "expired": {ptr(time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)), nil, 100},
		"active": {ptr(time.Now().Add(time.Hour).UTC().Format(time.RFC3339)), nil, 100}, "invalid": {ptr("invalid"), nil, 100},
		"exhausted": {nil, nil, 0},
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			f.enable(t)
			value := unstartedQuota()
			value.Weekly = nil
			if quota.five == nil && name == "unknown" {
				value.FiveHour = nil
			} else {
				value.FiveHour.RemainingPercent = quota.remaining
				if quota.five != nil {
					value.FiveHour.ResetsAt = *quota.five
				}
			}
			f.quotas.set(value)
			if err := f.service.Check(context.Background()); err != nil {
				t.Fatal(err)
			}
			if f.sends.Load() != 0 {
				t.Fatal("non-unstarted quota sent a message")
			}
		})
	}
	for _, state := range []string{"disabled", "logged_out", "no_identity", "api_key", "provider_disabled"} {
		t.Run(state, func(t *testing.T) {
			f := newFixture(t)
			f.enable(t)
			switch state {
			case "disabled":
				f.accounts.item.Enabled = false
			case "logged_out":
				f.accounts.item.AuthState = "logged_out"
			case "no_identity":
				f.accounts.item.AuthAccountID = ""
			case "api_key":
				f.accounts.item.AuthMethod = account.AuthAPIKey
			case "provider_disabled":
				f.providers.enabled = false
			}
			f.service.Check(context.Background())
			if f.sends.Load() != 0 || f.quotas.reads.Load() != 0 {
				t.Fatal("ineligible account used")
			}
		})
	}
}
func ptr(value string) *string { return &value }

func TestUncertainResultSurvivesRestartAndConcurrentChecksDoNotResend(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	sender := sendFunc(func(context.Context, account.Item, SendPlan) error {
		f.sends.Add(1)
		return errors.New("private request error")
	})
	f.service.sender = sender
	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() { defer wg.Done(); f.service.Check(context.Background()) }()
	}
	wg.Wait()
	if f.sends.Load() != 1 {
		t.Fatal("concurrent checks resent uncertain message")
	}
	restarted := New(f.db, f.accounts, f.providers, f.quotas, sender)
	restarted.Check(context.Background())
	if f.sends.Load() != 1 || restarted.Status().Pending != 1 {
		t.Fatal("restart repeated uncertain message")
	}
	f.quotas.mu.Lock()
	f.quotas.err = errors.New("private quota error")
	f.quotas.mu.Unlock()
	restarted.Check(context.Background())
	if f.sends.Load() != 1 {
		t.Fatal("quota failure triggered message")
	}
}

func TestNewObservedQuotaCycleCanBeActivatedOnce(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	now := time.Now().UTC().Truncate(time.Second)
	f.service.now = func() time.Time { return now }
	f.service.sender = sendFunc(func(context.Context, account.Item, SendPlan) error {
		f.sends.Add(1)
		fresh := activeQuota()
		fresh.FiveHour.ResetsAt = now.Add(5 * time.Hour).Format(time.RFC3339)
		fresh.Weekly.ResetsAt = now.Add(7 * 24 * time.Hour).Format(time.RFC3339)
		f.quotas.set(fresh)
		return nil
	})
	f.service.Check(context.Background())
	active := activeQuota()
	active.FiveHour.ResetsAt = now.Add(5 * time.Hour).Format(time.RFC3339)
	// 推进真实周期时钟，而不是将已确认目标倒改成更早的过期快照。
	now = now.Add(5*time.Hour + time.Second)
	f.quotas.set(active)
	f.service.Check(context.Background())
	pending := unstartedQuota()
	pending.Weekly = active.Weekly
	f.quotas.set(pending)
	f.service.Check(context.Background())
	if f.sends.Load() != 2 {
		t.Fatal("new five-hour cycle not activated")
	}
	f.service.Check(context.Background())
	if f.sends.Load() != 2 {
		t.Fatal("new cycle repeated")
	}
}

func TestRecordFailureCannotSend(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	if _, err := f.db.Exec(`DROP TABLE codex_quota_activation_windows`); err != nil {
		t.Fatal(err)
	}
	f.service.Check(context.Background())
	if f.sends.Load() != 0 || f.service.Status().Failed != 1 {
		t.Fatal("record failure did not fail closed")
	}
}

func TestKnownPreparationFailureCanRetryWithoutDuplicateMessages(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	sender := f.service.sender
	f.service.sender = sendFunc(func(context.Context, account.Item, SendPlan) error { return ErrNotSent })
	f.service.Check(context.Background())
	if f.sends.Load() != 0 {
		t.Fatal("preparation failure sent a message")
	}
	f.service.sender = sender
	f.service.Check(context.Background())
	if f.sends.Load() != 1 {
		t.Fatal("unsent preparation failure permanently blocked activation")
	}
	f.service.Check(context.Background())
	if f.sends.Load() != 1 {
		t.Fatal("activation was repeated")
	}
}

func TestIdentityChangeBeforeSendIsRejected(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	current := f.accounts.item
	current.AuthAccountID = "other"
	f.accounts.metadata = &current
	f.service.Check(context.Background())
	if f.sends.Load() != 0 || f.service.Status().Failed != 1 {
		t.Fatal("changed identity sent a message")
	}
}

func TestDelayedQuotaConfirmationUpdatesWithoutResending(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	f.service.sender = sendFunc(func(context.Context, account.Item, SendPlan) error { f.sends.Add(1); return nil })
	notifications := 0
	f.service.OnChange(func() { notifications++ })
	f.service.Check(context.Background())
	if f.service.Status().Pending != 1 {
		t.Fatal("unconfirmed response reported activated")
	}
	f.quotas.set(activeQuota())
	f.service.Check(context.Background())
	if f.sends.Load() != 1 || notifications != 2 || f.service.Status().Activated != 1 {
		t.Fatal("delayed confirmation not reported without another message")
	}
	f.service.Check(context.Background())
	if notifications != 2 {
		t.Fatal("unchanged quota creates a refresh loop")
	}
}
