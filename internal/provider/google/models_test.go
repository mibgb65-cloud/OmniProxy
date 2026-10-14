package google

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestModelsPaginationAndIndependentLimits(t *testing.T) {
	pages := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1beta/openai/models" {
			if r.Header.Get("Authorization") != "Bearer test-key" {
				t.Error("compatibility discovery auth missing")
			}
			fmt.Fprint(w, `{"data":[{"id":"models/gemini-test"},{"id":"gemini-other"}]}`)
			return
		}
		pages++
		if r.URL.Path != "/v1beta/models" || r.Header.Get("x-goog-api-key") != "test-key" || r.URL.Query().Has("key") || r.URL.Query().Get("pageSize") != "100" {
			t.Error("discovery transport incorrect")
		}
		if pages == 1 {
			fmt.Fprint(w, `{"models":[{"name":"models/gemini-test","displayName":"Gemini Test","inputTokenLimit":1048576,"outputTokenLimit":65536,"supportedGenerationMethods":["generateContent"]},{"name":"models/embed-test","supportedGenerationMethods":["embedContent"]}],"nextPageToken":"next"}`)
		} else {
			if r.URL.Query().Get("pageToken") != "next" {
				t.Error("cursor missing")
			}
			fmt.Fprint(w, `{"models":[{"name":"models/gemini-test","supportedGenerationMethods":["generateContent"]},{"name":"models/gemini-other","supportedGenerationMethods":["generateContent"]}]}`)
		}
	}))
	defer upstream.Close()
	models, err := New(nil).Models(context.Background(), upstream.URL+"/v1beta", []byte("test-key"))
	if err != nil || len(models) != 2 || pages != 2 {
		t.Fatal("pagination/filter/dedup failed", err)
	}
	meta := models[0].Metadata
	if meta.ContextWindow != nil || *meta.InputTokenLimit != 1048576 || *meta.OutputTokenLimit != 65536 {
		t.Fatal("limits guessed or lost")
	}
}

func TestInvalidDiscoveryBounds(t *testing.T) {
	for _, body := range []string{`{"models":[{"name":"models/gemini-test","inputTokenLimit":-1,"supportedGenerationMethods":["generateContent"]}]}`, `{"models":[{"name":"models/../bad","supportedGenerationMethods":["generateContent"]}]}`, `{"models":[],"nextPageToken":"repeat"}`, `{"models":null}`} {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/openai/models" {
				fmt.Fprint(w, `{"data":[{"id":"gemini-test"}]}`)
				return
			}
			fmt.Fprint(w, body)
		}))
		_, err := New(nil).Models(context.Background(), upstream.URL, []byte("test-key"))
		upstream.Close()
		if err == nil {
			t.Error("invalid discovery response accepted")
		}
	}
}
