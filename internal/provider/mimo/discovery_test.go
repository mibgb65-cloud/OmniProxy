package mimo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

func TestModelDiscoveryFiltersSpeechAndUnverifiedModels(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/models" || r.Header.Get("Api-Key") != "test-key" {
			t.Error("invalid discovery contract")
		}
		fmt.Fprint(w, `{"object":"list","data":[{"id":"mimo-v2.6-pro"},{"id":"mimo-v2.6-flash"},{"id":"mimo-v2.6-pro"},{"id":"mimo-v2.5-asr"},{"id":"mimo-v2.5-tts"},{"id":"mimo-future"}]}`)
	}))
	defer upstream.Close()
	models, err := New(nil).DiscoverModels(context.Background(), upstream.URL+"/v1", []byte("test-key"))
	if err != nil || strings.Join(models, ",") != "mimo-v2.6-pro,mimo-v2.6-flash" {
		t.Fatal("unsafe discovery catalog", models, err)
	}
	for _, model := range Models() {
		metadata := ModelMetadata(model)
		if catalog.ValidateMetadata(metadata) != nil || metadata.ContextWindow == nil || *metadata.ContextWindow != 1000000 || metadata.OutputTokenLimit == nil || *metadata.OutputTokenLimit != 131072 || metadata.ToolUse == nil || !*metadata.ToolUse || metadata.ContextVerified {
			t.Fatal("invalid or fabricated metadata", model)
		}
	}
}

func TestDiscoveryRejectsMalformedAndOversizedCatalogs(t *testing.T) {
	for _, body := range []string{`{}`, `{"data":null}`, `{"data":[]}`, `{"data":[{"id":"mimo-v2.5-tts"}]}`, `{"data":[{"id":"bad\nmodel"}]}`, strings.Repeat("x", (128<<10)+1)} {
		t.Run(fmt.Sprint(len(body)), func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
			defer upstream.Close()
			if _, err := New(nil).DiscoverModels(context.Background(), upstream.URL, []byte("test-key")); err == nil {
				t.Fatal("invalid catalog accepted")
			}
		})
	}
}
