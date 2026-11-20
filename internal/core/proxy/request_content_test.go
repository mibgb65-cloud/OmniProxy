package proxy

import (
	"context"
	"strings"
	"testing"
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

func TestPrepareRequestContentFoldsInlineDataAndKeepsFullBody(t *testing.T) {
	small := []byte(`{"messages":[{"role":"user","content":"hello"}]}`)
	content, original := prepareRequestContent(small)
	if content != string(small) || original != len(small) {
		t.Fatalf("small content changed: %q %d", content, original)
	}
	// 长 Base64 内联图片只保留占位文本，避免正文被一张图占满。
	image := `{"input":[{"type":"image_url","url":"data:image/png;base64,` + strings.Repeat("A", 4096) + `"}]}`
	content, _ = prepareRequestContent([]byte(image))
	if strings.Contains(content, strings.Repeat("A", 4096)) || !strings.Contains(content, "<omitted>") || !strings.Contains(content, "data:application/octet-stream") {
		t.Fatalf("inline data not folded: %q", content)
	}
	// 完整保存：超过旧 256 KiB 上限的正文不再截断。
	huge := []byte(`{"x":"` + strings.Repeat("中", 300<<10) + `"}`)
	content, original = prepareRequestContent(huge)
	if len(content) != len(huge) || original != len(huge) || !strings.HasSuffix(content, `"}`) {
		t.Fatalf("large content was altered: len=%d original=%d", len(content), original)
	}
	// 压缩或二进制正文不保存：返回空内容而不是把乱码写进数据库。
	content, original = prepareRequestContent([]byte{0x1f, 0x8b, 0x08, 0x00, 0xff})
	if content != "" || original != 5 {
		t.Fatalf("binary content accepted: %q %d", content, original)
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
