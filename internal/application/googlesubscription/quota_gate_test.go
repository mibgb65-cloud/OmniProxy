package googlesubscription

import (
	"context"
	"testing"
	"time"

	"omniproxy/internal/provider/antigravity"
)

func gatePercent(value float64) *float64 { return &value }

// gateFixture 保存一份「Gemini 主组 5h 已耗尽、其它分组仍满额」的快照。
func gateFixture(t *testing.T) (*QuotaGate, *Service, string, string, time.Time) {
	t.Helper()
	ctx := context.Background()
	s, _ := setupService(t)
	item, err := s.accounts.SaveGoogleSubscription(ctx, "builtin_google", "", "", quotaCredential())
	if err != nil {
		t.Fatal(err)
	}
	version, err := s.accounts.CatalogVersion(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 19, 2, 40, 0, 0, time.UTC)
	stamp := now.Add(-time.Minute).Format(time.RFC3339Nano)
	quota := antigravity.Quota{FetchedAt: stamp, Groups: []antigravity.QuotaGroup{
		{Name: "Gemini Models", Buckets: []antigravity.QuotaBucket{
			{ID: "gemini-weekly", Window: "weekly", RemainingPercent: gatePercent(19.1), ResetsAt: now.Add(6 * 24 * time.Hour).Format(time.RFC3339Nano)},
			{ID: "gemini-5h", Window: "5h", RemainingPercent: gatePercent(0), ResetsAt: now.Add(25 * time.Minute).Format(time.RFC3339Nano)},
		}},
		{Name: "Claude and GPT models", Buckets: []antigravity.QuotaBucket{
			{ID: "3p-5h", Window: "5h", RemainingPercent: gatePercent(100), ResetsAt: now.Add(5 * time.Hour).Format(time.RFC3339Nano)},
		}},
	}}
	if _, err := s.quotaCache.Save(ctx, item, version, quota); err != nil {
		t.Fatal(err)
	}
	return NewQuotaGate(s.quotaCache.db), s, item.ID, item.UpdatedAt, now
}

func TestQuotaGateBlocksOnlyTheExhaustedGroup(t *testing.T) {
	ctx := context.Background()
	gate, _, accountID, updatedAt, now := gateFixture(t)

	until, blocked := gate.BlockedUntil(ctx, accountID, updatedAt, "gemini-4-neon-high", now)
	if !blocked || until.Sub(now) != 25*time.Minute {
		t.Fatalf("gemini block = %v, %v", until, blocked)
	}
	// 第三方分组仍满额：同一个账号的 Claude 模型不能被跳。
	if until, blocked := gate.BlockedUntil(ctx, accountID, updatedAt, "claude-opus-4-6-thinking", now); blocked {
		t.Fatalf("third-party model blocked until %v", until)
	}
	if _, blocked := gate.BlockedUntil(ctx, accountID, updatedAt, "imagen-4", now); blocked {
		t.Fatal("unclassified model was prescreened")
	}
	// 重置时间已过就不再阻断。
	if _, blocked := gate.BlockedUntil(ctx, accountID, updatedAt, "gemini-4-neon-high", now.Add(26*time.Minute)); blocked {
		t.Fatal("expired window still blocked")
	}
}

func TestQuotaGateRequiresCurrentAccountVersionAndLogin(t *testing.T) {
	ctx := context.Background()
	gate, s, accountID, updatedAt, now := gateFixture(t)
	model := "gemini-4-neon-high"

	if _, blocked := gate.BlockedUntil(ctx, accountID, "stale-version", model, now); blocked {
		t.Fatal("stale account version was prescreened")
	}
	if _, err := s.quotaCache.db.ExecContext(ctx, `UPDATE accounts SET auth_state='reauth_required' WHERE id=?`, accountID); err != nil {
		t.Fatal(err)
	}
	if _, blocked := gate.BlockedUntil(ctx, accountID, updatedAt, model, now); blocked {
		t.Fatal("signed-out account was prescreened")
	}
}

func TestQuotaGateIgnoresUnreadableSnapshots(t *testing.T) {
	ctx := context.Background()
	gate, s, accountID, updatedAt, now := gateFixture(t)
	model := "gemini-4-neon-high"

	for name, raw := range map[string]string{
		// 落库时有 json_valid 与长度约束，这里只覆盖「结构可解析但不可信」的形态。
		"unknown fields":  `{"groups":[],"fetchedAt":"2026-11-19T02:39:00Z","extra":1}`,
		"invalid fetched": `{"groups":[],"fetchedAt":"not-a-time"}`,
		"empty groups":    `{"groups":[],"fetchedAt":"2026-11-19T02:39:00Z"}`,
	} {
		if _, err := s.quotaCache.db.ExecContext(ctx, `UPDATE google_subscription_quota_snapshots SET snapshot_json=? WHERE account_id=?`, raw, accountID); err != nil {
			t.Fatal(err)
		}
		if _, blocked := gate.BlockedUntil(ctx, accountID, updatedAt, model, now); blocked {
			t.Fatalf("%s: prescreened from an untrusted snapshot", name)
		}
	}
	// 没有快照行、空账号或空模型时都不预筛。
	if _, err := s.quotaCache.db.ExecContext(ctx, `DELETE FROM google_subscription_quota_snapshots`); err != nil {
		t.Fatal(err)
	}
	if _, blocked := gate.BlockedUntil(ctx, accountID, updatedAt, model, now); blocked {
		t.Fatal("prescreened without a snapshot")
	}
	if _, blocked := gate.BlockedUntil(ctx, "", updatedAt, model, now); blocked {
		t.Fatal("prescreened without an account")
	}
	if _, blocked := gate.BlockedUntil(ctx, accountID, updatedAt, "", now); blocked {
		t.Fatal("prescreened without a model")
	}
}
