package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPassthroughDeliveredStreamCountsAsSuccess 断言上游在交付终结事件后异常断开时，
// 请求不再被记成失败：客户端已收到完整回答，用量也按未声明 Content-Type 的 SSE 解析出来。
func TestPassthroughDeliveredStreamCountsAsSuccess(t *testing.T) {
	body := "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"resp_1\",\"usage\":{\"input_tokens\":12,\"output_tokens\":4}}}\n\n"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 故意不写 Content-Type，复现本机 gpt-6.1-sol 流式响应的实际情况。
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, body)
		w.(http.Flusher).Flush()
		// 交付后立即断开，不再发送 chunked 终止块。
		panic(http.ErrAbortHandler)
	}))
	defer upstream.Close()

	finished := make(chan RequestEvent, 2)
	session := &stubSession{attempts: []PassthroughAttempt{{
		BaseURL: upstream.URL + "/backend-api/codex",
		Header:  http.Header{"Authorization": {"Bearer pool-token"}},
	}}}
	server := New(Options{Port: 0, Passthrough: stubBinding{session}, Upstream: unusedUpstream(t), Observer: firstTokenObserver{finished}})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = server.Stop(context.Background()) }()

	resp, err := http.Post(server.Status().Address+"/codex/v1/responses", "application/json", strings.NewReader(`{"model":"gpt-6.1-sol"}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	received, readErr := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(received), "response.completed") {
		t.Fatalf("status = %d body = %q err = %v", resp.StatusCode, received, readErr)
	}
	event := <-finished
	if event.State != "success" || event.ErrorCode != "" {
		t.Fatalf("已交付的回答被记成失败: %+v", event)
	}
	if event.Usage.Input != 12 || event.Usage.Output != 4 || !event.Usage.InputKnown {
		t.Fatalf("未声明内容类型的 SSE 未统计用量: %+v", event.Usage)
	}
}

// TestPassthroughOutcomeClassification 断言结束分类只看可证实信号。
func TestPassthroughOutcomeClassification(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	cancelledRequest, _ := http.NewRequestWithContext(cancelled, http.MethodPost, "/codex/v1/responses", nil)
	liveRequest, _ := http.NewRequest(http.MethodPost, "/codex/v1/responses", nil)
	terminal := &Trace{}
	terminal.markTerminal()

	for _, test := range []struct {
		name     string
		trace    *Trace
		request  *http.Request
		result   passthroughCopy
		wantCode string
	}{
		{"终结事件已交付", terminal, liveRequest, passthroughCopy{written: 128, side: "upstream", err: io.ErrUnexpectedEOF}, ""},
		{"上游读流中断", &Trace{}, liveRequest, passthroughCopy{written: 128, side: "upstream", err: io.ErrUnexpectedEOF}, "upstream_stream_failed"},
		{"客户端断开", &Trace{}, liveRequest, passthroughCopy{written: 128, side: "client", err: io.ErrClosedPipe}, "request_cancelled"},
		{"请求上下文已取消", &Trace{}, cancelledRequest, passthroughCopy{written: 0, side: "upstream", err: io.ErrUnexpectedEOF}, "request_cancelled"},
	} {
		if got := passthroughOutcome(test.request, test.trace, test.result); got != test.wantCode {
			t.Fatalf("%s: code = %q, want %q", test.name, got, test.wantCode)
		}
	}
}
