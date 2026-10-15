package minimax

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestDiscoveryFiltersModelsAndPreviewByKeyType(t *testing.T) {
	for _, mode := range []string{"payg", "subscription"} {
		c := New(roundTrip(func(r *http.Request) (*http.Response, error) {
			if r.URL.String() != "https://api.minimax.cn/v1/models" {
				t.Fatal("wrong discovery region")
			}
			return reply(200, `{"data":[{"id":"MiniMax-M3"},{"id":"MiniMax-M3"},{"id":"MiniMax-M2.7-highspeed"},{"id":"speech-2.8-hd"},{"id":"MiniMax-M3.1-Flash-Preview"}]}`), nil
		}))
		result, err := c.DiscoverModels(context.Background(), Options{Region: "cn", BillingMode: mode}, []byte("placeholder"))
		expected := 2
		if mode == "subscription" {
			expected = 3
		}
		if err != nil || len(result) != expected || result[0].Metadata.ContextWindow == nil || *result[0].Metadata.ContextWindow != 1000000 {
			t.Fatal("invalid catalog", result, err)
		}
	}
	for _, body := range []string{`{}`, `{"data":null}`, `{"data":[{"id":"invalid model"}]}`, `{"data":[{"id":"MiniMax-M3","context_length":-1}]}`, `{"data":[{"id":"speech-hd"}]}`, strings.Repeat("x", 131073)} {
		c := New(roundTrip(func(*http.Request) (*http.Response, error) { return reply(200, body), nil }))
		if _, err := c.DiscoverModels(context.Background(), Options{Region: "cn", BillingMode: "payg"}, []byte("placeholder")); err == nil {
			t.Fatal("untrusted catalog accepted")
		}
	}
}
