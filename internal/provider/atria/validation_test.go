package atria

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

func TestOutputLimitsAndNonTextInput(t *testing.T) {
	for _, name := range []string{"max_tokens", "max_completion_tokens", "max_output_tokens"} {
		for _, limit := range []string{"0", "65537", "1.5", `"16"`, "null", "true"} {
			request := testRequest("responses", false)
			var fields map[string]json.RawMessage
			_ = json.Unmarshal(request.Body, &fields)
			fields[name] = json.RawMessage(limit)
			request.Body, _ = json.Marshal(fields)
			if _, err := validatedBody(request, "/responses"); err == nil {
				t.Errorf("accepted %s=%s", name, limit)
			}
		}
	}
	for _, content := range []string{
		`[{"role":"user","content":[{"type":"input_image","image_url":"https://example.test/image"}]}]`,
		`[{"role":"user","content":[{"type":"input_file","file_id":"file-private"}]}]`,
		`[{"type":"function_call_output","call_id":"call-one","output":[{"type":"input_image"}]}]`,
		`null`,
	} {
		request := testRequest("responses", false)
		request.Body = []byte(`{"model":"Atria-Dawn-Preview","input":` + content + `}`)
		if _, err := validatedBody(request, "/responses"); err == nil {
			t.Errorf("non-text input accepted: %s", content)
		}
	}
	request := testRequest("messages", false)
	request.Body = []byte(`{"model":"Atria-Dawn-Preview","max_tokens":16,"messages":[{"role":"user","content":[{"type":"tool_result","tool_use_id":"one","content":[{"type":"image"}]}]}]}`)
	if _, err := validatedBody(request, "/messages"); err == nil {
		t.Fatal("image in tool result accepted")
	}
}

func TestNativeToolHistoryRemainsUnchanged(t *testing.T) {
	for protocol, input := range map[string]string{
		"responses": `"input":[{"type":"function_call","call_id":"one","name":"read","arguments":"{}"},{"type":"function_call_output","call_id":"one","output":"text"}],"tools":[{"type":"function","name":"read","parameters":{"type":"object"}}],"max_output_tokens":65536`,
		"messages":  `"max_tokens":1,"messages":[{"role":"assistant","content":[{"type":"tool_use","id":"one","name":"read","input":{"type":"image","content":"business parameter"}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"one","content":[{"type":"text","text":"result"}]}]}]`,
	} {
		request := proxy.Request{Model: ModelID, Protocol: protocol, Body: []byte(`{"model":"Atria-Dawn-Preview",` + input + `}`)}
		path, _ := protocolPath(protocol)
		body, err := validatedBody(request, path)
		if err != nil || string(body) != string(request.Body) {
			t.Fatalf("%s tool history changed: %v", protocol, err)
		}
	}
}

// Codex 长会话的 Responses 输入项会超过 512 条：与 Messages 同口径使用 100,000 上限。
func TestResponsesInputItemsUseConversationLimit(t *testing.T) {
	build := func(n int) json.RawMessage {
		var b strings.Builder
		b.WriteString("[")
		for i := 0; i < n; i++ {
			if i > 0 {
				b.WriteString(",")
			}
			fmt.Fprintf(&b, `{"role":"user","content":"message %d"}`, i)
		}
		b.WriteString("]")
		return json.RawMessage(b.String())
	}
	if !textMessages(build(513), true) {
		t.Fatal("513 input items rejected")
	}
	if !textMessages(build(proxy.MaxConversationMessages), true) {
		t.Fatal("input items at the conversation limit rejected")
	}
	if textMessages(build(proxy.MaxConversationMessages+1), true) {
		t.Fatal("input items above the conversation limit accepted")
	}
}
