package kimi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"omniproxy/internal/core/proxy"
	"strings"
	"testing"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func reply(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func TestRegionalNativeProtocolsPreserveThoughtAndToolHistory(t *testing.T) {
	for _, region := range []string{"cn", "global"} {
		for _, protocol := range []string{"chat", "responses", "messages"} {
			t.Run(region+"/"+protocol, func(t *testing.T) {
				o := Options{Region: region}
				expected, _ := endpoint(o, protocol)
				body := `{"model":"kimi-k3","messages":[{"role":"assistant","content":null,"reasoning_content":"prior thought","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read","arguments":"{}"}}]},{"role":"tool","tool_call_id":"call_1","content":"OK"}],"max_tokens":32,"reasoning_effort":"high"}`
				if protocol == "messages" {
					body = `{"model":"kimi-k3","messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"prior thought","signature":"opaque-signature"},{"type":"tool_use","id":"call_1","name":"read","input":{}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"OK"}]}],"max_tokens":32,"output_config":{"effort":"high"}}`
				}
				if protocol == "responses" {
					body = `{"model":"kimi-k3","input":[{"type":"reasoning","content":[{"type":"reasoning_text","text":"prior thought"}]},{"type":"custom_tool_call","call_id":"call_1","name":"apply_patch","input":"patch"},{"type":"custom_tool_call_output","call_id":"call_1","output":"OK"}],"tools":[{"type":"namespace","name":"functions","description":"tools","tools":[{"type":"custom","name":"apply_patch","format":{"type":"grammar","syntax":"lark","definition":"start: /[a-z]+/"}}]}],"reasoning":{"effort":"high"},"store":false,"max_output_tokens":32}`
				}
				calls := 0
				client := New(roundTrip(func(r *http.Request) (*http.Response, error) {
					calls++
					if r.URL.String() != expected || r.Header.Get("Authorization") != "Bearer placeholder-kimi-key" || r.Header.Get("X-Api-Key") != "" {
						t.Fatal("wrong endpoint or authentication")
					}
					raw, _ := io.ReadAll(r.Body)
					if !strings.Contains(string(raw), "prior thought") || !strings.Contains(string(raw), "call_1") {
						t.Fatal("native context lost")
					}
					if protocol == "messages" && !strings.Contains(string(raw), "opaque-signature") {
						t.Fatal("thinking signature lost")
					}
					return reply(200, `{"native":true}`), nil
				}))
				err := client.Forward(context.Background(), o, []byte("placeholder-kimi-key"), proxy.Request{Model: "kimi-k3", Protocol: protocol, Body: []byte(body)}, func(response proxy.Response) error {
					raw, err := io.ReadAll(response.Body)
					if string(raw) != `{"native":true}` {
						t.Fatal("native response changed")
					}
					return err
				})
				if err != nil || calls != 1 {
					t.Fatalf("forward: %v calls=%d", err, calls)
				}
			})
		}
	}
}

func TestModelParametersAndUnsupportedFeaturesFailBeforeNetwork(t *testing.T) {
	for _, tc := range []struct{ model, protocol, extra string }{
		{"kimi-k3", "chat", `"reasoning_effort":"medium"`},
		{"kimi-k3", "chat", `"thinking":{"type":"disabled"}`},
		{"kimi-k3", "chat", `"temperature":0.7`},
		{"kimi-k2.7-code", "chat", `"thinking":{"type":"disabled"}`},
		{"kimi-k2.6", "chat", `"tool_choice":"required"`},
		{"kimi-k2.6", "responses", `"max_output_tokens":32`},
		{"kimi-k3", "responses", `"previous_response_id":"resp_other"`},
		{"kimi-k3", "responses", `"store":true`},
	} {
		body := `{"model":"` + tc.model + `","messages":[{"role":"user","content":"hi"}],` + tc.extra + `}`
		if tc.protocol == "responses" {
			body = `{"model":"` + tc.model + `","input":"hi",` + tc.extra + `}`
		}
		client := New(roundTrip(func(*http.Request) (*http.Response, error) {
			t.Fatal("unsupported request reached upstream")
			return nil, nil
		}))
		if err := client.Forward(context.Background(), Options{Region: "cn"}, []byte("placeholder-kimi-key"), proxy.Request{Model: tc.model, Protocol: tc.protocol, Body: []byte(body)}, func(proxy.Response) error { return nil }); err == nil {
			t.Fatal("unsupported capability accepted")
		}
	}
	request := proxy.Request{Model: "kimi-k2.6", Protocol: "chat", Body: []byte(`{"model":"kimi-k2.6","messages":[{"role":"user","content":"hi"}],"thinking":{"type":"disabled"},"temperature":0.6}`)}
	if _, err := validatedBody(request); err != nil {
		t.Fatal("valid non-thinking model rejected")
	}
}

func TestRegionalErrorsNeverRetryOrRedirect(t *testing.T) {
	for _, status := range []int{301, 401, 402, 429, 503} {
		calls := 0
		client := New(roundTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.URL.Host != "api.moonshot.ai" {
				t.Fatal("region changed")
			}
			response := reply(status, `{"error":{"message":"private upstream"}}`)
			response.Header.Set("Location", DefaultBaseURL+"/chat/completions")
			return response, nil
		}))
		err := client.Forward(context.Background(), Options{Region: "global"}, []byte("placeholder-kimi-key"), proxy.Request{Model: "kimi-k3", Protocol: "chat", Body: []byte(`{"model":"kimi-k3","messages":[{"role":"user","content":"hi"}]}`)}, func(response proxy.Response) error {
			if response.StatusCode != status {
				t.Fatal("status changed")
			}
			return nil
		})
		if err != nil || calls != 1 {
			t.Fatal("error changed region or retried")
		}
	}
}

func TestChatStreamingRequestsFinalUsage(t *testing.T) {
	request := proxy.Request{Model: "kimi-k3", Protocol: "chat", Stream: true, Body: []byte(`{"model":"kimi-k3","stream":true,"messages":[{"role":"user","content":"hi"}],"stream_options":{"other":true}}`)}
	body, err := validatedBody(request)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	json.Unmarshal(body, &fields)
	if !strings.Contains(string(fields["stream_options"]), `"include_usage":true`) || !strings.Contains(string(fields["stream_options"]), `"other":true`) {
		t.Fatal("usage or existing options lost")
	}
}
