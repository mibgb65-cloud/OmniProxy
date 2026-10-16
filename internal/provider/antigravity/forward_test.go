package antigravity

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

func TestChatToolSignatureRoundTripAndUsage(t *testing.T) {
	cred := testCredentials()
	first := `{"response":{"candidates":[{"content":{"parts":[{"functionCall":{"id":"call_test","name":"weather","args":{"city":"Paris"}},"thoughtSignature":"test-opaque-signature"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":5,"candidatesTokenCount":2,"thoughtsTokenCount":3,"totalTokenCount":10}}}`
	calls := 0
	client := New(transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Host != "daily-cloudcode-pa.googleapis.com" || r.Header.Get("Authorization") != "Bearer test-access" {
			t.Error("upstream identity mismatch")
		}
		data, _ := io.ReadAll(r.Body)
		if strings.Contains(string(data), "allowedCreditTypes") {
			t.Error("paid credits were enabled")
		}
		if calls == 1 {
			return reply(200, first), nil
		}
		if !strings.Contains(string(data), `"thoughtSignature":"test-opaque-signature"`) || !strings.Contains(string(data), `"functionResponse"`) {
			t.Error("tool association or signature was lost")
		}
		return reply(200, `{"response":{"candidates":[{"content":{"parts":[{"text":"Sunny"}]},"finishReason":"STOP"}]}}`), nil
	}))
	request := proxy.Request{Model: "gemini-test", Protocol: "chat", Body: []byte(`{"messages":[{"role":"user","content":"Weather?"}],"tools":[{"type":"function","function":{"name":"weather","parameters":{"type":"object"}}}]}`)}
	var result object
	err := client.Forward(context.Background(), cred, request, func(response proxy.Response) error { return json.NewDecoder(response.Body).Decode(&result) })
	if err != nil {
		t.Fatal(err)
	}
	message := result["choices"].([]any)[0].(map[string]any)["message"]
	usage := result["usage"].(map[string]any)
	if usage["completion_tokens"] != float64(5) {
		t.Fatal("thinking token usage was lost")
	}
	request.Body, _ = json.Marshal(object{"messages": []any{object{"role": "user", "content": "Weather?"}, message, object{"role": "tool", "tool_call_id": "call_test", "content": "sunny"}}})
	if err := client.Forward(context.Background(), cred, request, func(response proxy.Response) error { _, err := io.Copy(io.Discard, response.Body); return err }); err != nil {
		t.Fatal(err)
	}
}

func TestStreamingIsIncrementalAndTruncationFails(t *testing.T) {
	const textFrame = `{"response":{"candidates":[{"content":{"parts":[{"text":"hello"}]}}]}}`
	const endFrame = `{"response":{"candidates":[{"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":2,"candidatesTokenCount":1,"totalTokenCount":3}}}`
	for _, truncated := range []bool{false, true} {
		t.Run(map[bool]string{false: "complete", true: "truncated"}[truncated], func(t *testing.T) {
			read, write := io.Pipe()
			client := New(transportFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: read}, nil
			}))
			defer read.Close()
			defer write.Close()
			go func() {
				defer write.Close()
				io.WriteString(write, "data: "+textFrame+"\n\n")
				if !truncated {
					io.WriteString(write, "data: "+endFrame+"\n\n")
				}
			}()
			err := client.Forward(context.Background(), testCredentials(), proxy.Request{Model: "gemini-test", Protocol: "chat", Stream: true, Body: []byte(`{"messages":[{"role":"user","content":"hi"}]}`)}, func(response proxy.Response) error {
				buffer := make([]byte, 4096)
				n, err := response.Body.Read(buffer)
				if err != nil || !strings.Contains(string(buffer[:n]), "hello") || strings.Contains(string(buffer[:n]), "[DONE]") {
					t.Fatal("first delta waited for stream completion")
				}
				data, err := io.ReadAll(response.Body)
				if truncated {
					if err == nil {
						t.Fatal("truncated stream reported success")
					}
				} else if err != nil || !strings.Contains(string(data), "[DONE]") || !strings.Contains(string(data), "prompt_tokens") {
					t.Fatal("terminal usage or completion missing")
				}
				return err
			})
			if truncated && err == nil {
				t.Fatal("stream failure was swallowed")
			}
		})
	}
}

func TestUnsupportedInputAndCancelledConsumer(t *testing.T) {
	for _, body := range []string{`{"messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"file:///secret"}}]}]}`, `{"messages":[{"role":"tool","tool_call_id":"missing","content":"data"}]}`, `{"messages":[{"role":"user","content":"hi"}],"response_format":{"type":"json_object"}}`} {
		if _, err := chatPayload(proxy.Request{Model: "gemini-test", Protocol: "chat", Body: []byte(body)}, testCredentials()); err == nil {
			t.Fatal("unsupported input accepted")
		}
	}
	stopped := errors.New("consumer stopped")
	client := New(transportFunc(func(r *http.Request) (*http.Response, error) {
		return reply(200, `{"response":{"candidates":[{"content":{"parts":[{"text":"hi"}]},"finishReason":"STOP"}]}}`), nil
	}))
	err := client.Forward(context.Background(), testCredentials(), proxy.Request{Model: "gemini-test", Protocol: "chat", Body: []byte(`{"messages":[{"role":"user","content":"hi"}]}`)}, func(proxy.Response) error { return stopped })
	if !errors.Is(err, stopped) {
		t.Fatal("consumer cancellation was swallowed")
	}
}
