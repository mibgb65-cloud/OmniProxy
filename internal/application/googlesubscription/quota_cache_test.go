package googlesubscription

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/antigravity"
)

func cachedQuotaFixture(t *testing.T) (*Service, account.Item, string, antigravity.Quota) {
	t.Helper()
	s, _ := setupService(t)
	item, err := s.accounts.SaveGoogleSubscription(context.Background(), "builtin_google", "", "", quotaCredential())
	if err != nil {
		t.Fatal(err)
	}
	version, err := s.accounts.CatalogVersion(context.Background(), item.ID)
	if err != nil {
		t.Fatal(err)
	}
	percent := 76.25
	value := antigravity.Quota{FetchedAt: "2026-10-20T12:00:00Z", Groups: []antigravity.QuotaGroup{{
		Name: "Gemini", Buckets: []antigravity.QuotaBucket{{ID: "gemini-week", Window: "weekly", RemainingPercent: &percent, ResetsAt: "2026-10-25T12:00:00Z"}},
	}}}
	return s, item, version, value
}

func TestGoogleQuotaPersistsAcrossDatabaseReopenAndRetainsMissingFields(t *testing.T) {
	ctx := context.Background()
	s, item, version, value := cachedQuotaFixture(t)
	if _, err := s.quotaCache.Save(ctx, item, version, value); err != nil {
		t.Fatal(err)
	}
	partial := value
	partial.Groups = []antigravity.QuotaGroup{{Name: "Gemini", Buckets: []antigravity.QuotaBucket{{ID: "gemini-week", Window: "weekly"}}}}
	partial.FetchedAt = "2026-10-20T12:01:00Z"
	saved, err := s.quotaCache.Save(ctx, item, version, partial)
	if err != nil || !saved.FromCache || saved.Groups[0].Buckets[0].RemainingPercent == nil || saved.Groups[0].Buckets[0].ResetsAt == "" {
		t.Fatal("partial read erased retained values", saved, err)
	}
	var path string
	if err := s.quotaCache.db.QueryRow(`SELECT file FROM pragma_database_list WHERE name='main'`).Scan(&path); err != nil {
		t.Fatal(err)
	}
	if err := s.quotaCache.db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := database.Open(database.Config{Path: filepath.Clean(path)})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	got, err := NewQuotaCache(db).Read(ctx, item, version)
	if err != nil || got == nil || !got.FromCache || *got.Groups[0].Buckets[0].RemainingPercent != 76.25 {
		t.Fatal("reopened database lost quota", got, err)
	}
}

func TestGoogleQuotaAuthorizationRotationAndDeletionBoundaries(t *testing.T) {
	for _, change := range []string{"rotate", "relogin", "logout", "email", "delete"} {
		t.Run(change, func(t *testing.T) {
			ctx := context.Background()
			s, item, version, value := cachedQuotaFixture(t)
			if _, err := s.quotaCache.Save(ctx, item, version, value); err != nil {
				t.Fatal(err)
			}
			var err error
			switch change {
			case "rotate":
				_, err = s.accounts.RotateGoogleCredentials(ctx, item, quotaCredential())
			case "relogin":
				_, err = s.accounts.SaveGoogleSubscription(ctx, item.ProviderID, item.ID, "", quotaCredential())
			case "logout":
				_, err = s.accounts.LogoutGoogleSubscription(ctx, item.ID)
			case "email":
				_, err = s.quotaCache.db.Exec(`UPDATE accounts SET auth_email='other@example.invalid' WHERE id=?`, item.ID)
			case "delete":
				_, err = s.quotaCache.db.Exec(`DELETE FROM accounts WHERE id=?`, item.ID)
			}
			if err != nil {
				t.Fatal(err)
			}
			if change == "delete" {
				var count int
				if err := s.quotaCache.db.QueryRow(`SELECT COUNT(*) FROM google_subscription_quota_snapshots`).Scan(&count); err != nil || count != 0 {
					t.Fatal("orphan quota persisted", err)
				}
				return
			}
			got, err := s.CachedQuota(ctx, item.ID)
			if err != nil || (got != nil) != (change == "rotate") {
				t.Fatal("quota crossed authorization or identity boundary", got, err)
			}
		})
	}
}

func TestGoogleQuotaOldInvalidAndChangedWindowCannotOverwrite(t *testing.T) {
	ctx := context.Background()
	s, item, version, value := cachedQuotaFixture(t)
	if _, err := s.quotaCache.Save(ctx, item, version, value); err != nil {
		t.Fatal(err)
	}
	stale := value
	stale.FetchedAt = "2026-10-20T11:59:59.999999999Z"
	got, err := s.quotaCache.Save(ctx, item, version, stale)
	if err != nil || !got.FromCache || got.FetchedAt == stale.FetchedAt {
		t.Fatal("older quota replaced latest", got, err)
	}
	next := value
	next.FetchedAt = "2026-10-20T12:00:00.000000001Z"
	next.Groups = []antigravity.QuotaGroup{{Name: "Gemini", Buckets: []antigravity.QuotaBucket{{ID: "gemini-week", Window: "weekly", ResetsAt: "2026-11-01T12:00:00Z"}}}}
	got, err = s.quotaCache.Save(ctx, item, version, next)
	if err != nil || got.Groups[0].Buckets[0].RemainingPercent != nil {
		t.Fatal("new window inherited old percentage", got, err)
	}
	bad := next
	bad.FetchedAt = time.Now().Format(time.RFC3339Nano)
	percent := 101.0
	bad.Groups[0].Buckets[0].RemainingPercent = &percent
	if _, err := s.quotaCache.Save(ctx, item, version, bad); err == nil {
		t.Fatal("invalid percentage saved")
	}
	if _, err := s.quotaCache.db.Exec(`UPDATE google_subscription_quota_snapshots SET snapshot_json='{"fetchedAt":"bad","token":"private-marker"}'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CachedQuota(ctx, item.ID); antigravity.ErrorCode(err) != "quota_cache_unavailable" {
		t.Fatal("malformed snapshot accepted", err)
	}
}
