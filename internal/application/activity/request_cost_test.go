package activity

import (
	"context"
	"math"
	"omniproxy/internal/core/proxy"
	"testing"
)

func pricedEvent(id int, model string) proxy.RequestEvent {
	e := event(id, "a@example.invalid", "a")
	e.Model = model
	e.LatencyMS = 2000
	return e
}

func TestRequestCostAndCachedInput(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	cached := int64(400_000)
	known := pricedEvent(1, "gpt-6-luna")
	known.Usage = proxy.UsageSnapshot{Input: 1_000_000, Output: 100_000, CachedInput: &cached, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, known)
	unknown := pricedEvent(2, "gpt-6-luna")
	unknown.Usage = proxy.UsageSnapshot{Input: 1_000_000, Output: 100_000, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, unknown)
	unpriced := pricedEvent(3, "unlisted-model")
	unpriced.Usage = proxy.UsageSnapshot{Input: 1_000_000, Output: 100_000, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, unpriced)
	partial := pricedEvent(4, "gpt-6-luna")
	partial.Usage = proxy.UsageSnapshot{Input: 1_000_000, OutputKnown: true}
	record(t, s, partial)

	item, err := s.Request(ctx, known.ID)
	if err != nil || !item.CachedInputKnown || item.CachedInputTokens != "400000" {
		t.Fatal("cached input lost", item, err)
	}
	if !item.CostKnown || math.Abs(item.CostMin-0.114) > 1e-9 || math.Abs(item.CostMax-0.114) > 1e-9 {
		t.Fatal("known-cache cost", item.CostMin, item.CostMax)
	}

	item, _ = s.Request(ctx, unknown.ID)
	if item.CachedInputKnown || item.CachedInputTokens != "" {
		t.Fatal("unknown cache reported as a value")
	}
	// 缓存未知时按未命中和全部命中给出上下界，区间不能折叠成单值。
	if !item.CostKnown || math.Abs(item.CostMin-0.06) > 1e-9 || math.Abs(item.CostMax-0.15) > 1e-9 {
		t.Fatal("unknown-cache range", item.CostMin, item.CostMax)
	}

	item, _ = s.Request(ctx, unpriced.ID)
	if item.CostKnown || item.CostMin != 0 || item.CostMax != 0 {
		t.Fatal("unlisted model priced", item.CostKnown, item.CostMin, item.CostMax)
	}

	item, _ = s.Request(ctx, partial.ID)
	if item.CostKnown || item.UsageComplete {
		t.Fatal("incomplete usage priced", item.CostKnown, item.UsageComplete)
	}

	page, err := s.Requests(ctx, RequestQuery{Limit: 50})
	if err != nil || len(page.Items) != 4 {
		t.Fatal("list", err, len(page.Items))
	}
	for _, entry := range page.Items {
		if entry.ID == known.ID && (!entry.CachedInputKnown || !entry.CostKnown) {
			t.Fatal("list lost cached or cost")
		}
	}
}

func TestRequestCachedInputZeroStaysKnown(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	zero := int64(0)
	e := pricedEvent(1, "gpt-6-luna")
	e.Usage = proxy.UsageSnapshot{Input: 1_000_000, Output: 100_000, CachedInput: &zero, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, e)
	item, err := s.Request(ctx, e.ID)
	if err != nil || !item.CachedInputKnown || item.CachedInputTokens != "0" {
		t.Fatal("reported zero treated as unknown", item, err)
	}
	if !item.CostKnown || math.Abs(item.CostMin-0.15) > 1e-9 || math.Abs(item.CostMax-0.15) > 1e-9 {
		t.Fatal("zero-cache cost", item.CostMin, item.CostMax)
	}
}
