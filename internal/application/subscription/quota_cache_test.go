package subscription

import (
	"context"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/codex"
	"omniproxy/internal/infrastructure/database"
)

func savedQuotaFixture(t *testing.T) (*fixture, account.Item, string, Quota) {
	t.Helper()
	f := newFixture(t)
	view, client, _ := f.begin(t, "")
	client.authorize("cached@example.test")
	item := *awaitState(t, f.manager, view.ID, "completed").Account
	client.mu.Lock()
	reset := time.Now().Add(7 * 24 * time.Hour).Unix()
	windows := quotaSnapshot("plus", 25, 60)
	windows.Secondary.ResetsAt = &reset
	client.rateLimits = &codex.RateLimits{AccountID: item.AuthAccountID, Snapshot: windows}
	client.mu.Unlock()
	value, err := f.manager.ReadQuota(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, profile, err := f.accounts.SubscriptionProfile(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	return f, item, profile, value
}

func TestQuotaSnapshotSurvivesDatabaseReopenWithoutOfficialComponent(t *testing.T) {
	f, item, _, expected := savedQuotaFixture(t)
	var path string
	if err := f.db.QueryRow(`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&path); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(database.Config{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	restarted := New(account.NewService(db, f.store), f.manager.root, func(context.Context, string) (codex.Client, error) {
		t.Fatal("cache restore started official component")
		return nil, codex.ErrUnavailable
	})
	defer restarted.Close()
	restarted.SetQuotaCache(NewQuotaCache(db))
	value, err := restarted.CachedQuota(context.Background(), item.ID)
	if err != nil || value == nil || !value.FromCache || value.FiveHour.RemainingPercent != 75 || value.Weekly.RemainingPercent != 40 || value.FiveHour.ResetsAt != expected.FiveHour.ResetsAt || value.FetchedAt != expected.FetchedAt {
		t.Fatal("persisted quota or reset time lost", err)
	}
	if _, fresh := restarted.QuotaBalanceWeight(item.ID, item.UpdatedAt, time.Now()); fresh {
		t.Fatal("display snapshot became a fresh scheduling observation")
	}
	var raw string
	if err := db.QueryRow(`SELECT snapshot_json FROM subscription_quota_snapshots WHERE account_id=?`, item.ID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{item.AuthEmail, item.AuthAccountID, "access_token", "credential_ref", "resetCredits"} {
		if strings.Contains(raw, forbidden) {
			t.Fatal("snapshot persisted unrelated identity or authorization data")
		}
	}
}

func TestQuotaSnapshotFailureKeepsLastSuccessAndSubsequentSuccessUpdates(t *testing.T) {
	f, item, _, before := savedQuotaFixture(t)
	client := f.last
	client.mu.Lock()
	client.failQuota = true
	client.mu.Unlock()
	if _, err := f.manager.ReadDisplayQuota(context.Background(), item.ID); err == nil {
		t.Fatal("failed query was reported as success")
	}
	cached, err := f.manager.CachedQuota(context.Background(), item.ID)
	if err != nil || cached == nil || cached.FiveHour.RemainingPercent != 75 || cached.FiveHour.ResetsAt != before.FiveHour.ResetsAt {
		t.Fatal("failure erased previous snapshot")
	}
	client.mu.Lock()
	client.failQuota = false
	used := int32(50)
	client.rateLimits.Snapshot.Primary.UsedPercent = &used
	client.mu.Unlock()
	fresh, err := f.manager.ReadDisplayQuota(context.Background(), item.ID)
	if err != nil || fresh.FiveHour.RemainingPercent != 50 || fresh.FromCache {
		t.Fatal("new success did not replace the saved quota", err)
	}
}

func TestPartialQuotaRetainsDisplayButNotOperationalWindows(t *testing.T) {
	f, item, _, previous := savedQuotaFixture(t)
	client := f.last
	client.mu.Lock()
	client.rateLimits.Snapshot.Primary.ResetsAt = nil
	client.rateLimits.Snapshot.Secondary = nil
	client.mu.Unlock()
	raw, err := f.manager.ReadQuota(context.Background(), item.ID)
	if err != nil || raw.Weekly != nil || raw.FiveHour.ResetsAt != "" || raw.FromCache {
		t.Fatal("raw quota was contaminated by display history")
	}
	view, err := f.manager.ReadDisplayQuota(context.Background(), item.ID)
	if err != nil || !view.FromCache || view.Weekly == nil || view.Weekly.RemainingPercent != 40 || view.FiveHour.ResetsAt != previous.FiveHour.ResetsAt {
		t.Fatal("partial official response cleared previous display")
	}
	if _, fresh := f.manager.QuotaBalanceWeight(item.ID, item.UpdatedAt, time.Now()); fresh {
		t.Fatal("retained window was reused as a current balance weight")
	}
}

func TestQuotaSnapshotsRemainStoredAcrossLogoutAndDeleteWithAccount(t *testing.T) {
	f, item, _, _ := savedQuotaFixture(t)
	if _, err := f.manager.Logout(context.Background(), item.ID); err != nil {
		t.Fatal(err)
	}
	if cached, err := f.manager.CachedQuota(context.Background(), item.ID); err != nil || cached != nil {
		t.Fatal("logged-out account exposed snapshot")
	}
	var count int
	if err := f.db.QueryRow(`SELECT count(*) FROM subscription_quota_snapshots WHERE account_id=?`, item.ID).Scan(&count); err != nil || count != 1 {
		t.Fatal("logout erased display history")
	}
	if err := f.manager.Delete(context.Background(), item.ID); err != nil {
		t.Fatal(err)
	}
	if err := f.db.QueryRow(`SELECT count(*) FROM subscription_quota_snapshots WHERE account_id=?`, item.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("deleted account left an orphan snapshot")
	}
}
