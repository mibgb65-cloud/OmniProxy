package activity

import (
	"context"
	"testing"
	"time"

	"omniproxy/internal/application/subscription"
	"omniproxy/internal/core/proxy"
)

func TestAccountTokenUsageRetainedIdentityAndUnknownCache(t *testing.T) {
	s, item, _, now := estimateFixture(t)
	cached := int64(800000)
	estimateEvent(t, s, 1, now.Add(-time.Hour), "gpt-6.1-sol", &cached)
	estimateEvent(t, s, 2, now.Add(-8*24*time.Hour), "unpriced-model", nil)
	// Token 统计独立于模型价格和额度窗口，也不能合并其他成员、工作区或提供商类型。
	// 身份与记账索引、用量页一致：订阅身份为「提供商类型 + 邮箱 + 工作区」，
	// 同一成员的本地账号行删除重加后继续合并历史，不按本地账号 ID 或提供商配置 ID 拆分。
	for i, change := range []func(*proxy.RequestEvent){
		func(e *proxy.RequestEvent) { e.Target.Email = "other@example.invalid" },
		func(e *proxy.RequestEvent) { e.Target.WorkspaceID = "other-workspace" },
		func(e *proxy.RequestEvent) { e.Target.ProviderKind = "other-kind" },
	} {
		e := event(i+10, item.AuthEmail, item.ID)
		e.Usage = proxy.UsageSnapshot{Input: 100, Output: 50, InputKnown: true, OutputKnown: true, Complete: true}
		change(&e)
		record(t, s, e)
	}
	item.AuthEmail = " A@EXAMPLE.INVALID "
	got, err := s.EstimateQuota(context.Background(), item, subscription.Quota{}, now)
	if err != nil || got.Usage == nil {
		t.Fatal(got, err)
	}
	usage := got.Usage
	if usage.InputTokens != "2000000" || usage.OutputTokens != "200000" || usage.CachedInputTokens != "800000" ||
		usage.UnknownCacheRequests != 1 || usage.Requests != 2 || usage.IncompleteRequests != 0 {
		t.Fatal("usage crossed identity, window, or price boundaries", usage)
	}
	if got.Weekly.Status != "missing_window" {
		t.Fatal("tokens required a quota snapshot", got)
	}
	// 明细删除不能让账号累计变小：卡片数值是该身份的终身值，与留存明细无关。
	if _, err := s.db.Exec(`DELETE FROM request_records WHERE id=?`, event(1, "", "").ID); err != nil {
		t.Fatal(err)
	}
	usage, err = s.accountTokenUsage(context.Background(), item)
	if err != nil || usage.InputTokens != "2000000" || usage.OutputTokens != "200000" ||
		usage.CachedInputTokens != "800000" || usage.Requests != 2 || usage.UnknownCacheRequests != 1 {
		t.Fatal("deleted detail reduced the lifetime total", usage, err)
	}
}

func TestAccountTokenUsageZeroUnknownAndPartial(t *testing.T) {
	for _, kind := range []string{"empty", "explicit_zero", "unknown", "pending", "partial"} {
		t.Run(kind, func(t *testing.T) {
			s, item, _, now := estimateFixture(t)
			if kind != "empty" {
				e := event(1, item.AuthEmail, item.ID)
				switch kind {
				case "explicit_zero":
					zero := int64(0)
					e.Usage = proxy.UsageSnapshot{InputKnown: true, OutputKnown: true, Complete: true, CachedInput: &zero}
				case "pending":
					e.Usage = proxy.UsageSnapshot{Input: 123, InputKnown: true}
					if err := s.Begin(context.Background(), e); err != nil {
						t.Fatal(err)
					}
					if err := s.Update(context.Background(), e); err != nil {
						t.Fatal(err)
					}
				case "partial":
					e.Usage = proxy.UsageSnapshot{Output: 42, OutputKnown: true}
				}
				if kind != "pending" {
					record(t, s, e)
				}
			}
			result, err := s.EstimateQuota(context.Background(), item, subscription.Quota{}, now)
			if err != nil {
				t.Fatal(err)
			}
			u := result.Usage
			switch kind {
			case "empty", "explicit_zero":
				if u.InputTokens != "0" || u.OutputTokens != "0" || u.CachedInputTokens != "0" || u.IncompleteRequests != 0 || u.PendingRequests != 0 {
					t.Fatal("known zero rendered as unknown", u)
				}
			case "unknown":
				if u.InputTokens != "" || u.OutputTokens != "" || u.CachedInputTokens != "" || u.IncompleteRequests != 1 || u.PendingRequests != 0 {
					t.Fatal("missing usage fabricated zero", u)
				}
			case "pending":
				// 只有仍在进行中的记录计入 PendingRequests；历史未完整记录只影响 IncompleteRequests。
				if u.InputTokens != "123" || u.OutputTokens != "" || u.CachedInputTokens != "" || u.IncompleteRequests != 1 || u.PendingRequests != 1 {
					t.Fatal("pending known usage was hidden or invented", u)
				}
			case "partial":
				if u.InputTokens != "" || u.OutputTokens != "42" || u.IncompleteRequests != 1 || u.PendingRequests != 0 {
					t.Fatal("partial output was discarded", u)
				}
			}
		})
	}
}

// 用户要求账号卡片显示该账号所有 Token、不分时间：明细淘汰和账号删除都不能让数值变小。
func TestAccountTokenUsageKeepsLifetimeAfterEvictionAndDeletion(t *testing.T) {
	s, item, _, _ := estimateFixture(t)
	ctx := context.Background()
	if _, err := s.db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at)VALUES(?,'builtin_openai','A','unused',1,'now','now')`, item.ID); err != nil {
		t.Fatal(err)
	}
	cached := int64(400)
	for i := 1; i <= 105; i++ {
		e := event(i, item.AuthEmail, item.ID)
		e.Usage = proxy.UsageSnapshot{Input: 1000, Output: 100, InputKnown: true, OutputKnown: true, Complete: true, CachedInput: &cached}
		record(t, s, e)
	}
	// 保留上限调到 100 会淘汰 5 条明细，卡片仍必须是终身值。
	if _, err := s.SetRetention(ctx, 100); err != nil {
		t.Fatal(err)
	}
	if stored, err := s.Retention(ctx); err != nil || stored.StoredRecords != 100 {
		t.Fatal("detail was not evicted", stored, err)
	}
	usage, err := s.accountTokenUsage(ctx, item)
	if err != nil || usage.InputTokens != "105000" || usage.OutputTokens != "10500" ||
		usage.CachedInputTokens != "42000" || usage.Requests != 105 ||
		usage.IncompleteRequests != 0 || usage.UnknownCacheRequests != 0 {
		t.Fatal("evicted detail reduced the lifetime total", usage, err)
	}
	// 删除本地账号行同样不减少累计：累计属于记账身份。
	if _, err := s.db.Exec(`DELETE FROM accounts WHERE id=?`, item.ID); err != nil {
		t.Fatal(err)
	}
	if again, err := s.accountTokenUsage(ctx, item); err != nil || again.InputTokens != "105000" || again.Requests != 105 {
		t.Fatal("account deletion reduced the lifetime total", again, err)
	}
}

// 同一请求的多次快照不能让累计重复或回退：缺少缓存字段的中间快照既不把已知缓存改成未知，
// 也不能让之后重复上报的同一份缓存再计入一次。
func TestAccountTokenUsageCachedSnapshotsNeverDoubleCount(t *testing.T) {
	s, item, _, _ := estimateFixture(t)
	ctx := context.Background()
	cached := int64(800)
	e := event(1, item.AuthEmail, item.ID)
	e.Usage = proxy.UsageSnapshot{Input: 1000, Output: 10, InputKnown: true, OutputKnown: true, Complete: true, CachedInput: &cached}
	if err := s.Begin(ctx, e); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(ctx, e); err != nil {
		t.Fatal(err)
	}
	withoutCache := e
	withoutCache.Usage = proxy.UsageSnapshot{Input: 1000, Output: 20, InputKnown: true, OutputKnown: true}
	if err := s.Update(ctx, withoutCache); err != nil {
		t.Fatal(err)
	}
	repeated := e
	repeated.Usage = proxy.UsageSnapshot{Input: 1000, Output: 20, InputKnown: true, OutputKnown: true, Complete: true, CachedInput: &cached}
	if err := s.Update(ctx, repeated); err != nil {
		t.Fatal(err)
	}
	if err := s.Finish(ctx, repeated); err != nil {
		t.Fatal(err)
	}
	usage, err := s.accountTokenUsage(ctx, item)
	if err != nil || usage.InputTokens != "1000" || usage.OutputTokens != "20" ||
		usage.CachedInputTokens != "800" || usage.Requests != 1 || usage.UnknownCacheRequests != 0 {
		t.Fatal("cached snapshot double counted or downgraded to unknown", usage, err)
	}
	detail, err := s.Request(ctx, e.ID)
	if err != nil || detail.CachedInputTokens != "800" {
		t.Fatal("known cache detail was downgraded to unknown", detail, err)
	}
}
