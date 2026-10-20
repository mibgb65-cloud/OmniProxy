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
