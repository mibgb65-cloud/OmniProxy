package river

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

func TestNativeProtocolsPreserveToolsReasoningAndUsage(t *testing.T) {
	for _, protocol := range []string{"chat", "responses"} {
		for _, stream := range []bool{false, true} {
			t.Run(protocol+map[bool]string{false: "/json", true: "/sse"}[stream], func(t *testing.T) {
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					path, _ := protocolPath(protocol)
					if r.URL.Path != "/v1"+path || r.Header.Get("Authorization") != "Bearer test-key" || r.Header.Get("X-Api-Key") != "" {
						t.Error("unsafe upstream request")
					}
					var fields map[string]json.RawMessage
					if json.NewDecoder(r.Body).Decode(&fields) != nil || !strings.Contains(string(fields["tools"]), "lookup") {
						t.Error("tools lost")
					}
					if stream && protocol == "chat" && !strings.Contains(string(fields["stream_options"]), `"include_usage":true`) {
						t.Error("usage not requested")
					}
					w.Header().Set("Content-Type", "application/json")
					if stream {
						w.Header().Set("Content-Type", "text/event-stream")
					}
					io.WriteString(w, "native payload")
				}))
				defer upstream.Close()
				fields := map[string]any{"model": DeepSeekFlash, "stream": stream, "tools": []any{map[string]any{"type": "function", "function": map[string]any{"name": "lookup", "parameters": map[string]any{"type": "object"}}}}}
				if protocol == "chat" {
					fields["messages"] = []any{map[string]any{"role": "user", "content": "hello"}}
					fields["reasoning_effort"] = "max"
				} else {
					fields["tools"] = []any{map[string]any{"type": "function", "name": "lookup", "parameters": map[string]any{"type": "object"}}}
					fields["input"] = "hello"
					fields["reasoning"] = map[string]any{"effort": "high"}
					fields["store"] = false
				}
				body, _ := json.Marshal(fields)
				err := New(nil).Forward(context.Background(), upstream.URL+"/v1/", []byte("test-key"), proxy.Request{Model: DeepSeekFlash, Protocol: protocol, Stream: stream, Body: body}, func(r proxy.Response) error {
					data, err := io.ReadAll(r.Body)
					if string(data) != "native payload" || r.StatusCode != 200 {
						t.Error("native response changed")
					}
					return err
				})
				if err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

func TestInvalidRequestsNeverReachUpstream(t *testing.T) {
	for _, tc := range []struct{ protocol, model, extra string }{
		{"chat", DeepSeekFlash, `"max_tokens":384001`},
		{"chat", DeepSeekFlash1017, `"max_tokens":384001`},
		{"responses", DeepSeekFlash1017, `"max_output_tokens":384001`},
		{"chat", GLMFlash, `"max_tokens":131073`},
		{"chat", GLMFlash, `"reasoning_effort":"none"`},
		{"chat", NemotronVL, `"reasoning_effort":"high"`},
		{"chat", DeepSeekFlash, `"max_tokens":"16"`},
		{"chat", DeepSeekFlash, `"max_completion_tokens":null`},
		{"chat", DeepSeekFlash, `"max_tokens":1.5`},
		{"chat", DeepSeekFlash, `"temperature":3`},
		{"responses", DeepSeekFlash, `"store":true`},
		{"responses", DeepSeekFlash, `"background":true`},
		{"responses", DeepSeekFlash, `"previous_response_id":"r1"`},
		{"responses", DeepSeekFlash, `"conversation":{}`},
		{"responses", DeepSeekFlash, `"input":[{"role":"user","content":[{"type":"input_image","image_url":"https://example.test/a.png"}]}]`},
		{"messages", DeepSeekFlash, `"max_tokens":16`},
		{"chat", "unverified-model", `"max_tokens":16`},
	} {
		t.Run(tc.protocol+tc.extra+tc.model, func(t *testing.T) {
			body := []byte(`{"model":"` + tc.model + `","messages":[{"role":"user","content":"hello"}],"input":"hello",` + tc.extra + `}`)
			called := false
			c := New(roundTripFunc(func(*http.Request) (*http.Response, error) {
				called = true
				return nil, errors.New("unexpected request")
			}))
			if err := c.Forward(context.Background(), "", []byte("test-key"), proxy.Request{Model: tc.model, Protocol: tc.protocol, Body: body}, func(proxy.Response) error { return nil }); err == nil || called {
				t.Fatal("invalid request reached upstream", err)
			}
		})
	}
}

func TestUpstreamFailuresRedirectsAndCancellation(t *testing.T) {
	for _, status := range []int{401, 403, 402, 429, 500, 503, 307} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				w.Header().Set("Retry-After", "7")
				w.Header().Set("Location", "/redirect")
				w.WriteHeader(status)
			}))
			defer upstream.Close()
			r := proxy.Request{Model: DeepSeekFlash, Protocol: "chat", Body: []byte(`{"model":"` + DeepSeekFlash + `","messages":[{"role":"user","content":"hello"}]}`)}
			if err := New(nil).Forward(context.Background(), upstream.URL+"/v1", []byte("test-key"), r, func(response proxy.Response) error {
				if response.StatusCode != status || response.RetryAfter != "7" {
					t.Error("failure contract lost")
				}
				return nil
			}); err != nil || calls != 1 {
				t.Fatal("redirect/retry occurred", err, calls)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := New(nil).Forward(ctx, upstream.URL+"/v1", []byte("test-key"), r, func(proxy.Response) error { return nil }); err == nil {
				t.Fatal("cancellation ignored")
			}
			for _, base := range []string{"http://example.test/v1", "https://user:secret@example.test/v1", "https://example.test/v1?key=secret", "https://example.test:65536/v1"} {
				if err := New(nil).Forward(context.Background(), base, []byte("test-key"), r, func(proxy.Response) error { return nil }); err == nil {
					t.Fatal("invalid endpoint accepted")
				}
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
