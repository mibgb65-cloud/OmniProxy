package activity

import (
	"context"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/core/proxy"
)

func TestContentCaptureStoresFullCompressedBody(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	enabled, err := s.ContentCaptureEnabled(ctx)
	if err != nil || enabled {
		t.Fatal("content capture must default off", enabled, err)
	}
	retention, err := s.Retention(ctx)
	if err != nil || retention.ContentEnabled || retention.ContentBudget != ContentStoreBudget || retention.ContentBytes != 0 {
		t.Fatal("retention content fields wrong", retention, err)
	}
	e := event(1, "a@example.invalid", "a")
	record(t, s, e)
	// 完整保存：内容超过旧的 256 KiB 上限，读取必须逐字节一致。
	content := `{"messages":[{"role":"user","content":"` + strings.Repeat("中", 200<<10) + `"}]}`
	if err := s.RecordRequestContent(ctx, proxy.RequestContentEvent{RequestID: e.ID, Content: content, OriginalBytes: len(content)}); err != nil {
		t.Fatal(err)
	}
	// 重复写入保留首份，避免后到的快照改写正文。
	if err := s.RecordRequestContent(ctx, proxy.RequestContentEvent{RequestID: e.ID, Content: "other", OriginalBytes: 5}); err != nil {
		t.Fatal(err)
	}
	got, err := s.RequestContent(ctx, e.ID)
	if err != nil || !got.Available || got.Content != content || int64(len(content)) != got.OriginalBytes || got.Truncated {
		t.Fatalf("stored content mismatch: available=%v equal=%v original=%d stored=%d truncated=%v err=%v",
			got.Available, got.Content == content, got.OriginalBytes, got.StoredBytes, got.Truncated, err)
	}
	if got.StoredBytes <= 0 || got.StoredBytes >= got.OriginalBytes {
		t.Fatalf("content was not compressed: stored=%d original=%d", got.StoredBytes, got.OriginalBytes)
	}
	retention, err = s.Retention(ctx)
	if err != nil || retention.ContentBytes != got.StoredBytes {
		t.Fatalf("content usage not reported: %+v %v", retention, err)
	}
	missing, err := s.RequestContent(ctx, strings.Repeat("b", 32))
	if err != nil || missing.Available || missing.Content != "" {
		t.Fatal("missing content reported available", missing, err)
	}
	for _, invalid := range []proxy.RequestContentEvent{
		{RequestID: "short", Content: "x", OriginalBytes: 1},
		{RequestID: strings.Repeat("c", 32), Content: "", OriginalBytes: 1},
		{RequestID: strings.Repeat("d", 32), Content: "x", OriginalBytes: 0},
		{RequestID: strings.Repeat("e", 32), Content: "x", OriginalBytes: proxy.MaxPassthroughBytes + 1},
	} {
		if err := s.RecordRequestContent(ctx, invalid); err == nil {
			t.Fatal("invalid content accepted")
		}
	}
	removed, err := s.ClearRequestContents(ctx)
	if err != nil || removed != 1 {
		t.Fatalf("clear removed %d: %v", removed, err)
	}
}

func TestContentBudgetEvictsOldest(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	previous := contentStoreBudget
	contentStoreBudget = 120
	t.Cleanup(func() { contentStoreBudget = previous })
	var ids []string
	for id := 1; id <= 4; id++ {
		request := event(id, "a@example.invalid", "a")
		record(t, s, request)
		ids = append(ids, request.ID)
		content := strings.Repeat("a", 4000)
		if err := s.RecordRequestContent(ctx, proxy.RequestContentEvent{RequestID: request.ID, Content: content, OriginalBytes: len(content)}); err != nil {
			t.Fatal(err)
		}
	}
	var total int64
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(stored_bytes),0),COUNT(*) FROM request_contents`).Scan(&total, &count); err != nil {
		t.Fatal(err)
	}
	if total > contentStoreBudget || count >= 4 {
		t.Fatalf("budget not enforced: total=%d count=%d", total, count)
	}
	if _, err := s.RequestContent(ctx, ids[0]); err != nil {
		t.Fatal(err)
	}
	if newest, err := s.RequestContent(ctx, ids[3]); err != nil || !newest.Available {
		t.Fatalf("newest content evicted: %+v %v", newest, err)
	}
}

func TestLegacyTextContentStaysReadable(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	e := event(1, "a@example.invalid", "a")
	record(t, s, e)
	content := `{"messages":[{"role":"user","content":"迁移前的历史正文"}]}`
	if _, err := s.db.ExecContext(ctx, `INSERT INTO request_contents(request_id,data,format,preview,original_bytes,stored_bytes,truncated,captured_at) VALUES(?,?,'text','',?,?,1,?)`,
		e.ID, []byte(content), len(content), len(content), timeText(time.Now().UTC())); err != nil {
		t.Fatal(err)
	}
	got, err := s.RequestContent(ctx, e.ID)
	if err != nil || !got.Available || got.Content != content || !got.Truncated {
		t.Fatalf("legacy content lost: %+v %v", got, err)
	}
	page, err := s.RecordedRequests(ctx, RecordedQuery{Limit: 20})
	if err != nil || len(page.Items) != 1 || !strings.Contains(page.Items[0].Preview, "迁移前") || !page.Items[0].Truncated {
		t.Fatalf("legacy preview missing: %+v %v", page, err)
	}
}

func TestRetentionEvictionRemovesContent(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	if _, err := s.SetRetention(ctx, MinRetention); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= MinRetention; i++ {
		record(t, s, event(i, "a@example.invalid", "a"))
	}
	oldest := event(1, "a@example.invalid", "a")
	if err := s.RecordRequestContent(ctx, proxy.RequestContentEvent{RequestID: oldest.ID, Content: `{"old":true}`, OriginalBytes: 13}); err != nil {
		t.Fatal(err)
	}
	record(t, s, event(MinRetention+1, "a@example.invalid", "a"))
	if got, err := s.RequestContent(ctx, oldest.ID); err != nil || got.Available {
		t.Fatal("evicted request kept its content", got, err)
	}
	var remaining int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM request_contents`).Scan(&remaining); err != nil || remaining != 0 {
		t.Fatalf("orphan content remained: %d %v", remaining, err)
	}
}

func TestExportRequestsStreamsFullContent(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	first := event(1, "a@example.invalid", "a")
	first.Model, first.Usage = "gpt-one", proxy.UsageSnapshot{Input: 10, Output: 5, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, first)
	second := event(2, "b@example.invalid", "b")
	second.Model, second.Usage = "gpt-two", proxy.UsageSnapshot{Input: 3, Output: 2, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, second)
	content := `{"messages":[{"role":"user","content":"` + strings.Repeat("x", 300<<10) + `"}]}`
	if err := s.RecordRequestContent(ctx, proxy.RequestContentEvent{RequestID: first.ID, Content: content, OriginalBytes: len(content)}); err != nil {
		t.Fatal(err)
	}
	var rows []ExportRequest
	count, limited, err := s.ExportRequests(ctx, RequestQuery{}, func(row ExportRequest) error {
		rows = append(rows, row)
		return nil
	})
	if err != nil || limited || count != 2 || len(rows) != 2 {
		t.Fatalf("export rows=%d limited=%v err=%v", count, limited, err)
	}
	if rows[0].Item.Model != "gpt-one" || rows[0].Item.InputTokens != "10" || !rows[0].Content.Available || rows[0].Content.Truncated || rows[0].Content.Content != content {
		t.Fatalf("first export row mismatch: %+v", rows[0].Content)
	}
	if rows[1].Content.Available || rows[1].Item.Model != "gpt-two" {
		t.Fatalf("second export row mismatch: %+v", rows[1])
	}
	filtered := 0
	if count, _, err = s.ExportRequests(ctx, RequestQuery{Search: "gpt-two"}, func(ExportRequest) error {
		filtered++
		return nil
	}); err != nil || count != 1 || filtered != 1 {
		t.Fatalf("search filter ignored: count=%d filtered=%d err=%v", count, filtered, err)
	}
	if _, _, err = s.ExportRequests(ctx, RequestQuery{Search: "\x00"}, func(ExportRequest) error { return nil }); err == nil {
		t.Fatal("control character search accepted")
	}
}
