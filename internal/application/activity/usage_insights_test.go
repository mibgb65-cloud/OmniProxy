package activity

import (
	"context"
	"fmt"
	"omniproxy/internal/core/proxy"
	"testing"
	"time"
)

func insightEvent(id int, providerID, providerKind, providerName, model string, state string) proxy.RequestEvent {
	e := event(id, "a@example.invalid", "a")
	e.Model = model
	e.Target.ProviderID = providerID
	e.Target.ProviderKind = providerKind
	e.Target.ProviderName = providerName
	e.State = state
	e.StatusCode = 200
	if state == "failed" {
		e.StatusCode = 500
		e.ErrorCode = "upstream_failed"
	}
	return e
}

func TestUsageInsightsAggregateProvidersModelDaysHoursAndQuality(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	now := time.Now()
	localNow := now.In(time.Local)
	// 选当前小时之前两个不同的整点，避免测试依赖运行时刻。
	firstStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), localNow.Hour(), 5, 0, 0, time.Local).Add(-3 * time.Hour)
	cached := int64(60)
	first := int64(300)

	openai := insightEvent(1, "builtin_openai", "openai", "OpenAI", "gpt-test", "success")
	openai.StartedAt = firstStart
	openai.FinishedAt = openai.StartedAt.Add(time.Second)
	openai.LatencyMS = 1000
	openai.Stream = true
	openai.FirstTokenMS = &first
	openai.Usage = proxy.UsageSnapshot{Input: 100, Output: 50, InputKnown: true, OutputKnown: true, Complete: true, CachedInput: &cached}
	record(t, s, openai)

	deepseek := insightEvent(2, "deepseek", "deepseek", "DeepSeek", "deepseek-chat", "failed")
	deepseek.StartedAt = firstStart.Add(time.Hour)
	deepseek.FinishedAt = deepseek.StartedAt.Add(3 * time.Second)
	deepseek.LatencyMS = 3000
	deepseek.Usage = proxy.UsageSnapshot{Input: 200, Output: 20, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, deepseek)

	previous := now.AddDate(0, 0, -45).Format(time.DateOnly)
	if _, err := s.db.Exec(`INSERT INTO usage_buckets(day,subject,model,provider_name,account_name,email,workspace_id,requests,completed_usage,input_tokens,output_tokens) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, previous, "old", "legacy-model", "", "", "", "", 2, 2, 5, 5); err != nil {
		t.Fatal(err)
	}

	info, err := s.Usage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if info.Previous.TotalTokens != "10" || info.Previous.Requests != 2 || info.Previous.CompleteUsage != 2 {
		t.Fatalf("previous window = %+v", info.Previous)
	}
	if len(info.Providers) != 2 || info.Providers[0].Key != "deepseek" || info.Providers[0].Name != "DeepSeek" || info.Providers[0].Summary.TotalTokens != "220" {
		t.Fatalf("providers = %+v", info.Providers)
	}
	if info.Providers[1].Key != "builtin_openai" || info.Providers[1].Summary.TotalTokens != "150" {
		t.Fatalf("provider ordering = %+v", info.Providers)
	}
	if len(info.ModelDays) != 2 || info.ModelDays[0].Day != openai.StartedAt.In(time.Local).Format(time.DateOnly) {
		t.Fatalf("model days = %+v", info.ModelDays)
	}
	openaiHour := openai.StartedAt.In(time.Local).Hour()
	deepseekHour := deepseek.StartedAt.In(time.Local).Hour()
	if len(info.Hours) != 24 || info.Hours[openaiHour].Hour != openaiHour || info.Hours[openaiHour].Summary.Requests != 1 || info.Hours[openaiHour].Summary.InputTokens != "100" {
		t.Fatalf("openai hour = %+v", info.Hours[openaiHour])
	}
	if info.Hours[deepseekHour].Summary.Requests != 1 || info.Hours[deepseekHour].Summary.TotalTokens != "220" {
		t.Fatalf("deepseek hour = %+v", info.Hours[deepseekHour])
	}
	quality := info.Reliability
	if quality.Requests != 2 || quality.Success != 1 || quality.Failed != 1 || quality.Cancelled != 0 {
		t.Fatalf("reliability = %+v", quality)
	}
	if quality.SuccessRate == nil || *quality.SuccessRate != 50 {
		t.Fatalf("success rate = %v", quality.SuccessRate)
	}
	if quality.AverageLatencyMS == nil || *quality.AverageLatencyMS != 2000 {
		t.Fatalf("latency = %v", quality.AverageLatencyMS)
	}
	if quality.AverageFirstTokenMS == nil || *quality.AverageFirstTokenMS != 300 || quality.FirstTokenSamples != 1 {
		t.Fatalf("first token = %v samples %d", quality.AverageFirstTokenMS, quality.FirstTokenSamples)
	}
	if info.Cache.CachedInputTokens != "60" || info.Cache.KnownRequests != 1 {
		t.Fatalf("cache = %+v", info.Cache)
	}
}

func TestUsageInsightsKeepOnlyTopModelsForDailyTrend(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	today := time.Now().In(time.Local).Format(time.DateOnly)
	for i := 0; i < 7; i++ {
		if _, err := s.db.Exec(`INSERT INTO usage_buckets(day,subject,model,provider_name,account_name,email,workspace_id,requests,completed_usage,input_tokens,output_tokens) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, today, "trend", fmt.Sprintf("m%d", i), "", "", "", "", 1, 1, (i+1)*10, 0); err != nil {
			t.Fatal(err)
		}
	}
	info, err := s.Usage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.ModelDays) != usageModelTrendLimit {
		t.Fatalf("model days = %+v", info.ModelDays)
	}
	for _, item := range info.ModelDays {
		if item.Model == "m0" {
			t.Fatal("lowest model should be aggregated as other")
		}
	}
}

func TestUsageInsightsEmptyProfile(t *testing.T) {
	s := fixture(t)
	info, err := s.Usage(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if info.AllTime.TotalTokens != "0" || info.AllTime.Requests != 0 || info.Previous.TotalTokens != "0" {
		t.Fatalf("empty totals = %+v %+v", info.AllTime, info.Previous)
	}
	if len(info.Days) != 0 || len(info.Providers) != 0 || len(info.Models) != 0 || len(info.ModelDays) != 0 || len(info.Hours) != 24 {
		t.Fatalf("empty breakdowns = %+v", info)
	}
	if info.Reliability.Requests != 0 || info.Reliability.SuccessRate != nil || info.Reliability.AverageLatencyMS != nil {
		t.Fatalf("empty reliability = %+v", info.Reliability)
	}
	if info.Cache.CachedInputTokens != "0" || info.Cache.KnownRequests != 0 {
		t.Fatalf("empty cache = %+v", info.Cache)
	}
}
