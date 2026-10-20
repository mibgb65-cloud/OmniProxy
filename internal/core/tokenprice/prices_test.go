package tokenprice

import (
	"math"
	"testing"
)

func TestPriceLookupAndCacheAwareCost(t *testing.T) {
	price, ok := Lookup("gpt-6.1-sol[1m]")
	cached := int64(800000)
	low, high := price.Cost(1000000, 100000, &cached)
	if !ok || math.Abs(low-1.48) > 1e-9 || high != low {
		t.Fatal("cache reads were charged twice or reference model mismatched", low, high)
	}
	low, high = price.Cost(1000000, 100000, nil)
	if math.Abs(low-1.1) > 1e-9 || math.Abs(high-3) > 1e-9 {
		t.Fatal("unknown historical cache counts became zero", low, high)
	}
	for _, unknown := range []string{"gpt-6.1-sol-new-unverified", "unknown", "glm-4-flash", "moonshot-v1-8k"} {
		if _, ok := Lookup(unknown); ok {
			t.Fatal("unknown or CNY model silently acquired a USD price", unknown)
		}
	}
}

// 2026-10-20 核对 https://developers.openai.com/api/docs/pricing 的标准短上下文价。
func TestCodexOfficialReferencePrices(t *testing.T) {
	for model, want := range map[string]Price{
		"gpt-6-astra":   {10, 1, 50},
		"gpt-6.1-sol":   {2, 0.1, 10},
		"gpt-6-luna":    {0.1, 0.01, 0.5},
		"gpt-5.6-sol":   {4, 0.4, 20},
		"gpt-5.3-codex": {1.75, 0.175, 14},
	} {
		if got, ok := Lookup(model); !ok || got != want {
			t.Errorf("%s: got %+v, want %+v", model, got, want)
		}
	}
}
