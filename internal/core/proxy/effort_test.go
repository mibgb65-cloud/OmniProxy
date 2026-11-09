package proxy

import (
	"strings"
	"testing"
)

func TestReasoningEffortReadsEachProtocol(t *testing.T) {
	cases := map[string]struct {
		protocol, body, want string
	}{
		"messages":  {"messages", `{"output_config":{"effort":"high","format":"kept"}}`, "high"},
		"responses": {"responses", `{"reasoning":{"effort":"low","summary":"auto"}}`, "low"},
		"chat":      {"chat", `{"reasoning_effort":"medium"}`, "medium"},
		"absent":    {"messages", `{"messages":[]}`, ""},
		"null":      {"responses", `{"reasoning":null}`, ""},
		"invalid":   {"messages", `{"output_config":[]}`, ""},
		"number":    {"chat", `{"reasoning_effort":3}`, ""},
	}
	for name, tc := range cases {
		if got := ReasoningEffort(tc.protocol, []byte(tc.body)); got != tc.want {
			t.Fatalf("%s = %q, want %q", name, got, tc.want)
		}
	}
}

// 档位来自客户端正文，只能接受可入库的短取值；异常值按未指定处理。
func TestNormalizeEffortRejectsUnsafeValues(t *testing.T) {
	if got := NormalizeEffort(" high "); got != "high" {
		t.Fatalf("trim lost: %q", got)
	}
	for _, value := range []string{"", "hi gh", "high\x00", strings.Repeat("x", 33), "档位"} {
		if got := NormalizeEffort(value); got != "" {
			t.Fatalf("%q accepted as %q", value, got)
		}
	}
}

// 协议解析同时给出模型和档位，透明转发与 HTTP 入口共用同一份判定。
func TestParseProtocolRequestCapturesEffort(t *testing.T) {
	responses, err := ParseProtocolRequest([]byte(`{"model":"gpt-test","reasoning":{"effort":"xhigh"},"input":[]}`), "/v1/responses")
	if err != nil || responses.Effort != "xhigh" {
		t.Fatalf("responses effort = %q, %v", responses.Effort, err)
	}
	messages, err := ParseProtocolRequest([]byte(`{"model":"claude-test","max_tokens":8,"output_config":{"effort":"medium"},"messages":[{"role":"user","content":"hi"}]}`), "/v1/messages")
	if err != nil || messages.Effort != "medium" {
		t.Fatalf("messages effort = %q, %v", messages.Effort, err)
	}
	chat, err := ParseProtocolRequest([]byte(`{"model":"deepseek-flash","reasoning_effort":"low","messages":[{"role":"user","content":"hi"}]}`), "/v1/chat/completions")
	if err != nil || chat.Effort != "low" {
		t.Fatalf("chat effort = %q, %v", chat.Effort, err)
	}
	absent, err := ParseProtocolRequest([]byte(`{"model":"gpt-test","input":[]}`), "/v1/responses")
	if err != nil || absent.Effort != "" {
		t.Fatalf("absent effort = %q, %v", absent.Effort, err)
	}
}
