package activity

import (
	"context"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

// 推理档位随请求明细保存：先到的空快照不能被后来的档位覆盖，后来的空快照也不能抹掉它。
func TestRequestEffortStoredWithoutFabricatingDefault(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	e := event(1, "a@example.invalid", "a")
	if err := s.Begin(ctx, e); err != nil {
		t.Fatal(err)
	}
	pending, err := s.Request(ctx, e.ID)
	if err != nil || pending.Effort != "" {
		t.Fatal("unset effort fabricated", pending, err)
	}
	snapshot := e
	snapshot.Effort = "high"
	snapshot.Usage = proxy.UsageSnapshot{Input: 10, Output: 4, InputKnown: true, OutputKnown: true, Complete: true}
	if err := s.Update(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	later := snapshot
	later.Effort = ""
	later.Usage = proxy.UsageSnapshot{Input: 12, Output: 5, InputKnown: true, OutputKnown: true, Complete: true}
	if err := s.Update(ctx, later); err != nil {
		t.Fatal(err)
	}
	item, err := s.Request(ctx, e.ID)
	if err != nil || item.Effort != "high" || item.InputTokens != "12" {
		t.Fatal("effort lost or usage wrong", item, err)
	}
}

// 档位来自客户端正文，超长或含控制字符的取值按未指定处理，不能让记账失败。
func TestRequestEffortRejectsUnusableValues(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	oversized := event(1, "a@example.invalid", "a")
	oversized.Effort = strings.Repeat("x", 33)
	record(t, s, oversized)
	item, err := s.Request(ctx, oversized.ID)
	if err != nil || item.Effort != "" {
		t.Fatal("oversized effort stored", item, err)
	}
	control := event(2, "a@example.invalid", "a")
	control.Effort = "hi\ngh"
	record(t, s, control)
	item, err = s.Request(ctx, control.ID)
	if err != nil || item.Effort != "" {
		t.Fatal("control characters stored", item, err)
	}
	page, err := s.Requests(ctx, RequestQuery{Limit: 10})
	if err != nil || len(page.Items) != 2 {
		t.Fatal("requests unavailable", page, err)
	}
}
