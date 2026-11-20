package activity

import (
	"context"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

func TestRecordedRequestsListsOnlySavedContent(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	first := event(1, "a@example.invalid", "a")
	first.Model, first.Client = "gpt-one", "codex"
	record(t, s, first)
	second := event(2, "b@example.invalid", "b")
	second.Model, second.Client = "gpt-two", "claude-code"
	record(t, s, second)
	// 只有一条请求保存了正文；另一条不能出现在记录页。
	content := `{"messages":[{"role":"user","content":"大家好，检查这段代码"}]}`
	if err := s.RecordRequestContent(ctx, proxy.RequestContentEvent{
		RequestID: first.ID, Content: content, OriginalBytes: len(content),
	}); err != nil {
		t.Fatal(err)
	}
	page, err := s.RecordedRequests(ctx, RecordedQuery{Limit: 20})
	if err != nil || page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	item := page.Items[0]
	if item.ID != first.ID || item.Model != "gpt-one" || item.Client != "codex" || item.Truncated ||
		item.OriginalBytes != int64(len(content)) || item.StoredBytes <= 0 {
		t.Fatalf("item=%+v", item)
	}
	if !strings.Contains(item.Preview, "检查这段代码") {
		t.Fatalf("preview=%q", item.Preview)
	}
	page, err = s.RecordedRequests(ctx, RecordedQuery{Search: "claude-code", Limit: 20})
	if err != nil || page.Total != 0 {
		t.Fatalf("search matched request without content: %+v %v", page, err)
	}
	page, err = s.RecordedRequests(ctx, RecordedQuery{Search: "gpt-one", Limit: 20})
	if err != nil || page.Total != 1 {
		t.Fatalf("search by model failed: %+v %v", page, err)
	}
}

func TestRecordedRequestsPaginationAndValidation(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	for id := 1; id <= 3; id++ {
		request := event(id, "a@example.invalid", "a")
		record(t, s, request)
		content := `{"messages":[{"role":"user","content":"请求 ` + strings.Repeat("x", id) + `"}]}`
		if err := s.RecordRequestContent(ctx, proxy.RequestContentEvent{
			RequestID: request.ID, Content: content, OriginalBytes: len(content),
		}); err != nil {
			t.Fatal(err)
		}
	}
	page, err := s.RecordedRequests(ctx, RecordedQuery{Limit: 2})
	if err != nil || page.Total != 3 || len(page.Items) != 2 {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	second, err := s.RecordedRequests(ctx, RecordedQuery{Limit: 2, Offset: 2})
	if err != nil || len(second.Items) != 1 {
		t.Fatalf("second page=%+v err=%v", second, err)
	}
	for _, q := range []RecordedQuery{{Limit: 0}, {Limit: 51}, {Limit: 20, Offset: -1}, {Limit: 20, Search: "\x00"}} {
		if _, err := s.RecordedRequests(ctx, q); err == nil {
			t.Fatalf("invalid query accepted: %+v", q)
		}
	}
}
