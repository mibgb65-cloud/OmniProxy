package agentprotocol

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
)

func TestMessagesToolRoundTripAndStream(t *testing.T) {
	source := []byte(`{"model":"test","system":[{"type":"text","text":"help"}],"max_tokens":200,"tools":[{"name":"read_file","input_schema":{"type":"object"}}],"messages":[{"role":"assistant","content":[{"type":"tool_use","id":"call_1","name":"read_file","input":{"path":"a"}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"contents"}]}]}`)
	body, kinds, err := ToChat(source, "messages")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), `"tool_call_id":"call_1"`) || kinds["read_file"] != "function" {
		t.Fatal("tool association lost")
	}
	result, err := ReadChat(strings.NewReader(`{"id":"c1","model":"test","choices":[{"finish_reason":"tool_calls","message":{"content":"","tool_calls":[{"id":"call_2","function":{"name":"read_file","arguments":"{\"path\":\"b\"}"}}]}}],"usage":{"prompt_tokens":5,"completion_tokens":8}}`), kinds)
	if err != nil {
		t.Fatal(err)
	}
	stream, _ := Encode(result, "messages", true)
	bytes, _ := io.ReadAll(stream)
	for _, event := range []string{"message_start", "content_block_start", "input_json_delta", "content_block_stop", "message_delta", "message_stop", "call_2", "tool_use"} {
		if !strings.Contains(string(bytes), event) {
			t.Fatal("missing event", event)
		}
	}
}
func TestResponsesCustomToolsAndCompletion(t *testing.T) {
	source := []byte(`{"model":"test","input":[{"type":"custom_tool_call","name":"apply_patch","call_id":"c","input":"patch"},{"type":"custom_tool_call_output","call_id":"c","output":"ok"}],"tools":[{"type":"custom","name":"apply_patch"}]}`)
	body, kinds, err := ToChat(source, "responses")
	if err != nil || !json.Valid(body) || kinds["apply_patch"] != "custom" {
		t.Fatal("custom tool lost", err)
	}
	r := Result{ID: "test", Model: "test", Text: "OK", Finish: "stop", Calls: []Call{{"c2", "apply_patch", `{"input":"patch2"}`, "custom"}}}
	reader, _ := Encode(r, "responses", true)
	b, _ := io.ReadAll(reader)
	if !strings.Contains(string(b), "response.custom_tool_call_input.done") || !strings.Contains(string(b), "response.completed") {
		t.Fatal("missing terminal tool events")
	}
	if _, err := ReadResponses(strings.NewReader(`data: {"type":"response.output_text.delta","delta":"partial"}` + "\n\n")); err == nil {
		t.Fatal("truncated response became successful")
	}
	if _, _, err := ToChat([]byte(`{"model":"test","input":[{"role":"user","content":[{"type":"input_image","image_url":"https://invalid"}]}]}`), "responses"); err == nil {
		t.Fatal("unsupported multimodal input silently lost")
	}
}
