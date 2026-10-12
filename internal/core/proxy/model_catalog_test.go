package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestModelCatalogFormatsAuthenticationAndQueries(t *testing.T) {
	server := New(Options{Models: []string{"a-model", "b-model"}, Authorize: func(_ context.Context, key string) bool { return key == "qa-key" }})
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
