package subscription

import (
	"context"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/application/account"
)

func TestQuotaCacheSeparatesMembersWorkspacesAndRetainsSameIdentityReauthorization(t *testing.T) {
	f, item, profile, _ := savedQuotaFixture(t)
	ctx := context.Background()
	for _, change := range []func(*account.Item){
		func(a *account.Item) { a.AuthEmail = "other@example.test" },
		func(a *account.Item) { a.AuthAccountID = "other-workspace" },
		func(a *account.Item) { a.ProviderID = "other-provider" },
		func(a *account.Item) { a.ID = "other-account" },
	} {
		other := item
		change(&other)
		if value, err := f.manager.quotaCache.Read(ctx, other, profile); err != nil || value != nil {
			t.Fatal("snapshot crossed account identity boundary")
		}
	}
	if value, err := f.manager.quotaCache.Read(ctx, item, strings.Repeat("b", 32)); err != nil || value == nil {
		t.Fatal("same confirmed identity lost its snapshot on reauthorization")
	}
	unknown := item
	unknown.AuthAccountID = ""
	if value, err := f.manager.quotaCache.Read(ctx, unknown, strings.Repeat("b", 32)); err != nil || value != nil {
		t.Fatal("unconfirmed workspace reused another profile's snapshot")
	}
	if value, err := f.manager.quotaCache.Read(ctx, unknown, profile); err != nil || value == nil {
		t.Fatal("same profile could not bridge missing workspace metadata")
	}
}

func TestQuotaCacheRejectsInvalidOrStaleWritesWithoutClearingHistory(t *testing.T) {
	f, item, profile, previous := savedQuotaFixture(t)
	ctx := context.Background()
	for _, value := range []Quota{
		{Plan: "plus", FetchedAt: "invalid"},
		{Plan: "plus", FetchedAt: time.Now().Add(time.Hour).UTC().Format(time.RFC3339)},
		{Plan: "bad\x00plan", FetchedAt: previous.FetchedAt},
		{FetchedAt: previous.FetchedAt, Weekly: &QuotaWindow{RemainingPercent: 101}},
		{FetchedAt: previous.FetchedAt, Weekly: &QuotaWindow{RemainingPercent: -1}},
		{FetchedAt: previous.FetchedAt, Weekly: &QuotaWindow{ResetsAt: "invalid"}},
	} {
		if _, err := f.manager.quotaCache.Save(ctx, item, profile, value); err == nil {
			t.Fatal("invalid snapshot accepted")
		}
	}
	older := previous
	older.FetchedAt = time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	older.FiveHour = &QuotaWindow{RemainingPercent: 1}
	weekly := *older.Weekly
	weekly.ObservedAt = older.FetchedAt
	older.Weekly = &weekly
	if kept, err := f.manager.quotaCache.Save(ctx, item, profile, older); err != nil || kept.FiveHour.RemainingPercent != 75 {
		t.Fatal("older response overwrote current snapshot")
	}
	changed := item
	changed.UpdatedAt = "stale-version"
	if _, err := f.manager.quotaCache.Save(ctx, changed, profile, previous); err == nil {
		t.Fatal("stale account version wrote snapshot")
	}
	kept, err := f.manager.CachedQuota(ctx, item.ID)
	if err != nil || kept == nil || kept.FiveHour.RemainingPercent != 75 {
		t.Fatal("rejected writes erased saved history")
	}
}

func TestQuotaCacheDoesNotExpireDisplayOrInventRecovery(t *testing.T) {
	f, item, profile, _ := savedQuotaFixture(t)
	old := Quota{Plan: "plus", FiveHourEnabled: true, FetchedAt: "2020-01-01T00:00:00Z", FiveHour: &QuotaWindow{RemainingPercent: 0, ResetsAt: "2020-01-01T01:00:00Z"}, Weekly: &QuotaWindow{RemainingPercent: 25, ResetsAt: "2020-01-07T00:00:00Z"}}
	if _, err := f.db.Exec(`DELETE FROM subscription_quota_snapshots WHERE account_id=?`, item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.manager.quotaCache.Save(context.Background(), item, profile, old); err != nil {
		t.Fatal(err)
	}
	value, err := f.manager.CachedQuota(context.Background(), item.ID)
	if err != nil || value == nil || value.FiveHour.RemainingPercent != 0 || value.FiveHour.ResetsAt != old.FiveHour.ResetsAt {
		t.Fatal("old snapshot expired or was reset to full")
	}
}

func TestQuotaCacheCorruptionAndStorageFailureReturnSafeErrors(t *testing.T) {
	f, item, profile, previous := savedQuotaFixture(t)
	if _, err := f.db.Exec(`UPDATE subscription_quota_snapshots SET snapshot_json='{"plan":"private-extra","unknown":true}' WHERE account_id=?`, item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.manager.CachedQuota(context.Background(), item.ID); err == nil || err.Error() != "subscription: quota_cache_unavailable" {
		t.Fatal("corrupted snapshot was accepted or raw database contents exposed")
	}
	if _, err := f.db.Exec(`DROP TABLE subscription_quota_snapshots`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.manager.quotaCache.Save(context.Background(), item, profile, previous); err == nil || err.Error() != "subscription: quota_cache_unavailable" {
		t.Fatal("storage failure swallowed or exposed")
	}
}
