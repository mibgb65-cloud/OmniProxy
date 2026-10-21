package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/antigravity"
)

func TestGoogleAgentToolJSONSchemaAcrossProtocols(t *testing.T) {
	const rawSchema = `{"$schema":"http://json-schema.org/draft-07/schema#","type":"object","properties":{"city":{"type":"string"},"limit":{"type":"integer","exclusiveMinimum":0}},"required":["city"],"additionalProperties":false}`
	var schema map[string]any
	if err := json.Unmarshal([]byte(rawSchema), &schema); err != nil {
		t.Fatal(err)
	}
	for _, model := range []string{"gemini-3.8-flash-high", "claude-sonnet-4-6", "gpt-oss-120b-medium"} {
		for _, protocol := range []string{"chat", "messages", "responses"} {
			for _, stream := range []bool{false, true} {
				name := model + "/" + protocol + map[bool]string{false: "/json", true: "/sse"}[stream]
				t.Run(name, func(t *testing.T) {
					calls := 0
					service := &Service{antigravity: antigravity.New(googleTransport(func(r *http.Request) (*http.Response, error) {
						calls++
						var envelope struct {
							Request struct {
								Tools []struct {
									Functions []map[string]any `json:"functionDeclarations"`
								} `json:"tools"`
							} `json:"request"`
						}
						if err := json.NewDecoder(r.Body).Decode(&envelope); err != nil {
							t.Fatal(err)
						}
						if len(envelope.Request.Tools) != 1 || len(envelope.Request.Tools[0].Functions) != 1 {
							t.Fatal("tool declaration lost in protocol conversion")
						}
						function := envelope.Request.Tools[0].Functions[0]
						field, other := "parameters", "parametersJsonSchema"
						if strings.HasPrefix(model, "gemini-") {
							field, other = other, field
						}
						if !reflect.DeepEqual(function[field], schema) {
							t.Fatal("protocol conversion changed tool schema or dialect")
						}
						if _, exists := function[other]; exists {
							t.Fatal("schema dialect fields must be mutually exclusive")
						}
						body := `{"response":{"candidates":[{"content":{"parts":[{"text":"OK"}]},"finishReason":"STOP"}]}}`
						contentType := "application/json"
						if r.URL.Path == "/v1internal:streamGenerateContent" {
							contentType, body = "text/event-stream", "data: "+body+"\n\n"
						}
						return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {contentType}}, Body: io.NopCloser(strings.NewReader(body))}, nil
					}))}
					body, _ := googleAgentRequest(model, protocol, stream, false)
					var fields map[string]any
					if err := json.Unmarshal([]byte(body), &fields); err != nil {
						t.Fatal(err)
					}
					tool := fields["tools"].([]any)[0].(map[string]any)
					switch protocol {
					case "chat":
						tool["function"].(map[string]any)["parameters"] = schema
					case "messages":
						tool["input_schema"] = schema
					case "responses":
						tool["parameters"] = schema
					}
					encoded, err := json.Marshal(fields)
					if err != nil {
						t.Fatal(err)
					}
					consumed := false
					err = service.forwardGoogleAgent(context.Background(), antigravity.Credentials{}, proxy.Request{
						Model: model, Protocol: protocol, Stream: stream, Body: encoded,
					}, func(response proxy.Response) error {
						consumed = true
						if closer, ok := response.Body.(io.Closer); ok {
							defer closer.Close()
						}
						data, err := io.ReadAll(response.Body)
						if response.StatusCode != 200 || !strings.Contains(string(data), "OK") {
							t.Fatal("missing successful response")
						}
						if stream && response.ContentType != "text/event-stream" {
							t.Fatal("missing downstream SSE")
						}
						return err
					})
					if err != nil || !consumed || calls != 1 {
						t.Fatal("request failed or was retried", err, calls)
					}
				})
			}
		}
	}
}
