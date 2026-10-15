package mimo

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"omniproxy/internal/core/proxy"
)

type planTransport func(*http.Request) (*http.Response, error)

func (f planTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestTokenPlanRegionalNativeContracts(t *testing.T) {
	paths := map[string]string{"chat": "/v1/chat/completions", "responses": "/v1/responses", "messages": "/anthropic/v1/messages"}
	for _, region := range []string{"cn", "sgp", "ams"} {
		for _, key := range []string{"tp-placeholder-personal-key", "ttp-placeholder-team-key"} {
			for protocol, path := range paths {
				for _, stream := range []bool{false, true} {
					t.Run(region+"/"+key[:3]+"/"+protocol+"/"+map[bool]string{true: "stream", false: "json"}[stream], func(t *testing.T) {
						o := Options{BillingMode: "token_plan", Region: region}
						base, err := o.BaseURL("https://untrusted.invalid")
						if err != nil {
							t.Fatal(err)
						}
						var request *http.Request
						c := New(planTransport(func(r *http.Request) (*http.Response, error) {
							request = r
							if r.URL.String() != "https://token-plan-"+region+".xiaomimimo.com"+path || r.Header.Get("Api-Key") != key || r.Header.Get("Authorization") != "" {
								t.Fatal("wrong cluster, protocol or credential")
							}
							raw, err := io.ReadAll(r.Body)
							if err != nil || !strings.Contains(string(raw), `"model":"mimo-v2.6-pro"`) {
								t.Fatal("native body lost")
							}
							kind, body := "application/json", `{"native":true}`
							if stream {
								kind, body = "text/event-stream", "data: native\n\ndata: [DONE]\n\n"
							}
							return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{kind}}, Body: io.NopCloser(strings.NewReader(body))}, nil
						}))
						if err := c.Forward(context.Background(), base, []byte(key), testRequest(protocol, stream), func(response proxy.Response) error {
							raw, err := io.ReadAll(response.Body)
							if !strings.Contains(string(raw), "native") {
								t.Fatal("native output lost")
							}
							return err
						}); err != nil {
							t.Fatal(err)
						}
						if request.Header.Get("Api-Key") != "" {
							t.Fatal("credential header retained")
						}
					})
				}
			}
		}
	}
}

func TestTokenPlanRejectsCredentialMixingUnknownClustersAndUnsupportedModels(t *testing.T) {
	calls := 0
	c := New(planTransport(func(*http.Request) (*http.Response, error) {
		calls++
		t.Fatal("invalid request reached network")
		return nil, nil
	}))
	for _, tc := range []struct{ base, key string }{
		{DefaultBaseURL, "tp-placeholder-key"}, {DefaultBaseURL, "ttp-placeholder-key"},
		{"https://token-plan-cn.xiaomimimo.com/v1", "sk-placeholder-key"},
		{"https://token-plan-sgp.xiaomimimo.com.evil.invalid/v1", "tp-placeholder-key"},
		{"https://token-plan-unknown.xiaomimimo.com/v1", "tp-placeholder-key"},
		{"http://token-plan-ams.xiaomimimo.com/v1", "tp-placeholder-key"},
		{"https://token-plan-ams.xiaomimimo.com/v1?secret=value", "tp-placeholder-key"},
	} {
		if err := c.Forward(context.Background(), tc.base, []byte(tc.key), testRequest("chat", false), func(proxy.Response) error { return nil }); err == nil {
			t.Fatal("invalid channel accepted")
		}
	}
	r := testRequest("chat", false)
	r.Model = "mimo-v2.6-pro-ultraspeed"
	if err := c.Forward(context.Background(), "https://token-plan-cn.xiaomimimo.com/v1", []byte("tp-placeholder-key"), r, func(proxy.Response) error { return nil }); err == nil || calls != 0 {
		t.Fatal("unsupported plan model accepted")
	}
	for _, o := range []Options{{Region: "global", BillingMode: "token_plan"}, {Region: "cn", BillingMode: "payg"}, {Region: "cn", BillingMode: "token_plan", Version: -1}} {
		if _, err := o.BaseURL(""); err == nil {
			t.Fatal("invalid options accepted")
		}
	}
}

func TestTokenPlanDiscoveryFiltersOrdinaryOnlyModelsAndKeepsFailures(t *testing.T) {
	calls := 0
	c := New(planTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://token-plan-sgp.xiaomimimo.com/v1/models" || r.Header.Get("Api-Key") != "ttp-placeholder-key" {
			t.Fatal("wrong plan discovery")
		}
		status, body := 200, `{"data":[{"id":"mimo-v2.6-pro"},{"id":"mimo-v2.6-pro-ultraspeed"},{"id":"mimo-v2.6-flash"}]}`
		if calls > 1 {
			status = 401
			body = `{"error":{"message":"private upstream failure"}}`
		}
		return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	models, err := c.DiscoverModels(context.Background(), "https://token-plan-sgp.xiaomimimo.com/v1", []byte("ttp-placeholder-key"))
	if err != nil || len(models) != 2 {
		t.Fatalf("plan models=%v err=%v", models, err)
	}
	if _, err := c.DiscoverModels(context.Background(), "https://token-plan-sgp.xiaomimimo.com/v1", []byte("ttp-placeholder-key")); err == nil || strings.Contains(err.Error(), "private") || calls != 2 {
		t.Fatal("discovery failure exposed or retried")
	}
}
