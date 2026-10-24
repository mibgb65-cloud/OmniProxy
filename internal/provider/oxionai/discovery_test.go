package oxionai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDiscoverModelsKeepsChatModelsAndConfirmedCapabilities(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer cc_test_key" {
			t.Errorf("Authorization = %q", got)
		}
		io.WriteString(w, `{"object":"list","data":[
			{"id":"gemma-2-2b","name":"Gemma 2 2B","type":"chat","context_window":128000,"capabilities":["reasoning","vision","tools","streaming"]},
			{"id":"plain-chat","name":"","type":"chat","context_window":0,"capabilities":[]},
			{"id":"text-embedding-3-small","type":"embedding","context_window":8191},
			{"id":"gemma-2-2b","name":"dup","type":"chat"}]}`)
	}))
	defer upstream.Close()

	models, err := New(nil).DiscoverModels(context.Background(), upstream.URL+"/v1", []byte("cc_test_key"))
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 {
		t.Fatalf("models = %d, want 2", len(models))
	}
	first, second := models[0], models[1]
	if first.ID != "gemma-2-2b" || first.Name != "Gemma 2 2B" {
		t.Fatalf("unexpected model %+v", first)
	}
	if first.Metadata.ContextWindow == nil || *first.Metadata.ContextWindow != 128000 {
		t.Fatalf("context window = %v", first.Metadata.ContextWindow)
	}
	if strings.Join(first.Metadata.InputModalities, ",") != "text,image" || first.Metadata.ToolUse == nil || !*first.Metadata.ToolUse {
		t.Fatalf("capabilities = %+v", first.Metadata)
	}
	if second.Name != "plain-chat" || second.Metadata.ContextWindow != nil || second.Metadata.ToolUse != nil || second.Metadata.InputModalities != nil {
		t.Fatalf("unknown capabilities must stay unknown: %+v", second)
	}
}

func TestDiscoverModelsRejectsUnsafeCatalogs(t *testing.T) {
	cases := map[string]string{
		"空目录":     `{"object":"list","data":[]}`,
		"缺少 data": `{"object":"list"}`,
		"非法模型 ID": `{"data":[{"id":"bad id with space","type":"chat"}]}`,
		"只有嵌入模型":  `{"data":[{"id":"text-embedding-3-small","type":"embedding"}]}`,
		"不是 JSON": `not-json`,
	}
	for name, payload := range cases {
		t.Run(name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				io.WriteString(w, payload)
			}))
			defer upstream.Close()
			if _, err := New(nil).DiscoverModels(context.Background(), upstream.URL+"/v1", []byte("cc_test_key")); err == nil {
				t.Fatal("unsafe catalog must be rejected")
			}
		})
	}
}

func TestDiscoverModelsFailsOnUpstreamError(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, `{"error":{"code":"invalid_api_key"}}`)
	}))
	defer upstream.Close()
	if _, err := New(nil).DiscoverModels(context.Background(), upstream.URL+"/v1", []byte("cc_test_key")); err == nil {
		t.Fatal("401 must fail discovery")
	}
}
