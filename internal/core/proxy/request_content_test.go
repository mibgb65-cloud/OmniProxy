package proxy

import (
	"context"
	"strings"
	"testing"
	"unicode/utf8"
)

type contentSpy struct {
	fail   bool
	events []RequestContentEvent
}

func (s *contentSpy) Begin(context.Context, RequestEvent) error  { return nil }
func (s *contentSpy) Update(context.Context, RequestEvent) error { return nil }
func (s *contentSpy) Finish(context.Context, RequestEvent) error { return nil }
func (s *contentSpy) RecordRequestContent(_ context.Context, event RequestContentEvent) error {
	if s.fail {
		return Failure("content_unavailable", 500)
	}
	s.events = append(s.events, event)
	return nil
}

func testTrace(spy *contentSpy) *Trace {
	return &Trace{event: RequestEvent{ID: strings.Repeat("a", 32)}, observer: spy}
}

func TestPrepareRequestContentFoldsInlineDataAndTruncates(t *testing.T) {
	small := []byte(`{"messages":[{"role":"user","content":"hello"}]}`)
	content, truncated, original := prepareRequestContent(small)
	if truncated || content != string(small) || original != len(small) {
		t.Fatalf("small content changed: %q %v %d", content, truncated, original)
	}
	// 长 Base64 内联图片只保留占位文本，避免正文被一张图占满。
	image := `{"input":[{"type":"image_url","url":"data:image/png;base64,` + strings.Repeat("A", 4096) + `"}]}`
	content, truncated, _ = prepareRequestContent([]byte(image))
	if truncated || strings.Contains(content, strings.Repeat("A", 4096)) || !strings.Contains(content, "<omitted>") || !strings.Contains(content, "data:application/octet-stream") {
		t.Fatalf("inline data not folded: %q", content)
	}
	// 超过保存上限时按 UTF-8 边界截断并标记。
	huge := []byte(`{"x":"` + strings.Repeat("中", MaxRequestContentBytes) + `"}`)
	content, truncated, original = prepareRequestContent(huge)
	if !truncated || len(content) > MaxRequestContentBytes || original != len(huge) || !utf8.ValidString(content) {
		t.Fatalf("huge content not bounded: len=%d truncated=%v original=%d valid=%v", len(content), truncated, original, utf8.ValidString(content))
	}
	// 压缩或二进制正文不保存：返回空内容而不是把乱码写进数据库。
	content, truncated, original = prepareRequestContent([]byte{0x1f, 0x8b, 0x08, 0x00, 0xff})
	if content != "" || truncated || original != 5 {
		t.Fatalf("binary content accepted: %q %v %d", content, truncated, original)
	}
}

func TestRecordContentWritesOnceAndRecordsFailure(t *testing.T) {
	spy := &contentSpy{}
	trace := testTrace(spy)
	trace.RecordContent(context.Background(), []byte(`{"model":"x"}`))
	trace.RecordContent(context.Background(), []byte(`{"model":"y"}`))
	if len(spy.events) != 1 || spy.events[0].Content != `{"model":"x"}` || spy.events[0].RequestID != trace.ID() {
		t.Fatalf("unexpected content events: %+v", spy.events)
	}
	failing := &contentSpy{fail: true}
	failedTrace := testTrace(failing)
	failedTrace.RecordContent(context.Background(), []byte(`{"model":"z"}`))
	if !failedTrace.contentFailed || failedTrace.diagnosticLogFields() == nil {
		t.Fatal("content failure was not recorded")
	}
}

func TestServerRecordRequestContentHonoursSetting(t *testing.T) {
	spy := &contentSpy{}
	disabled := New(Options{RecordRequestContent: func(context.Context) (bool, error) { return false, nil }})
	disabledTrace := &Trace{event: RequestEvent{ID: strings.Repeat("b", 32)}, observer: spy}
	disabled.recordRequestContent(context.Background(), disabledTrace, []byte(`{"a":1}`))
	if len(spy.events) != 0 {
		t.Fatal("disabled setting captured content")
	}
	enabled := New(Options{RecordRequestContent: func(context.Context) (bool, error) { return true, nil }})
	enabled.recordRequestContent(context.Background(), disabledTrace, []byte(`{"a":1}`))
	if len(spy.events) != 1 {
		t.Fatal("enabled setting did not capture content")
	}
	failing := New(Options{RecordRequestContent: func(context.Context) (bool, error) {
		return true, Failure("content_unavailable", 500)
	}})
	third := &Trace{event: RequestEvent{ID: strings.Repeat("c", 32)}, observer: spy}
	failing.recordRequestContent(context.Background(), third, []byte(`{"a":2}`))
	if len(spy.events) != 1 {
		t.Fatal("setting failure still captured content")
	}
}
