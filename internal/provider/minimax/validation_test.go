package minimax

import (
	"context"
	"net/http"
	"omniproxy/internal/core/proxy"
	"strings"
	"testing"
)

func TestInvalidOrUnsupportedRequestNeverSendsKey(t *testing.T) {
	for _, tc := range []struct{ protocol, model, extra, input string }{
		{"chat", "MiniMax-M3", `"temperature":3`, ""},
		{"chat", "MiniMax-M3", `"top_p":null`, ""},
		{"chat", "MiniMax-M3", `"max_tokens":0`, ""},
		{"chat", "MiniMax-M3", `"n":2`, ""},
		{"chat", "MiniMax-M3", `"reasoning_effort":"unknown"`, ""},
		{"chat", PreviewModel, `"thinking":{"type":"disabled"}`, ""},
		{"chat", "MiniMax-M2.7", `"thinking":{"type":"disabled"}`, ""},
		{"chat", "MiniMax-M3", `"service_tier":"free"`, ""},
		{"chat", "MiniMax-M3", `"tools":[{"type":"web_search"}]`, ""},
		{"responses", "MiniMax-M3", `"store":true`, ""},
		{"responses", "MiniMax-M3", `"store":null`, ""},
		{"chat", "MiniMax-M3", `"tools":null`, ""},
		{"chat", "MiniMax-M3", `"tools":[{"type":"function","function":{"name":"bad name"}}]`, ""},
		{"responses", "MiniMax-M3", `"background":true`, ""},
		{"responses", "MiniMax-M3", `"previous_response_id":"resp_secret"`, ""},
		{"responses", "MiniMax-M3", `"tool_choice":"required"`, ""},
		{"responses", "MiniMax-M3", `"tools":[{"type":"namespace","name":"functions","tools":[]}]`, ""},
		{"responses", "MiniMax-M3", `"tools":[{"type":"custom","name":"apply_patch"}]`, ""},
		{"messages", "MiniMax-M3", `"max_tokens":32`, `[{"role":"user","content":[{"type":"image","source":{"type":"url","url":"https://example.invalid"}}]}]`},
	} {
		input := tc.input
		if input == "" {
			input = `[{"role":"user","content":"hi"}]`
		}
		field := "messages"
		if tc.protocol == "responses" {
			field = "input"
			input = `"hi"`
		}
		body := `{"model":"` + tc.model + `","` + field + `":` + input + `,` + tc.extra + `}`
		c := New(roundTrip(func(*http.Request) (*http.Response, error) {
			t.Fatal("invalid request reached network")
			return nil, nil
		}))
		err := c.Forward(context.Background(), Options{Region: "cn", BillingMode: "subscription"}, []byte("placeholder-key"), proxy.Request{Model: tc.model, Protocol: tc.protocol, Body: []byte(body)}, func(proxy.Response) error { return nil })
		if err == nil {
			t.Fatal("unsupported request accepted", tc.extra)
		}
	}
}

func TestModelGatingAndKeyType(t *testing.T) {
	payg := Options{Region: "cn", BillingMode: "payg"}
	plan := Options{Region: "cn", BillingMode: "subscription"}
	if Supports(payg, PreviewModel, "chat") || !Supports(plan, PreviewModel, "responses") {
		t.Fatal("preview gating")
	}
	for _, id := range AccountModels(plan) {
		if id == PreviewModel {
			t.Fatal("legacy subscription granted preview without discovery")
		}
	}
	if ValidKeyForOptions(plan, []byte("sk-api-placeholder")) || ValidKeyForOptions(payg, []byte("sk-cp-placeholder")) {
		t.Fatal("mismatched key types")
	}
	if ValidAPIKey([]byte("bad\nheader")) || ValidAPIKey([]byte(strings.Repeat("x", 8193))) {
		t.Fatal("unsafe header accepted")
	}
	for _, model := range []string{"MiniMax-M3", PreviewModel} {
		r := proxy.Request{Model: model, Protocol: "chat", Stream: true, Body: []byte(`{"model":"` + model + `","stream":true,"messages":[{"role":"user","content":"hi"}],"stream_options":{"include_usage":false}}`)}
		raw, err := validatedBody(plan, r)
		if err != nil || !strings.Contains(string(raw), `"include_usage":true`) {
			t.Fatal("final usage missing", err)
		}
	}
}
