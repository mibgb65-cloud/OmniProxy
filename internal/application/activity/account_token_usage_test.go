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
	// Token 统计独立于模型价格和额度窗口，但不能合并其他本地账号、成员或工作区。
	for i, change := range []func(*proxy.RequestEvent){
		func(e *proxy.RequestEvent) { e.Target.AccountID = "other-account" },
		func(e *proxy.RequestEvent) { e.Target.Email = "other@example.invalid" },
		func(e *proxy.RequestEvent) { e.Target.WorkspaceID = "other-workspace" },
		func(e *proxy.RequestEvent) { e.Target.ProviderID = "other-provider" },
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
	if _, err := s.db.Exec(`DELETE FROM request_records WHERE id=?`, event(1, "", "").ID); err != nil {
		t.Fatal(err)
	}
	usage, err = s.accountTokenUsage(context.Background(), item)
	if err != nil || usage.InputTokens != "1000000" || usage.CachedInputTokens != "" {
		t.Fatal("cleared history or unknown cache was misrepresented", usage, err)
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
