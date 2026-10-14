package mimo

import (
	"encoding/json"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

func parseTestBody(t *testing.T, protocol, body string) map[string]json.RawMessage {
	t.Helper()
	request := proxy.Request{Model: "mimo-v2.6-pro", Protocol: protocol, Body: []byte(body)}
	output, err := validatedBody(request)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(output, &fields) != nil {
		t.Fatal("invalid output JSON")
	}
	return fields
}

func TestThinkingAndFunctionContextIsPreserved(t *testing.T) {
	cases := map[string]string{
		"chat":      `{"model":"mimo-v2.6-pro","thinking":{"type":"enabled"},"max_tokens":20,"messages":[{"role":"assistant","content":null,"reasoning_content":"retained thought","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{}"}}]},{"role":"tool","tool_call_id":"call_1","content":"file text"}],"tools":[{"type":"function","function":{"name":"read_file","parameters":{"type":"object"}}}]}`,
		"responses": `{"model":"mimo-v2.6-pro","input":[{"type":"reasoning","summary":[{"type":"summary_text","text":"retained thought"}]},{"type":"function_call","call_id":"call_1","name":"read_file","arguments":"{}"},{"type":"function_call_output","call_id":"call_1","output":"file text"}],"reasoning":{"effort":"high"},"tools":[{"type":"function","name":"read_file","parameters":{"type":"object"}}],"store":false,"background":false}`,
		"messages":  `{"model":"mimo-v2.6-pro","max_tokens":20,"thinking":{"type":"adaptive"},"system":[{"type":"text","text":"coding"}],"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"retained thought","signature":"signature"},{"type":"tool_use","id":"call_1","name":"read_file","input":{"type":"image","content":"business field"}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"file text"}]}],"tools":[{"name":"read_file","input_schema":{"type":"object"}}]}`,
	}
	for protocol, body := range cases {
		t.Run(protocol, func(t *testing.T) {
			fields := parseTestBody(t, protocol, body)
			field := "messages"
			if protocol == "responses" {
				field = "input"
				if _, exists := fields["store"]; exists {
					t.Error("unsupported storage flag forwarded")
				}
			}
			if !strings.Contains(string(fields[field]), "retained thought") || !strings.Contains(string(fields[field]), "call_1") || !strings.Contains(string(fields[field]), "file text") {
				t.Error("tool or thinking context lost")
			}
			if protocol == "messages" && string(fields["thinking"]) != `{"type":"enabled"}` {
				t.Error("adaptive thinking not normalized")
			}
			if protocol == "chat" && string(fields["max_completion_tokens"]) != "20" {
				t.Error("legacy token limit not adapted")
			}
		})
	}
}

func TestInvalidRequestsFailBeforeSendingCredentials(t *testing.T) {
	for _, body := range []string{
		`{"model":"unknown","input":"hello"}`,
		`{"model":"mimo-v2.6-pro","input":null}`,
		`{"model":"mimo-v2.6-pro","input":[]}`,
		`{"model":"mimo-v2.6-pro","input":"hello","max_output_tokens":131073}`,
		`{"model":"mimo-v2.6-pro","input":"hello","max_output_tokens":1.5}`,
		`{"model":"mimo-v2.6-pro","input":"hello","temperature":1.6}`,
		`{"model":"mimo-v2.6-pro","input":"hello","top_p":0}`,
		`{"model":"mimo-v2.6-pro","input":"hello","background":true}`,
		`{"model":"mimo-v2.6-pro","input":"hello","previous_response_id":"resp_old"}`,
		`{"model":"mimo-v2.6-pro","input":"hello","context_management":[]}`,
		`{"model":"mimo-v2.6-pro","input":"hello","tools":[{"type":"custom","name":"apply_patch"}]}`,
		`{"model":"mimo-v2.6-pro","input":[{"role":"user","content":[{"type":"input_image","image_url":"https://example.test/image"}]}]}`,
	} {
		if _, err := validatedBody(proxy.Request{Model: "mimo-v2.6-pro", Protocol: "responses", Body: []byte(body)}); err == nil {
			t.Fatal("unsafe request accepted", body)
		}
	}
	request := testRequest("chat", false)
	request.Stream = true
	if _, err := validatedBody(request); err == nil {
		t.Fatal("stream/body mismatch accepted")
	}
}
