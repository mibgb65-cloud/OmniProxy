package river

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

// userItems 构造 n 条合法用户消息输入项，用于验证条数上限。
func userItems(n int) json.RawMessage {
	var builder strings.Builder
	builder.WriteString("[")
	for i := 0; i < n; i++ {
		if i > 0 {
			builder.WriteString(",")
		}
		fmt.Fprintf(&builder, `{"role":"user","content":"message %d"}`, i)
	}
	builder.WriteString("]")
	return json.RawMessage(builder.String())
}

// Codex 长会话的历史输入项很容易超过 512 条：Responses 校验必须与 Messages 同口径
// 使用 100,000 上限，否则会被本地误拒成 unsupported_content（请求 1479 ms 内失败）。
func TestResponsesInputItemsUseConversationLimit(t *testing.T) {
	if !textMessages(userItems(513)) {
		t.Fatal("513 input items rejected")
	}
	if !textMessages(userItems(proxy.MaxConversationMessages)) {
		t.Fatal("input items at the conversation limit rejected")
	}
	if textMessages(userItems(proxy.MaxConversationMessages + 1)) {
		t.Fatal("input items above the conversation limit accepted")
	}
}

// Codex 用 apply_patch 时会在历史里带 custom_tool_call／custom_tool_call_output；
// 真实 River Key 实测接受这些条目，本地不能拒成 unsupported_content。
func TestCustomToolHistoryIsAccepted(t *testing.T) {
	body := []byte(`{"model":"` + DeepSeekFlash + `","stream":true,"input":[` +
		`{"role":"user","content":"patch it"},` +
		`{"type":"custom_tool_call","call_id":"call_1","name":"apply_patch","input":"*** Begin Patch"},` +
		`{"type":"custom_tool_call_output","call_id":"call_1","output":"Exit code: 0"}]}`)
	request := proxy.Request{Protocol: "responses", Model: DeepSeekFlash, Stream: true, Body: body}
	if _, err := validatedBody(request, "/responses"); err != nil {
		t.Fatalf("custom tool history rejected: %v", err)
	}
	// 缺 call_id 的自定义工具结果无法关联，仍需明确拒绝。
	orphan := []byte(`{"model":"` + DeepSeekFlash + `","stream":true,"input":[` +
		`{"type":"custom_tool_call_output","output":"Exit code: 0"}]}`)
	if _, err := validatedBody(proxy.Request{Protocol: "responses", Model: DeepSeekFlash, Stream: true, Body: orphan}, "/responses"); err == nil {
		t.Fatal("orphan custom tool output accepted")
	}
}
