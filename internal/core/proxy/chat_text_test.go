package proxy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestChatNormalizesTextPartsPreservingToolsAndOptions(t *testing.T) {
	body := `{"model":"deepseek-flash","messages":[{"role":"user","content":[{"type":"text","text":"hello"},{"type":"text","text":"world"}],"name":"user"},{"role":"assistant","content":null,"reasoning_content":"thought","tool_calls":[{"id":"call-1"}]},{"role":"tool","tool_call_id":"call-1","content":[{"type":"text","text":"result"}]}],"thinking":{"type":"enabled"}}`
	request, err := ParseRequest([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	var config struct {
		Messages []map[string]any `json:"messages"`
		Thinking map[string]any   `json:"thinking"`
	}
	if json.Unmarshal(request.Body, &config) != nil || config.Messages[0]["content"] != "hello\nworld" || config.Messages[0]["name"] != "user" || config.Messages[1]["reasoning_content"] != "thought" || config.Messages[1]["tool_calls"] == nil || config.Messages[2]["content"] != "result" || config.Thinking["type"] != "enabled" {
		t.Fatal("text normalization lost tool or provider fields")
	}
}

func TestChatRejectsUnsupportedAndUnboundedParts(t *testing.T) {
	// image_url 已由适配器按上游契约处理，不再属于「不支持」的部分。
	for _, content := range []string{`[]`, `[{"type":"text"}]`, `[{"type":"text","text":null}]`, `[{"type":"text","text":4}]`, `[{"type":"audio_url","audio_url":{"url":"https://example.invalid"}}]`, `[` + strings.Repeat(`{"type":"text","text":"x"},`, 128) + `{"type":"text","text":"x"}]`, `[{"type":"text","text":"` + strings.Repeat("x", 1<<20+1) + `"}]`} {
		if _, err := ParseRequest([]byte(`{"model":"deepseek-flash","messages":[{"role":"user","content":` + content + `}]} `)); err == nil {
			t.Fatal("unsupported text parts accepted")
		}
	}
}
