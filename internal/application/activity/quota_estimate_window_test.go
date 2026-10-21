package activity

import (
	"context"
	"testing"
	"time"

	"omniproxy/internal/application/subscription"
)

func TestQuotaEstimateWeeklyRetainsFullWindowAcrossFiveHourReset(t *testing.T) {
	s, item, quota, now := estimateFixture(t)
	start, err := time.Parse(time.RFC3339, quota.Weekly.ResetsAt)
	if err != nil {
		t.Fatal(err)
	}
	start = start.Add(-7 * 24 * time.Hour)
	// 周窗口起点包含在内；前一秒排除，多个旧 5h 周期均应保留。
	for i, at := range []time.Time{start.Add(-time.Second), start, now.Add(-3 * 24 * time.Hour), now.Add(-3 * time.Hour), now.Add(-time.Hour)} {
		estimateEvent(t, s, i+1, at, "gpt-6.1-sol", nil)
	}
	before, err := s.EstimateQuota(context.Background(), item, quota, now)
	if err != nil {
		t.Fatal(err)
	}
	if before.FiveHour.Requests != 1 || before.Weekly.Requests != 4 || !closeEnough(before.Weekly.Total, 60) || !closeEnough(before.Weekly.Remaining, 48) {
		t.Fatal("weekly window omitted old 5h periods or included expired usage", before)
	}
	// 5h 自然重置进入新窗口，周快照不变，不能丢弃上一 5h 周期的消费。
	quota.FiveHour = &subscription.QuotaWindow{RemainingPercent: 90, ResetsAt: now.Add(5 * time.Hour).Format(time.RFC3339)}
	after, err := s.EstimateQuota(context.Background(), item, quota, now)
	if err != nil {
		t.Fatal(err)
	}
	if after.FiveHour.Status != "no_usage" || *after.Weekly != *before.Weekly {
		t.Fatal("5h reset changed weekly estimate", after)
	}
}

func TestQuotaEstimateWeeklyIgnoresInvalidLegacyBaseline(t *testing.T) {
	s, item, quota, now := estimateFixture(t)
	if _, err := s.db.Exec(`INSERT INTO accounts(id,provider_id,name,created_at,updated_at) VALUES('a','builtin_openai','test','now','now')`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO subscription_quota_estimate_baselines VALUES(?,?,?,?)`, item.ID, "weekly", quota.Weekly.ResetsAt, "invalid"); err != nil {
		t.Fatal(err)
	}
	estimateEvent(t, s, 1, now.Add(-24*time.Hour), "gpt-6.1-sol", nil)
	got, err := s.EstimateQuota(context.Background(), item, quota, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Weekly.Status != "ready" || got.Weekly.ResetAdjusted || got.Weekly.Requests != 1 {
		t.Fatal("unused legacy weekly baseline blocked full-window estimate", got.Weekly)
	}
}
