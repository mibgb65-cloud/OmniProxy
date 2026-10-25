package quotaactivation

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/subscription"
)

// slidingQuotas 复现官方对未开始窗口返回的随查询后移的占位目标。
type slidingQuotas struct {
	base  time.Time
	step  time.Duration
	reads atomic.Int32
}

func (q *slidingQuotas) ReadQuota(context.Context, string) (subscription.Quota, error) {
	at := q.base.Add(time.Duration(int(q.reads.Add(1))-1) * q.step)
	return subscription.Quota{
		FiveHourEnabled: true,
		FiveHour:        &subscription.QuotaWindow{RemainingPercent: 100, ResetsAt: at.Add(5 * time.Hour).UTC().Format(time.RFC3339)},
		Weekly:          &subscription.QuotaWindow{RemainingPercent: 100, ResetsAt: at.Add(7 * 24 * time.Hour).UTC().Format(time.RFC3339)},
		FetchedAt:       at.UTC().Format(time.RFC3339),
	}, nil
}

func (q *slidingQuotas) now() time.Time {
	return q.base.Add(time.Duration(q.reads.Load()) * q.step)
}

// 启动后不必等满一个完整检测周期：滑动候选在第二次独立观测后立即给出结论。
func TestBackgroundLoopCompletesSlidingObservationWithoutAFullInterval(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	quotas := &slidingQuotas{base: time.Now().UTC(), step: 35 * time.Second}
	f.service.quotas = quotas
	f.service.now = quotas.now
	f.service.interval = time.Hour
	f.service.fastInterval = 20 * time.Millisecond
	sent := make(chan int, 1)
	f.service.sender = sendFunc(func(context.Context, account.Item, SendPlan) error {
		f.sends.Add(1)
		sent <- int(quotas.reads.Load())
		return nil
	})
	f.service.Start(context.Background())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := f.service.Close(ctx); err != nil {
			t.Errorf("close: %v", err)
		}
	})
	select {
	case reads := <-sent:
		if reads < 2 {
			t.Fatalf("activation used %d observation(s); sliding target needs a second one", reads)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("sliding window was not re-observed within the fast interval")
	}
}

func TestActiveWindowDoesNotRequestFastReobservation(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	f.quotas.set(activeQuota())
	if err := f.service.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if f.service.reobserve.Load() {
		t.Fatal("already started window requested a fast re-observation")
	}
}

func TestCheckPublishesAnonymousStatusOncePerRound(t *testing.T) {
	f := newFixture(t)
	f.enable(t)
	var published []Status
	f.service.OnStatus(func(status Status) { published = append(published, status) })
	if err := f.service.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(published) != 1 {
		t.Fatalf("published %d status reports", len(published))
	}
	status := published[0]
	if status.Running || status.UpdatedAt == "" || status.Checked != 1 || status.Sent != 1 {
		t.Fatalf("unexpected status %+v", status)
	}
}
