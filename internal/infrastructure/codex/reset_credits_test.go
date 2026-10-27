package codex

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestResetSummaryPreservesUnknownEmptyAndCountOnly(t *testing.T) {
	for _, raw := range []string{`null`, `{"availableCount":0,"credits":[]}`, `{"availableCount":3,"credits":null}`} {
		var summary *ResetCreditSummary
		if json.Unmarshal([]byte(raw), &summary) != nil || !validResetCredits(summary) {
			t.Fatal("valid reset summary rejected")
		}
	}
	// 缺少获得时间不再是无效元数据（官方 HTTP 汇总可能不返回），缺少卡片 ID 仍然无效。
	for _, raw := range []string{`{}`, `{"availableCount":-1}`, `{"availableCount":1,"credits":[{"status":"available","resetType":"codexRateLimits"}]}`, `{"availableCount":1,"credits":[{"id":"x","status":"available","expiresAt":253402300800}]}`, `{"availableCount":1,"credits":[{"id":"x","status":"available"},{"id":"x","status":"available"}]}`} {
		var summary *ResetCreditSummary
		if json.Unmarshal([]byte(raw), &summary) != nil {
			t.Fatal("test fixture invalid")
		}
		if validResetCredits(summary) {
			t.Fatal("invalid reset metadata accepted")
		}
	}
}

func TestResetProtocolUsesSameKeyAndOnlyOfficialOutcome(t *testing.T) {
	for _, outcome := range []string{"reset", "alreadyRedeemed", "nothingToReset", "noCredit", "unknown"} {
		client := &rpcClient{pending: map[uint64]chan wireResponse{}, done: make(chan struct{})}
		writer := &identityWriter{client: client, result: map[string]any{"outcome": outcome}}
		client.input = writer
		_, err := client.ConsumeReset(context.Background(), "11111111-2222-4333-8444-555555555555", "credit-test")
		if writer.method != "account/rateLimitResetCredit/consume" || (err == nil) != ValidResetOutcome(outcome) {
			t.Fatal("invalid consume method or outcome accepted")
		}
		if !strings.Contains(string(writer.request), `"idempotencyKey":"11111111-2222-4333-8444-555555555555"`) || !strings.Contains(string(writer.request), `"creditId":"credit-test"`) {
			t.Fatal("consume dropped key or selected credit")
		}
	}
}
