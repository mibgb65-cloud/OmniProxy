package wails

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/platform/profile"
)

func TestSubscriptionEstimateWithoutQuotaSnapshotStillReturnsLocalUsage(t *testing.T) {
	prof, err := profile.New(profile.Test)
	if err != nil {
		t.Fatal(err)
	}
	prof.DataDir = t.TempDir()
	db, err := database.Open(database.Config{Path: prof.DatabasePath()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	s := NewAppService(prof, nil, db, nil, nil)
	s.activity = activity.New(db)
	accounts, err := s.accountService()
	if err != nil {
		t.Fatal(err)
	}
	item, err := accounts.CreateSubscription(context.Background(), "builtin_openai", "合成用量账号",
		"0123456789abcdef0123456789abcdef", account.SubscriptionMetadata{
			Email: "test@example.invalid", AccountID: "test-workspace", State: "authenticated",
		})
	if err != nil {
		t.Fatal(err)
	}
	// 无真实授权、无快照也能返回已知零；不允许启动官方 CLI 或进行网络查询。
	result, err := s.SubscriptionQuotaEstimate(item.ID)
	if err != nil || result.Usage == nil || result.Usage.InputTokens != "0" || result.FetchedAt != "" {
		t.Fatal(result, err)
	}
	if _, err := db.Exec(`UPDATE accounts SET auth_state='logged_out' WHERE id=?`, item.ID); err != nil {
		t.Fatal(err)
	}
	result, err = s.SubscriptionQuotaEstimate(item.ID)
	if err != nil || result.Usage != nil {
		t.Fatal("logged-out account exposed a live summary", result, err)
	}
	for _, id := range []string{"", strings.Repeat("a", 129), "a\nb"} {
		if _, err := s.SubscriptionQuotaEstimate(id); !errors.Is(err, activity.ErrInvalid) {
			t.Fatal("invalid binding input accepted", err)
		}
	}
}
