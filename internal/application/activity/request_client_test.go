package activity

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestRequestClientAttributionIsStoredAndSearchable(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	e := event(1, "a@example.invalid", "a")
	e.Client = "codex"
	record(t, s, e)
	dash := event(2, "a@example.invalid", "a")
	record(t, s, dash)
	page, err := s.Requests(ctx, RequestQuery{Limit: 10})
	if err != nil || len(page.Items) != 2 {
		t.Fatal("requests unavailable", page, err)
	}
	byID := map[string]string{}
	for _, item := range page.Items {
		byID[item.ID] = item.Client
	}
	// 直接调用或其他客户端产生的历史请求保持未知，不能猜测归属。
	if byID[e.ID] != "codex" || byID[dash.ID] != "" {
		t.Fatal("client attribution lost or fabricated", byID)
	}
	item, err := s.Request(ctx, e.ID)
	if err != nil || item.Client != "codex" {
		t.Fatal("detail lost client attribution", item, err)
	}
	filtered, err := s.Requests(ctx, RequestQuery{Search: "CODE", Limit: 10})
	if err != nil || len(filtered.Items) != 1 || filtered.Items[0].ID != e.ID {
		t.Fatal("client not searchable", filtered, err)
	}
	bad := event(3, "a@example.invalid", "a")
	bad.Client = strings.Repeat("x", 33)
	if err := s.Begin(ctx, bad); !errors.Is(err, ErrInvalid) {
		t.Fatal("oversized client accepted")
	}
}
