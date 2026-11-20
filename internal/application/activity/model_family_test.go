package activity

import (
	"context"
	"reflect"
	"testing"
	"time"

	"omniproxy/internal/core/proxy"
)

func TestModelFamiliesOnlyMergeConfirmedEffortVariants(t *testing.T) {
	families := modelFamilies([]string{
		"gemini-4-neon-high",
		"gemini-4-neon-xhigh",
		"gemini-4-neon-low",
		"gemini-4-carbon-high",
		"gemini-4-hydrogen-tiered-high",
		"gemini-4-hydrogen-tiered-xhigh",
		"gemini-2.5-flash",
		"gemini-2.5-flash-lite",
		"gemini-2.5-flash-thinking",
		"claude-opus-4-6-thinking",
		"gpt-5.6-luna",
		"",
	})
	expected := map[string]string{
		"gemini-4-neon-high":             "gemini-4-neon",
		"gemini-4-neon-xhigh":            "gemini-4-neon",
		"gemini-4-neon-low":              "gemini-4-neon",
		"gemini-4-hydrogen-tiered-high":  "gemini-4-hydrogen-tiered",
		"gemini-4-hydrogen-tiered-xhigh": "gemini-4-hydrogen-tiered",
	}
	if !reflect.DeepEqual(families, expected) {
		t.Fatalf("families = %#v", families)
	}
	// 只有一个档位后缀、基础名也不存在时不认定它是档位。
	if _, merged := families["gemini-4-carbon-high"]; merged {
		t.Fatal("single level suffix without a base model was merged")
	}
}

func TestModelFamiliesMergeVariantWhenBaseModelExists(t *testing.T) {
	families := modelFamilies([]string{"gemini-4-carbon", "gemini-4-carbon-high"})
	if families["gemini-4-carbon-high"] != "gemini-4-carbon" {
		t.Fatalf("base model did not confirm the level suffix: %#v", families)
	}
}

func TestProviderModelsMergeEffortVariants(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	for i, row := range []struct {
		model  string
		input  int64
		output int64
	}{
		{"gemini-4-neon-high", 100, 10},
		{"gemini-4-neon-xhigh", 200, 20},
		{"gemini-4-carbon-high", 30, 3},
		{"gemini-3.8-flash", 7, 1},
		{"gemini-3.8-flash-high", 20, 2},
	} {
		e := event(i+1, "a@example.invalid", "a")
		e.Model = row.model
		e.Target.ProviderID = "builtin_google"
		e.Target.ProviderKind = "google"
		e.Target.ProviderName = "Google"
		e.Usage = proxy.UsageSnapshot{Input: row.input, Output: row.output, InputKnown: true, OutputKnown: true, Complete: true}
		record(t, s, e)
	}
	info, err := s.Usage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.ProviderModels) != 3 {
		t.Fatalf("provider models were not merged by family: %+v", info.ProviderModels)
	}
	byName := map[string]ProviderModel{}
	for _, item := range info.ProviderModels {
		byName[item.Model] = item
	}
	merged := byName["gemini-4-neon"]
	if merged.Summary.TotalTokens != "330" || merged.Summary.Requests != 2 {
		t.Fatalf("merged provider model = %+v", merged)
	}
	if !reflect.DeepEqual(merged.Variants, []string{"gemini-4-neon-high", "gemini-4-neon-xhigh"}) {
		t.Fatalf("merged variants = %v", merged.Variants)
	}
	// 基础模型存在时，单个档位变体也归并到基础模型，并保留原始型号。
	base := byName["gemini-3.8-flash"]
	if base.Summary.TotalTokens != "30" || base.Summary.Requests != 2 {
		t.Fatalf("variant without base row not merged: %+v", base)
	}
	if !reflect.DeepEqual(base.Variants, []string{"gemini-3.8-flash", "gemini-3.8-flash-high"}) {
		t.Fatalf("variants of confirmed family = %v", base.Variants)
	}
	lone := byName["gemini-4-carbon-high"]
	if lone.ProviderID == "" || lone.Variants != nil {
		t.Fatalf("unconfirmed variant was changed: %+v", lone)
	}
	if _, merged := byName["gemini-4-carbon"]; merged {
		t.Fatal("unconfirmed variant was renamed")
	}
	// 模型占比与每日趋势使用同一家族口径。
	if len(info.Models) != 3 || info.Models[0].Key != "gemini-4-neon" || info.Models[0].Summary.TotalTokens != "330" {
		t.Fatalf("models breakdown = %+v", info.Models)
	}
	modelDay := map[string]string{}
	for _, item := range info.ModelDays {
		modelDay[item.Model] = item.Summary.TotalTokens
	}
	if modelDay["gemini-4-neon"] != "330" || modelDay["gemini-3.8-flash"] != "30" {
		t.Fatalf("model days were not merged by family: %+v", info.ModelDays)
	}
}

func TestModelDaysSelectTopFamiliesAfterMerge(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	now := time.Now()
	// 两个档位变体各自量不大，合并后应挤进前六并整体出现，而不是各自占一个名额。
	rows := []struct {
		model string
		input int64
	}{
		{"gemini-4-neon-high", 500},
		{"gemini-4-neon-xhigh", 500},
		{"model-a", 400},
		{"model-b", 300},
		{"model-c", 200},
		{"model-d", 100},
		{"model-e", 90},
		{"model-f", 80},
	}
	for i, row := range rows {
		e := event(i+1, "a@example.invalid", "a")
		e.StartedAt = now.Add(-time.Duration(i) * time.Minute)
		e.Model = row.model
		e.Usage = proxy.UsageSnapshot{Input: row.input, InputKnown: true, OutputKnown: true, Complete: true}
		record(t, s, e)
	}
	info, err := s.Usage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	byModel := map[string]bool{}
	for _, item := range info.ModelDays {
		byModel[item.Model] = true
	}
	if !byModel["gemini-4-neon"] || byModel["gemini-4-neon-high"] || byModel["gemini-4-neon-xhigh"] {
		t.Fatalf("model trend did not merge the variant family: %+v", info.ModelDays)
	}
	// 只有前六家族进入趋势；合并后的家族算出 1000 Token，应排在 model-a 之前。
	if len(byModel) != 6 {
		t.Fatalf("model trend size = %d (%+v)", len(byModel), info.ModelDays)
	}
	if byModel["model-f"] {
		t.Fatal("seventh family leaked into the trend after merging")
	}
}

func TestDashboardModelsMergeEffortVariants(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	local := time.Now().In(time.Local)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
	query := DashboardQuery{Boundaries: []int64{start.Unix(), start.AddDate(0, 0, 1).Unix()}}
	for i, row := range []struct {
		model  string
		input  int64
		output int64
	}{
		{"gemini-4-neon-high", 100, 50},
		{"gemini-4-neon-xhigh", 100, 50},
		{"gpt-5.6-luna", 10, 5},
	} {
		e := event(i+1, "a@example.invalid", "a")
		e.StartedAt = start.Add(time.Duration(9+i) * time.Hour)
		e.FinishedAt = e.StartedAt.Add(time.Second)
		e.Model = row.model
		e.Usage = proxy.UsageSnapshot{Input: row.input, Output: row.output, InputKnown: true, OutputKnown: true, Complete: true}
		record(t, s, e)
	}
	info, err := s.Dashboard(ctx, query)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.Models) != 2 {
		t.Fatalf("dashboard models were not merged by family: %+v", info.Models)
	}
	if info.Models[0].Model != "gemini-4-neon" || info.Models[0].TotalTokens != "300" || info.Models[0].Requests != 2 {
		t.Fatalf("dashboard top model = %+v", info.Models[0])
	}
	if info.Models[1].Model != "gpt-5.6-luna" {
		t.Fatalf("dashboard second model = %+v", info.Models[1])
	}
}
