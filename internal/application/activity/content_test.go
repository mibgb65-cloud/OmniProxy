package activity

import (
	"context"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

func TestContentCaptureSettingAndStorage(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	enabled, err := s.ContentCaptureEnabled(ctx)
	if err != nil || enabled {
		t.Fatal("content capture must default off", enabled, err)
	}
	retention, err := s.Retention(ctx)
	if err != nil || retention.ContentEnabled {
		t.Fatal("retention reported content enabled", retention, err)
	}
	e := event(1, "a@example.invalid", "a")
	record(t, s, e)
	content := strings.Repeat("x", 10)
	if err := s.RecordRequestContent(ctx, proxy.RequestContentEvent{RequestID: e.ID, Content: content, OriginalBytes: 12}); err != nil {
		t.Fatal(err)
	}
	// 重复写入保留首份，避免后到的快照改写正文。
	if err := s.RecordRequestContent(ctx, proxy.RequestContentEvent{RequestID: e.ID, Content: "other", OriginalBytes: 5}); err != nil {
		t.Fatal(err)
	}
	got, err := s.RequestContent(ctx, e.ID)
	if err != nil || !got.Available || got.Content != content || got.OriginalBytes != 12 || got.StoredBytes != 10 || got.Truncated {
		t.Fatalf("stored content mismatch: %+v %v", got, err)
	}
	missing, err := s.RequestContent(ctx, strings.Repeat("b", 32))
	if err != nil || missing.Available || missing.Content != "" {
		t.Fatal("missing content reported available", missing, err)
	}
	for _, invalid := range []proxy.RequestContentEvent{
		{RequestID: "short", Content: "x", OriginalBytes: 1},
		{RequestID: strings.Repeat("c", 32), Content: "", OriginalBytes: 1},
		{RequestID: strings.Repeat("d", 32), Content: "x", OriginalBytes: 0},
		{RequestID: strings.Repeat("e", 32), Content: strings.Repeat("x", proxy.MaxRequestContentBytes+1), OriginalBytes: proxy.MaxRequestContentBytes + 1},
	} {
		if err := s.RecordRequestContent(ctx, invalid); err == nil {
			t.Fatal("invalid content accepted")
		}
	}
	retention, err = s.SetContentCapture(ctx, true)
	if err != nil || !retention.ContentEnabled {
		t.Fatal("content capture not enabled", retention, err)
	}
	if enabled, err = s.ContentCaptureEnabled(ctx); err != nil || !enabled {
		t.Fatal("setting not persisted", enabled, err)
	}
	removed, err := s.ClearRequestContents(ctx)
	if err != nil || removed != 1 {
		t.Fatalf("clear removed %d: %v", removed, err)
	}
	if got, err = s.RequestContent(ctx, e.ID); err != nil || got.Available {
		t.Fatal("cleared content still available", got, err)
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
	if err := s.RecordRequestContent(ctx, proxy.RequestContentEvent{RequestID: oldest.ID, Content: `{"old":true}`, OriginalBytes: 13, Truncated: true}); err != nil {
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

func TestExportRequestsStreamsContent(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	first := event(1, "a@example.invalid", "a")
	first.Model, first.Usage = "gpt-one", proxy.UsageSnapshot{Input: 10, Output: 5, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, first)
	second := event(2, "b@example.invalid", "b")
	second.Model, second.Usage = "gpt-two", proxy.UsageSnapshot{Input: 3, Output: 2, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, second)
	if err := s.RecordRequestContent(ctx, proxy.RequestContentEvent{RequestID: first.ID, Content: `{"model":"gpt-one"}`, OriginalBytes: 1024, Truncated: true}); err != nil {
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
	if rows[0].Item.Model != "gpt-one" || rows[0].Item.InputTokens != "10" || !rows[0].Content.Available || !rows[0].Content.Truncated || rows[0].Content.OriginalBytes != 1024 {
		t.Fatalf("first export row mismatch: %+v", rows[0])
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
