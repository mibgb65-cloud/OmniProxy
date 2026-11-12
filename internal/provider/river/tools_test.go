package river

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

// functionTools 构造 count 个结构合法、名字唯一的工具；Chat 用 function.name，
// Responses 用顶层 name，与各自协议的原生形态一致。
func functionTools(count int, protocol string) []any {
	tools := make([]any, 0, count)
	for i := 0; i < count; i++ {
		name := fmt.Sprintf("tool_%d", i)
		parameters := map[string]any{"type": "object", "properties": map[string]any{"q": map[string]any{"type": "string"}}}
		if protocol == "chat" {
			tools = append(tools, map[string]any{
				"type":     "function",
				"function": map[string]any{"name": name, "parameters": parameters},
			})
			continue
		}
		tools = append(tools, map[string]any{"type": "function", "name": name, "parameters": parameters})
	}
	return tools
}

func encodedTools(t *testing.T, tools any) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(tools)
	if err != nil {
		t.Fatalf("marshal tools: %v", err)
	}
	return encoded
}

func failureCode(err error) string {
	var failure *proxy.Error
	if errors.As(err, &failure) {
		return failure.Code
	}
	if err == nil {
		return ""
	}
	return err.Error()
}

// TestToolCountIsNotCapped 断言 River 适配器不再按 128 个工具拒绝请求。
//
// 官方 Limits 未公布工具数量上限，2026-11-12 实测 176 个函数工具仍被上游接受；
// 旧实现在 129 个就本地返回 400 invalid_request，DeepSeek Harness 因带 176 个
// （内置 + MCP）工具而全部失败。
func TestToolCountIsNotCapped(t *testing.T) {
	for _, count := range []int{1, 128, 129, 176, 200, 512} {
		for _, protocol := range []string{"chat", "responses"} {
			t.Run(fmt.Sprintf("%s/%d", protocol, count), func(t *testing.T) {
				if err := validateTools(encodedTools(t, functionTools(count, protocol)), protocol); err != nil {
					t.Fatalf("%d tools rejected: %v", count, err)
				}
			})
		}
	}
}

// TestChatForwardAcceptsManyTools 走完整适配器路径，确认 176 个工具不会被本地拦下，
// 而是原样到达上游（这里用契约测试的假上游接收）。
func TestChatForwardAcceptsManyTools(t *testing.T) {
	const tools = 176
	var received int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
			return
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(body, &fields); err != nil {
			t.Errorf("decode body: %v", err)
			return
		}
		var list []json.RawMessage
		if err := json.Unmarshal(fields["tools"], &list); err != nil {
			t.Errorf("decode tools: %v", err)
			return
		}
		received = len(list)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"x","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`))
	}))
	defer upstream.Close()

	requestBody, err := json.Marshal(map[string]any{
		"model":    DeepSeekFlash,
		"stream":   false,
		"messages": []any{map[string]any{"role": "user", "content": "hi"}},
		"tools":    functionTools(tools, "chat"),
	})
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	request := proxy.Request{Model: DeepSeekFlash, Protocol: "chat", Body: requestBody}
	if err := New(nil).Forward(context.Background(), upstream.URL, []byte("rv_test_key"), request, func(proxy.Response) error { return nil }); err != nil {
		t.Fatalf("forward rejected: %v", err)
	}
	if received != tools {
		t.Fatalf("upstream received %d tools, want %d", received, tools)
	}
}

func TestToolStructureStillValidated(t *testing.T) {
	one := func(tool map[string]any) json.RawMessage {
		return encodedTools(t, []any{tool})
	}
	cases := []struct {
		name     string
		protocol string
		raw      json.RawMessage
		code     string
	}{
		{
			name:     "chat tool without function",
			protocol: "chat",
			raw:      one(map[string]any{"type": "function"}),
			code:     "invalid_request",
		},
		{
			name:     "chat empty function name",
			protocol: "chat",
			raw:      one(map[string]any{"type": "function", "function": map[string]any{"name": ""}}),
			code:     "invalid_request",
		},
		{
			name:     "chat long function name",
			protocol: "chat",
			raw:      one(map[string]any{"type": "function", "function": map[string]any{"name": strings.Repeat("a", 129)}}),
			code:     "invalid_request",
		},
		{
			name:     "responses empty name",
			protocol: "responses",
			raw:      one(map[string]any{"type": "function", "name": ""}),
			code:     "invalid_request",
		},
		{
			name:     "non-function tool",
			protocol: "chat",
			raw:      one(map[string]any{"type": "custom", "custom": map[string]any{"name": "x"}}),
			code:     "unsupported_content",
		},
		{
			name:     "null tools",
			protocol: "chat",
			raw:      json.RawMessage("null"),
			code:     "invalid_request",
		},
		{
			name:     "malformed tools",
			protocol: "chat",
			raw:      json.RawMessage(`{"type":"function"}`),
			code:     "invalid_request",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTools(tc.raw, tc.protocol)
			if got := failureCode(err); got != tc.code {
				t.Fatalf("code = %q, want %q (err=%v)", got, tc.code, err)
			}
		})
	}
}

func TestAbsentToolsIsAccepted(t *testing.T) {
	if err := validateTools(nil, "chat"); err != nil {
		t.Fatalf("absent tools rejected: %v", err)
	}
}
