package stepfun

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

func TestFourChannelsPreserveNativeProtocols(t *testing.T) {
	for _, region := range []string{"cn", "global"} {
		for _, mode := range []string{"payg", "step_plan"} {
			for _, protocol := range []string{"chat", "responses", "messages"} {
				t.Run(region+"/"+mode+"/"+protocol, func(t *testing.T) {
					o := Options{Region: region, BillingMode: mode}
					base, _ := o.BaseURL()
					path := "/" + protocol
					body := `{"model":"step-5-preview","messages":[{"role":"user","content":"hello"}],"max_tokens":32}`
					if protocol == "chat" {
						path = "/chat/completions"
					}
					if protocol == "responses" {
						body = `{"model":"step-5-preview","input":[{"type":"function_call","id":"fc_1","call_id":"call_1","name":"read","arguments":"{}"},{"type":"function_call_output","call_id":"call_1","output":"OK"}],"max_output_tokens":32,"store":false}`
					}
					called := 0
					client := New(roundTrip(func(r *http.Request) (*http.Response, error) {
						called++
						if r.URL.String() != base+path || r.Header.Get("Authorization") != "Bearer placeholder-step-key" || r.Header.Get("X-Api-Key") != "" {
							t.Fatal("wrong native endpoint or authentication")
						}
						raw, _ := io.ReadAll(r.Body)
						if protocol == "responses" && (!strings.Contains(string(raw), `"call_id":"call_1"`) || strings.Contains(string(raw), `"store":`)) {
							t.Fatal("tool history or stateless handling changed")
						}
						return reply(200, `{"result":"native"}`), nil
					}))
					err := client.Forward(context.Background(), o, []byte("placeholder-step-key"), proxy.Request{Model: "step-5-preview", Protocol: protocol, Body: []byte(body)}, func(response proxy.Response) error {
						raw, err := io.ReadAll(response.Body)
						if string(raw) != `{"result":"native"}` {
							t.Fatal("native response changed")
						}
						return err
					})
					if err != nil || called != 1 {
						t.Fatalf("forward: %v calls=%d", err, called)
					}
				})
			}
		}
	}
}

func TestUnsafeRequestsNeverReachUpstream(t *testing.T) {
	client := New(roundTrip(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid request reached upstream")
		return nil, nil
	}))
	for _, tc := range []struct{ protocol, body string }{
		{"chat", `{"model":"step-5-preview","messages":[{"role":"user","content":"hello"}],"max_tokens":64001}`},
		{"chat", `{"model":"step-5-preview","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"https://example.com"}}]}]}`},
		{"responses", `{"model":"step-5-preview","input":"hello","previous_response_id":"resp_old"}`},
		{"responses", `{"model":"step-5-preview","input":"hello","store":true}`},
		{"messages", `{"model":"step-5-preview","messages":[{"role":"user","content":"hi"}],"tools":[{"type":"web_search","name":"search"}]}`},
	} {
		if err := client.Forward(context.Background(), Options{Region: "cn", BillingMode: "payg"}, []byte("placeholder-step-key"), proxy.Request{Model: "step-5-preview", Protocol: tc.protocol, Body: []byte(tc.body)}, func(proxy.Response) error { return nil }); err == nil {
			t.Fatal("unsafe request accepted")
		}
	}
	if Supports(Options{Region: "cn", BillingMode: "payg"}, "step-router-v1", "messages") || Supports(Options{Region: "cn", BillingMode: "step_plan"}, "step-router-v1", "responses") {
		t.Fatal("Router capability invented")
	}
}

func TestErrorsDoNotChangeRegionOrBilling(t *testing.T) {
	for _, status := range []int{301, 401, 402, 429, 503} {
		calls := 0
		client := New(roundTrip(func(r *http.Request) (*http.Response, error) {
			calls++
			if r.URL.String() != "https://api.stepfun.ai/step_plan/v1/chat/completions" {
				t.Fatal("billing channel changed")
			}
			response := reply(status, `{"error":"private upstream error"}`)
			response.Header.Set("Location", "https://api.stepfun.com/v1/chat/completions")
			return response, nil
		}))
		err := client.Forward(context.Background(), Options{Region: "global", BillingMode: "step_plan"}, []byte("placeholder-step-key"), proxy.Request{Model: "step-5-preview", Protocol: "chat", Body: []byte(`{"model":"step-5-preview","messages":[{"role":"user","content":"hi"}]}`)}, func(response proxy.Response) error {
			if response.StatusCode != status {
				t.Fatal("status changed")
			}
			return nil
		})
		if err != nil || calls != 1 {
			t.Fatalf("error retried or redirected: %v calls=%d", err, calls)
		}
	}
}
