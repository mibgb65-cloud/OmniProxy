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

func TestCodingNativeProtocolsRegionsModelsAndRealClientIdentity(t *testing.T) {
	for _, region := range []string{"cn", "global"} {
		o := Options{Region: region, BillingMode: "coding"}
		for _, model := range AccountModels(o) {
			for _, protocol := range []string{"chat", "responses", "messages"} {
				for _, stream := range []bool{false, true} {
					body := `{"model":"` + model + `","stream":` + map[bool]string{false: "false", true: "true"}[stream] + `,"messages":[{"role":"user","content":"hi"}],"max_tokens":32}`
					if protocol == "responses" {
						body = `{"model":"` + model + `","stream":` + map[bool]string{false: "false", true: "true"}[stream] + `,"input":"hi","max_output_tokens":32}`
					}
					calls := 0
					client := New(roundTrip(func(r *http.Request) (*http.Response, error) {
						calls++
						host := "api.kimi.com"
						if region == "global" {
							host = "api.kimi.ai"
						}
						path := "/coding/v1/" + protocol
						if protocol == "chat" {
							path = "/coding/v1/chat/completions"
						}
						if r.URL.Host != host || r.URL.Path != path || r.Header.Get("User-Agent") != "integration-client/1.0" || r.Header.Get("Authorization") != "Bearer placeholder-code-key" {
							t.Fatal("wrong identity, endpoint or authentication")
						}
						var fields map[string]json.RawMessage
						wireModel := model
						if model == "kimi-k3" {
							wireModel = "k3"
						}
						if model == "kimi-k2.7-code-highspeed" {
							wireModel = "kimi-for-coding-highspeed"
						}
						if json.NewDecoder(r.Body).Decode(&fields) != nil || string(fields["model"]) != `"`+wireModel+`"` {
							t.Fatal("wrong canonical model translation")
						}
						content := `{"native":true}`
						if stream {
							content = "data: native\n\ndata: [DONE]\n\n"
						}
						return reply(200, content), nil
					}))
					err := client.Forward(context.Background(), o, []byte("placeholder-code-key"), proxy.Request{Model: model, Protocol: protocol, Body: []byte(body), Stream: stream, ClientUserAgent: "integration-client/1.0"}, func(r proxy.Response) error {
						raw, err := io.ReadAll(r.Body)
						if len(raw) == 0 {
							t.Fatal("native body lost")
						}
						return err
					})
					if err != nil || calls != 1 {
						t.Fatalf("%s/%s/%s/%t: %v", region, model, protocol, stream, err)
					}
				}
			}
		}
	}
}

func TestCodingSafetyBoundariesAndMetadata(t *testing.T) {
	ctx := context.Background()
	o := Options{Region: "cn", BillingMode: "coding"}
	client := New(roundTrip(func(*http.Request) (*http.Response, error) {
		t.Fatal("unsupported call reached network")
		return nil, nil
	}))
	if _, err := client.ReadBalance(ctx, o, []byte("placeholder")); err == nil {
		t.Fatal("subscription queried ordinary balance")
	}
	if _, err := client.DiscoverModels(ctx, o, []byte("placeholder")); err == nil {
		t.Fatal("undocumented discovery queried")
	}
	for _, tc := range []struct{ model, extra, ua string }{
		{"kimi-k2.7-code", "", "client"},
		{"kimi-k3", `,"thinking":{"type":"disabled"}`, "client"},
		{"kimi-k2.7-code-highspeed", `,"reasoning_effort":"high"`, "client"},
		{"kimi-k3", "", "forged\r\nAuthorization: private"},
		{"kimi-k3", "", strings.Repeat("x", 1025)},
	} {
		r := proxy.Request{Model: tc.model, Protocol: "chat", ClientUserAgent: tc.ua, Body: []byte(`{"model":"` + tc.model + `","messages":[{"role":"user","content":"hi"}]` + tc.extra + `}`)}
		if err := client.Forward(ctx, o, []byte("placeholder"), r, func(proxy.Response) error { return nil }); err == nil {
			t.Fatal("invalid coding request accepted")
		}
	}
	metadata := AccountModelMetadata(o, "kimi-k3")
	if metadata.ContextWindow == nil || *metadata.ContextWindow != 262144 || metadata.ContextVerified {
		t.Fatal("unverified tier advertised 1M")
	}
	if AccountModelMetadata(o, "kimi-for-coding").DefaultEffort != "max" {
		t.Fatal("K2.8 default effort lost")
	}
	if SupportsAccount(Options{Region: "cn"}, "kimi-for-coding", "chat") {
		t.Fatal("subscription model leaked to ordinary API")
	}
}
