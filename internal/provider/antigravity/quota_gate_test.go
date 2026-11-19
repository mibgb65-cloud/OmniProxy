package antigravity

import (
	"testing"
	"time"
)

func gateGroups() []QuotaGroup {
	return []QuotaGroup{{Name: "Gemini Models"}, {Name: "Claude and GPT models"}, {Name: "Gemini 4 Argon"}}
}

func percent(value float64) *float64 { return &value }

func TestQuotaGroupForModelClassifiesKnownFamilies(t *testing.T) {
	groups := gateGroups()
	for _, tc := range []struct{ model, want string }{
		{"gemini-4-neon-high", "Gemini Models"},
		{"gemini-3.8-flash", "Gemini Models"},
		{"gemini-4-argon-high", "Gemini 4 Argon"},
		{"claude-opus-4-6-thinking", "Claude and GPT models"},
		{"gpt-oss-120b", "Claude and GPT models"},
		{"Gemini-4-Neon-Low", "Gemini Models"},
		{"imagen-4", ""},
		{"", ""},
	} {
		if got := QuotaGroupForModel(tc.model, groups); got != tc.want {
			t.Errorf("QuotaGroupForModel(%q) = %q, want %q", tc.model, got, tc.want)
		}
	}
}

func TestQuotaGroupForModelKeepsArgonOutOfGeminiGroup(t *testing.T) {
	// 账号没有 Argon 分组时不猜：argon 模型不能落到 Gemini 主组，否则会按错的池跳过。
	groups := []QuotaGroup{{Name: "Gemini Models"}, {Name: "Claude and GPT models"}}
	if got := QuotaGroupForModel("gemini-4-argon-high", groups); got != "" {
		t.Fatalf("argon model without argon group = %q, want empty", got)
	}
}

func TestQuotaExhaustedUntilWaitsForEverySpentWindow(t *testing.T) {
	now := time.Date(2026, 11, 19, 2, 40, 0, 0, time.UTC)
	fiveHour := now.Add(25 * time.Minute).Format(time.RFC3339Nano)
	weekly := now.Add(6 * 24 * time.Hour).Format(time.RFC3339Nano)
	quota := Quota{Groups: []QuotaGroup{
		{Name: "Gemini Models", Buckets: []QuotaBucket{
			{ID: "gemini-weekly", Window: "weekly", RemainingPercent: percent(19.1), ResetsAt: weekly},
			{ID: "gemini-5h", Window: "5h", RemainingPercent: percent(0), ResetsAt: fiveHour},
		}},
		{Name: "Claude and GPT models", Buckets: []QuotaBucket{
			{ID: "3p-5h", Window: "5h", RemainingPercent: percent(100), ResetsAt: weekly},
		}},
	}}
	until, ok := QuotaExhaustedUntil(quota, "gemini-4-neon-high", now)
	if !ok || until.Format(time.RFC3339) != now.Add(25*time.Minute).Format(time.RFC3339) {
		t.Fatalf("until = %v, %v", until, ok)
	}
	// 第三方分组仍满额，不能被 Gemini 的 5h 耗尽牵连。
	if _, blocked := QuotaExhaustedUntil(quota, "claude-opus-4-6-thinking", now); blocked {
		t.Fatal("third-party model was blocked by the Gemini window")
	}
}

func TestQuotaExhaustedUntilTakesTheLatestSpentWindow(t *testing.T) {
	now := time.Date(2026, 11, 19, 2, 40, 0, 0, time.UTC)
	near := now.Add(20 * time.Minute).Format(time.RFC3339Nano)
	far := now.Add(5 * 24 * time.Hour).Format(time.RFC3339Nano)
	quota := Quota{Groups: []QuotaGroup{{Name: "Gemini Models", Buckets: []QuotaBucket{
		{ID: "gemini-5h", RemainingPercent: percent(0), ResetsAt: near},
		{ID: "gemini-weekly", RemainingPercent: percent(0), ResetsAt: far},
	}}}}
	until, ok := QuotaExhaustedUntil(quota, "gemini-4-neon-high", now)
	if !ok || until.Sub(now) < 4*24*time.Hour {
		t.Fatalf("until = %v, %v (weekly reset must still gate)", until, ok)
	}
}

func TestQuotaExhaustedUntilIgnoresUnknownOrStaleStates(t *testing.T) {
	now := time.Date(2026, 11, 19, 2, 40, 0, 0, time.UTC)
	future := now.Add(time.Hour).Format(time.RFC3339Nano)
	for name, quota := range map[string]Quota{
		"small positive balance": {Groups: []QuotaGroup{{Name: "Gemini Models", Buckets: []QuotaBucket{
			{RemainingPercent: percent(0.0001), ResetsAt: future},
		}}}},
		"unknown percent": {Groups: []QuotaGroup{{Name: "Gemini Models", Buckets: []QuotaBucket{
			{RemainingPercent: nil, ResetsAt: future},
		}}}},
		"already reset": {Groups: []QuotaGroup{{Name: "Gemini Models", Buckets: []QuotaBucket{
			{RemainingPercent: percent(0), ResetsAt: now.Add(-time.Minute).Format(time.RFC3339Nano)},
		}}}},
		"missing reset": {Groups: []QuotaGroup{{Name: "Gemini Models", Buckets: []QuotaBucket{
			{RemainingPercent: percent(0)},
		}}}},
		"unclassified model": {Groups: []QuotaGroup{{Name: "Gemini Models", Buckets: []QuotaBucket{
			{RemainingPercent: percent(0), ResetsAt: future},
		}}}},
	} {
		model := "gemini-4-neon-high"
		if name == "unclassified model" {
			model = "imagen-4"
		}
		if until, blocked := QuotaExhaustedUntil(quota, model, now); blocked {
			t.Fatalf("%s: unexpectedly blocked until %v", name, until)
		}
	}
}
