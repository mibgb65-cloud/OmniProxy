package antigravity

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

// 上游 Gemini 的 promptTokenCount 已包含缓存，cachedContentTokenCount 只是其中的子集；
// 转换层必须把它写成 Chat 的 prompt_tokens_details.cached_tokens，否则请求页只能显示「—」。
func TestChatUsageReportsCachedPromptTokens(t *testing.T) {
	body, err := forwardChatResponse(t, chatUsageResponse(`{"promptTokenCount":100,"candidatesTokenCount":20,"thoughtsTokenCount":5,"cachedContentTokenCount":80,"totalTokenCount":125}`))
	if err != nil {
		t.Fatal(err)
	}
	var result object
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	usage := result["usage"].(map[string]any)
	details, ok := usage["prompt_tokens_details"].(map[string]any)
	if !ok || details["cached_tokens"] != float64(80) {
		t.Fatalf("cached token detail missing: %v", usage)
	}
	parsed := proxy.ParseUsage(body, "chat", false)
	if parsed.Input != 100 || parsed.CachedInput == nil || *parsed.CachedInput != 80 {
		t.Fatalf("cached usage not observable downstream: %+v", parsed)
	}
}

// 上游没给缓存字段时保持未知，不能补零。
func TestChatUsageLeavesCacheUnknownWhenUpstreamOmitsIt(t *testing.T) {
	body, err := forwardChatResponse(t, chatUsageResponse(`{"promptTokenCount":100,"candidatesTokenCount":20,"totalTokenCount":120}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "prompt_tokens_details") {
		t.Fatal("missing upstream cache detail was invented")
	}
	if parsed := proxy.ParseUsage(body, "chat", false); parsed.CachedInput != nil {
		t.Fatalf("cache should stay unknown: %+v", parsed)
	}
}

// 上游明确报告的零命中也算已知值。
func TestChatUsageKeepsReportedZeroCache(t *testing.T) {
	body, err := forwardChatResponse(t, chatUsageResponse(`{"promptTokenCount":100,"candidatesTokenCount":20,"cachedContentTokenCount":0,"totalTokenCount":120}`))
	if err != nil {
		t.Fatal(err)
	}
	parsed := proxy.ParseUsage(body, "chat", false)
	if parsed.CachedInput == nil || *parsed.CachedInput != 0 {
		t.Fatalf("reported zero cache must stay known: %+v", parsed)
	}
}

// 缓存量不可能超过提示词总数；自相矛盾时按未知处理，不写入错误比例。
func TestChatUsageDropsCacheAbovePrompt(t *testing.T) {
	body, err := forwardChatResponse(t, chatUsageResponse(`{"promptTokenCount":100,"candidatesTokenCount":20,"cachedContentTokenCount":120,"totalTokenCount":120}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "prompt_tokens_details") {
		t.Fatal("inconsistent cache count was published")
	}
	if parsed := proxy.ParseUsage(body, "chat", false); parsed.CachedInput != nil {
		t.Fatalf("inconsistent cache should stay unknown: %+v", parsed)
	}
}

// 负数与其它异常用量一样按非法上游响应拒绝。
func TestChatUsageRejectsNegativeCache(t *testing.T) {
	_, err := forwardChatResponse(t, chatUsageResponse(`{"promptTokenCount":100,"candidatesTokenCount":20,"cachedContentTokenCount":-1,"totalTokenCount":120}`))
	var failure *proxy.Error
	if !errors.As(err, &failure) || failure.Code != "invalid_upstream_response" || failure.StatusCode != 502 {
		t.Fatalf("negative cache count accepted: %v", err)
	}
}

// 流式逐帧累计：终帧省略缓存字段时保留已报告的命中量，不抹成未知。
func TestStreamKeepsReportedCacheWhenFinalFrameOmitsIt(t *testing.T) {
	const head = `{"response":{"candidates":[{"content":{"parts":[{"text":"hel"}]}}],"usageMetadata":{"promptTokenCount":100,"candidatesTokenCount":0,"cachedContentTokenCount":80,"totalTokenCount":100}}}`
	const tail = `{"response":{"candidates":[{"content":{"parts":[{"text":"lo"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":100,"candidatesTokenCount":20,"thoughtsTokenCount":5,"totalTokenCount":125}}}`
	read, write := io.Pipe()
	client := New(transportFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: read}, nil
	}))
	defer read.Close()
	go func() {
		defer write.Close()
		io.WriteString(write, "data: "+head+"\n\n")
		io.WriteString(write, "data: "+tail+"\n\n")
	}()
	var streamed []byte
	var readErr error
	err := client.Forward(context.Background(), testCredentials(), proxy.Request{Model: "gemini-test", Protocol: "chat", Stream: true, Body: []byte(`{"messages":[{"role":"user","content":"hi"}]}`)}, func(response proxy.Response) error {
		streamed, readErr = io.ReadAll(response.Body)
		return readErr
	})
	if err != nil {
		t.Fatal(err)
	}
	terminal := lastUsageFrame(t, string(streamed))
	parsed := proxy.ParseUsage(terminal, "chat", true)
	if parsed.Input != 100 || parsed.CachedInput == nil || *parsed.CachedInput != 80 {
		t.Fatalf("stream terminal usage lost the cache count: %+v", parsed)
	}
}

func chatUsageResponse(usage string) string {
	return `{"response":{"candidates":[{"content":{"parts":[{"text":"hi"}]},"finishReason":"STOP"}],"usageMetadata":` + usage + `}}`
}

func forwardChatResponse(t *testing.T, upstream string) ([]byte, error) {
	t.Helper()
	client := New(transportFunc(func(*http.Request) (*http.Response, error) { return reply(200, upstream), nil }))
	var body []byte
	err := client.Forward(context.Background(), testCredentials(), proxy.Request{Model: "gemini-test", Protocol: "chat", Body: []byte(`{"messages":[{"role":"user","content":"hi"}]}`)}, func(response proxy.Response) error {
		var readErr error
		body, readErr = io.ReadAll(response.Body)
		return readErr
	})
	return body, err
}

// lastUsageFrame 取流里最后一个携带 usage 的 data 帧。
func lastUsageFrame(t *testing.T, stream string) []byte {
	t.Helper()
	var terminal []byte
	for _, frame := range strings.Split(stream, "\n\n") {
		if !strings.Contains(frame, `"usage"`) {
			continue
		}
		terminal = []byte(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(frame), "data:")))
	}
	if len(terminal) == 0 {
		t.Fatal("stream carried no usage frame")
	}
	return terminal
}
