package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

type catalogUpstream struct{ entries []catalog.Entry }

func (catalogUpstream) Forward(context.Context, Request, func(Response) error) error { return nil }
func (c catalogUpstream) Catalog(context.Context) ([]catalog.Entry, error) {
	return c.entries, nil
}

func TestAntigravityCatalogFormatPublishesAllGoogleModels(t *testing.T) {
	server := New(Options{
		Upstream:  catalogUpstream{entries: []catalog.Entry{{ID: "gemini-3.8-flash", Name: "Gemini", ProviderKind: "google"}, {ID: "claude-opus-4-6-thinking", Name: "Claude", ProviderKind: "google"}, {ID: "gpt-oss-120b-medium", Name: "GPT-OSS", ProviderKind: "google"}, {ID: "gpt-6.1-sol", Name: "GPT", ProviderKind: "openai"}}},
		Authorize: func(_ context.Context, key string) (string, bool) { return "antigravity", key == "qa-key" },
	})
	request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:0/v1/models?format=antigravity", nil)
	request.RemoteAddr = "127.0.0.1:12345"
	// 桌面端的远端目录可能按 Google SDK 习惯带这个头，网关只把它当本地接入 Key。
	request.Header.Set("X-Goog-Api-Key", "qa-key")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status %d: %s", response.Code, response.Body.String())
	}
	var body struct {
		List []catalog.AntigravityModel `json:"list"`
	}
	if json.Unmarshal(response.Body.Bytes(), &body) != nil || len(body.List) != 3 {
		t.Fatalf("unexpected Antigravity catalog: %s", response.Body.String())
	}
	found := map[string]bool{}
	for _, model := range body.List {
		found[model.ModelID] = true
	}
	for _, want := range []string{"gemini-3.8-flash", "claude-opus-4-6-thinking", "gpt-oss-120b-medium"} {
		if !found[want] {
			t.Fatalf("model %q missing from catalog: %s", want, response.Body.String())
		}
	}
}

func TestModelCatalogFormatsAuthenticationAndQueries(t *testing.T) {
	server := New(Options{Models: []string{"a-model", "b-model"}, Authorize: func(_ context.Context, key string) (string, bool) { return "", key == "qa-key" }})
	for _, test := range []struct {
		path, key string
		status    int
		codex     bool
	}{
		{"/v1/codex-models?client_version=0.157.1", "qa-key", 200, true},
		{"/v1/models?limit=1", "qa-key", 200, false},
		{"/v1/models?limit=1000", "", 401, false},
		{"/v1/codex-models?client_version=a&client_version=b", "qa-key", 400, false},
		{"/v1/codex-models?client_version=%0A", "qa-key", 400, false},
		{"/v1/models?limit=1001", "qa-key", 400, false},
		{"/v1/models?redirect=https://external.invalid", "qa-key", 400, false},
		{"/v1/responses?client_version=0.157.1", "qa-key", 400, false},
	} {
		t.Run(test.path+test.key, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:0"+test.path, nil)
			request.RemoteAddr = "127.0.0.1:12345"
			request.Header.Set("Authorization", "Bearer "+test.key)
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status %d, expected %d", response.Code, test.status)
			}
			if test.status != 200 {
				return
			}
			var body map[string]json.RawMessage
			if json.Unmarshal(response.Body.Bytes(), &body) != nil {
				t.Fatal("invalid JSON")
			}
			if test.codex {
				if body["models"] == nil || body["data"] != nil || !strings.Contains(string(body["models"]), "a-model") {
					t.Fatal("wrong Codex schema")
				}
			} else if body["data"] == nil || string(body["has_more"]) != "true" {
				t.Fatal("ordinary model pagination changed")
			}
		})
	}
}
