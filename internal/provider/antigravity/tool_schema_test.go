package antigravity

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

func TestToolSchemaDialectPreservesJSONSchema(t *testing.T) {
	// Claude Code 2.1.299 自带工具使用完整 JSON Schema；不能为适配而删除参数约束。
	const schema = `{"$schema":"http://json-schema.org/draft-07/schema#","type":"object","properties":{"limit":{"type":"integer","exclusiveMinimum":0,"maximum":100},"filter":{"anyOf":[{"type":"string"},{"type":"null"}]},"options":{"type":"object","additionalProperties":{"type":"boolean"}}},"required":["limit"],"additionalProperties":false,"$defs":{"unused":{"type":"string"}}}`
	var want any
	if err := json.Unmarshal([]byte(schema), &want); err != nil {
		t.Fatal(err)
	}
	for _, model := range []string{"gemini-3.8-flash-high", "gemini-2.5-flash", "claude-sonnet-4-6", "gpt-oss-120b-medium"} {
		t.Run(model, func(t *testing.T) {
			body, err := chatPayload(proxy.Request{
				Protocol: "chat", Model: model,
				Body: []byte(`{"messages":[{"role":"user","content":"hello"}],"tools":[{"type":"function","function":{"name":"inspect","parameters":` + schema + `}}]}`),
			}, testCredentials())
			if err != nil {
				t.Fatal(err)
			}
			var envelope struct {
				Request struct {
					Tools []struct {
						Functions []map[string]any `json:"functionDeclarations"`
					} `json:"tools"`
				} `json:"request"`
			}
			if err := json.Unmarshal(body, &envelope); err != nil {
				t.Fatal(err)
			}
			if len(envelope.Request.Tools) != 1 || len(envelope.Request.Tools[0].Functions) != 1 {
				t.Fatal("missing tool declaration")
			}
			function := envelope.Request.Tools[0].Functions[0]
			field, other := "parameters", "parametersJsonSchema"
			if strings.HasPrefix(model, "gemini-") {
				field, other = other, field
			}
			if function["name"] != "inspect" || !reflect.DeepEqual(function[field], want) {
				t.Fatalf("tool schema was lost or used the wrong dialect: %s", field)
			}
			if _, exists := function[other]; exists {
				t.Fatal("mutually exclusive schema fields were both emitted")
			}
		})
	}
}

func TestToolSchemaInputBoundaries(t *testing.T) {
	for _, parameters := range []string{`[]`, `"schema"`, `true`, `1`} {
		t.Run(parameters, func(t *testing.T) {
			_, err := chatPayload(proxy.Request{
				Protocol: "chat", Model: "gemini-test",
				Body: []byte(`{"messages":[{"role":"user","content":"hello"}],"tools":[{"type":"function","function":{"name":"inspect","parameters":` + parameters + `}}]}`),
			}, testCredentials())
			if err == nil {
				t.Fatal("non-object schema was accepted")
			}
		})
	}
	for _, parameters := range []string{"", `,"parameters":null`} {
		body, err := chatPayload(proxy.Request{
			Protocol: "chat", Model: "gemini-test",
			Body: []byte(`{"messages":[{"role":"user","content":"hello"}],"tools":[{"type":"function","function":{"name":"inspect"` + parameters + `}}]}`),
		}, testCredentials())
		if err != nil || strings.Contains(string(body), `"parameters"`) || strings.Contains(string(body), `"parametersJsonSchema"`) {
			t.Fatal("absent schema handling changed", err)
		}
	}
}
