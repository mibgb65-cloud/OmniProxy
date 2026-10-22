package activity

import (
	"context"
	"errors"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/core/proxy"
)

func riverUsageEvent(id int, accountID string) proxy.RequestEvent {
	e := event(id, "", accountID)
	e.Target = proxy.TargetSnapshot{ProviderID: "builtin_river", ProviderKind: "river", AccountID: accountID, AuthMethod: "api_key"}
	return e
}

func TestRiverTokenUsageKeepsAccountAndProviderBoundaries(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	item := account.Item{ID: "river-a", ProviderID: "builtin_river", AuthMethod: "api_key"}
	cached := int64(80)
	e := riverUsageEvent(1, item.ID)
	e.Usage = proxy.UsageSnapshot{Input: 100, Output: 20, CachedInput: &cached, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, e)
	// 累计快照重复写入不得累计两次；未知缓存不得伪装为已知零。
	if err := s.Finish(ctx, e); err != nil {
		t.Fatal(err)
	}
	e = riverUsageEvent(2, item.ID)
	e.Usage = proxy.UsageSnapshot{Input: 50, Output: 5, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, e)
	for i, change := range []func(*proxy.RequestEvent){
		func(e *proxy.RequestEvent) { e.Target.AccountID = "river-b" },
		func(e *proxy.RequestEvent) { e.Target.ProviderID = "other-provider" },
		func(e *proxy.RequestEvent) { e.Target.ProviderKind = "other-kind" },
	} {
		e := riverUsageEvent(i+10, item.ID)
		change(&e)
		e.Usage = proxy.UsageSnapshot{Input: 1000, Output: 1000, InputKnown: true, OutputKnown: true, Complete: true}
		record(t, s, e)
	}
	got, err := s.RiverTokenUsage(ctx, item)
	if err != nil || got.InputTokens != "150" || got.OutputTokens != "25" ||
		got.CachedInputTokens != "80" || got.Requests != 2 || got.UnknownCacheRequests != 1 {
		t.Fatal("cross-account, duplicate or unknown usage", got, err)
	}
	if _, err := s.db.Exec(`DELETE FROM request_records WHERE id=?`, riverUsageEvent(1, "").ID); err != nil {
		t.Fatal(err)
	}
	got, err = s.RiverTokenUsage(ctx, item)
	if err != nil || got.InputTokens != "50" || got.CachedInputTokens != "" {
		t.Fatal("retention boundary ignored", got, err)
	}
}

func TestRiverTokenUsageEmptyPartialFailedAndInvalid(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	item := account.Item{ID: "river-a", ProviderID: "builtin_river", AuthMethod: "api_key"}
	got, err := s.RiverTokenUsage(ctx, item)
	if err != nil || got.InputTokens != "0" || got.OutputTokens != "0" || got.CachedInputTokens != "0" {
		t.Fatal("empty history must be known zero", got, err)
	}
	e := riverUsageEvent(1, item.ID)
	e.State, e.StatusCode = "failed", 502
	e.Usage = proxy.UsageSnapshot{Input: 30, InputKnown: true}
	record(t, s, e)
	pending := riverUsageEvent(2, item.ID)
	if err := s.Begin(ctx, pending); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(ctx, pending); err != nil {
		t.Fatal(err)
	}
	got, err = s.RiverTokenUsage(ctx, item)
	if err != nil || got.InputTokens != "30" || got.OutputTokens != "" || got.CachedInputTokens != "" || got.IncompleteRequests != 2 {
		t.Fatal("known partial failure usage discarded or fabricated", got, err)
	}
	for _, bad := range []account.Item{{}, {ID: item.ID, ProviderID: "builtin_openai", AuthMethod: "api_key"},
		{ID: item.ID, ProviderID: item.ProviderID, AuthMethod: "subscription_login"}} {
		if _, err := s.RiverTokenUsage(ctx, bad); !errors.Is(err, ErrInvalid) {
			t.Fatal("unsupported account accepted", err)
		}
	}
	if err := s.db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RiverTokenUsage(ctx, item); !errors.Is(err, ErrUnavailable) {
		t.Fatal("database failure swallowed", err)
	}
}
