package gateway

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// 上游只接受流式；正文必须被改成 stream=true 并索取用量，其余字段原样保留。
func TestChatStreamBodyForcesStreamAndKeepsFields(t *testing.T) {
	body := []byte(`{"model":"glm-5.3","stream":false,"messages":[{"role":"user","content":"hi"}],"stream_options":{"existing":true}}`)
	converted, err := chatStreamBody(body)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(converted, &fields); err != nil {
		t.Fatal(err)
	}
	if string(fields["stream"]) != "true" || string(fields["model"]) != `"glm-5.3"` {
		t.Fatalf("stream/model not kept: %s", converted)
	}
	var options map[string]bool
	if err := json.Unmarshal(fields["stream_options"], &options); err != nil || !options["include_usage"] || !options["existing"] {
		t.Fatalf("stream options not merged: %s", fields["stream_options"])
	}
	if _, err := chatStreamBody([]byte(`[]`)); err == nil {
		t.Fatal("non-object body accepted")
	}
}

// SSE 分片要聚合成一份完整快照：文本、工具调用、结束原因与用量都不能丢。
func TestMergeChatStreamAggregatesTextCallsAndUsage(t *testing.T) {
	stream := strings.Join([]string{
		`data: {"id":"chat-1","model":"glm-5.3","choices":[{"index":0,"delta":{"role":"assistant","content":"Hel"}}]}`,
		``,
		`data: {"id":"chat-1","model":"glm-5.3","choices":[{"index":0,"delta":{"content":"lo"}}]}`,
		``,
		`data: {"id":"chat-1","model":"glm-5.3","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"read_file","arguments":"{\"path\":"}}]}}]}`,
		``,
		`data: {"id":"chat-1","model":"glm-5.3","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"a.txt\"}"}}]},"finish_reason":"tool_calls"}]}`,
		``,
		`data: {"id":"chat-1","model":"glm-5.3","choices":[],"usage":{"prompt_tokens":12,"completion_tokens":7}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")
	merged, err := mergeChatStream(strings.NewReader(stream))
	if err != nil {
		t.Fatal(err)
	}
	var raw struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Finish  string `json:"finish_reason"`
			Message struct {
				Content string `json:"content"`
				Calls   []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			Prompt     int `json:"prompt_tokens"`
			Completion int `json:"completion_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(merged, &raw); err != nil {
		t.Fatalf("merged body is not JSON: %s", merged)
	}
	if raw.ID != "chat-1" || raw.Model != "glm-5.3" || raw.Usage.Prompt != 12 || raw.Usage.Completion != 7 {
		t.Fatalf("ids or usage lost: %s", merged)
	}
	if len(raw.Choices) != 1 || raw.Choices[0].Message.Content != "Hello" || raw.Choices[0].Finish != "tool_calls" {
		t.Fatalf("text or finish lost: %s", merged)
	}
	if len(raw.Choices[0].Message.Calls) != 1 || raw.Choices[0].Message.Calls[0].Function.Name != "read_file" || raw.Choices[0].Message.Calls[0].Function.Arguments != `{"path":"a.txt"}` {
		t.Fatalf("tool call not aggregated: %s", merged)
	}
}

// 已经是整份 JSON 的回包原样保留；空流与错误事件都不能当成成功。
func TestMergeChatStreamHandlesJSONAndFailures(t *testing.T) {
	whole := []byte(`{"id":"chat-2","choices":[{"message":{"content":"ok"},"finish_reason":"stop"}]}`)
	merged, err := mergeChatStream(bytes.NewReader(whole))
	if err != nil || !bytes.Equal(merged, whole) {
		t.Fatalf("whole JSON not kept: %s %v", merged, err)
	}
	if _, err := mergeChatStream(strings.NewReader("")); err == nil {
		t.Fatal("empty stream accepted")
	}
	if _, err := mergeChatStream(strings.NewReader("data: {\"error\":{\"message\":\"bad\"}}\n\n")); err == nil {
		t.Fatal("stream error accepted")
	}
	if _, err := mergeChatStream(strings.NewReader("data: {\"choices\":[]}\n\ndata: [DONE]\n\n")); err == nil {
		t.Fatal("empty reply accepted")
	}
}
