package minimax

import (
	"context"
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

func TestFourChannelsPreserveNativeHistory(t *testing.T) {
	for _, region := range []string{"cn", "global"} {
		for _, mode := range []string{"payg", "subscription"} {
			for _, protocol := range []string{"chat", "responses", "messages"} {
				t.Run(region+"/"+mode+"/"+protocol, func(t *testing.T) {
					o := Options{Region: region, BillingMode: mode}
					body := `{"model":"MiniMax-M3","messages":[{"role":"assistant","content":"<think>prior thought</think>","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read","arguments":"{}"}}]},{"role":"tool","tool_call_id":"call_1","content":"OK"}]}`
					if protocol == "messages" {
						body = `{"model":"MiniMax-M3","max_tokens":32,"messages":[{"role":"assistant","content":[{"type":"thinking","thinking":"prior thought","signature":"opaque-signature"},{"type":"tool_use","id":"call_1","name":"read","input":{}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_1","content":"OK"}]}]}`
					}
					if protocol == "responses" {
						body = `{"model":"MiniMax-M3","input":[{"type":"reasoning","content":[{"type":"reasoning_text","text":"prior thought"}]},{"type":"function_call","call_id":"call_1","name":"read","arguments":"{}"},{"type":"function_call_output","call_id":"call_1","output":"OK"}],"tools":[{"type":"function","name":"read","parameters":{}}],"store":false}`
					}
					host := "api.minimax.cn"
					if region == "global" {
						host = "api.minimax.io"
					}
					path := "/v1/" + protocol
					if protocol == "chat" {
						path = "/v1/chat/completions"
					}
					if protocol == "messages" {
						path = "/anthropic/v1/messages"
					}
					client := New(roundTrip(func(r *http.Request) (*http.Response, error) {
						if r.URL.Host != host || r.URL.Path != path || r.Header.Get("Authorization") != "Bearer placeholder-minimax-key" || r.Header.Get("X-Api-Key") != "" {
							t.Fatal("wrong platform/path/credential")
						}
						raw, err := io.ReadAll(r.Body)
						if err != nil {
							return nil, err
						}
						if !strings.Contains(string(raw), "prior thought") || !strings.Contains(string(raw), "call_1") || protocol == "messages" && !strings.Contains(string(raw), "opaque-signature") {
							t.Fatal("native history changed")
						}
						return reply(200, `{"native":true}`), nil
					}))
					if err := client.Forward(context.Background(), o, []byte("placeholder-minimax-key"), proxy.Request{Model: "MiniMax-M3", Protocol: protocol, Body: []byte(body)}, func(r proxy.Response) error {
						raw, err := io.ReadAll(r.Body)
						if string(raw) != `{"native":true}` {
							t.Fatal("response changed")
						}
						return err
					}); err != nil {
						t.Fatal(err)
					}
				})
			}
		}
	}
}

func TestFailuresNeverSwitchPlatformOrBilling(t *testing.T) {
	for _, status := range []int{301, 401, 402, 429, 503} {
		calls := 0
		c := New(roundTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.URL.Host != "api.minimax.io" {
				t.Fatal("platform changed")
			}
			rpl := reply(status, `{"error":{"message":"private"}}`)
			rpl.Header.Set("Location", DefaultBaseURL+"/chat/completions")
			return rpl, nil
		}))
		err := c.Forward(context.Background(), Options{Region: "global", BillingMode: "subscription"}, []byte("placeholder-key"), proxy.Request{Model: "MiniMax-M3", Protocol: "chat", Body: []byte(`{"model":"MiniMax-M3","messages":[{"role":"user","content":"hi"}]}`)}, func(r proxy.Response) error {
			if r.StatusCode != status {
				t.Fatal("status changed")
			}
			return nil
		})
		if err != nil || calls != 1 {
			t.Fatal("unexpected retry", err, calls)
		}
	}
}
