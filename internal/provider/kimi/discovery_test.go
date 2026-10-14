package kimi

import (
	"context"
	"net/http"
	"testing"
)

func TestDiscoveryFiltersRetiredModelsAndKeepsOfficialContext(t *testing.T) {
	client := New(roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://api.moonshot.cn/v1/models" {
			t.Fatal("wrong discovery endpoint")
		}
		return reply(200, `{"data":[{"id":"kimi-k3","context_length":1048576},{"id":"kimi-k2.7-code-highspeed","context_length":262144},{"id":"kimi-k3"},{"id":"kimi-k2.5"},{"id":"moonshot-v1-8k"}]}`), nil
	}))
	models, err := client.DiscoverModels(context.Background(), Options{Region: "cn"}, []byte("placeholder-kimi-key"))
	if err != nil || len(models) != 2 || *models[0].Metadata.ContextWindow != 1048576 {
		t.Fatalf("models=%+v %v", models, err)
	}
	if ModelMetadata("kimi-k3").ContextVerified || ModelMetadata("kimi-k3").DefaultEffort != "max" || Supports("kimi-k2.7-code", "responses") {
		t.Fatal("model capability invented")
	}
	if Supports("kimi-k2.7-code-highspeed", "messages") {
		t.Fatal("unconfirmed high-speed Messages enabled")
	}
	for _, body := range []string{`{}`, `{"data":[{"id":"kimi-k3","context_length":-1}]}`, `{"data":[{"id":"invalid model"}]}`, `{"data":[{"id":"kimi-k2.5"}]}`} {
		client := New(roundTrip(func(*http.Request) (*http.Response, error) { return reply(200, body), nil }))
		if _, err := client.DiscoverModels(context.Background(), Options{Region: "cn"}, []byte("placeholder-kimi-key")); err == nil {
			t.Fatal("invalid catalog accepted")
		}
	}
}
