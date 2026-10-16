package antigravity

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestAccountModelsUseOfficialPickerAndMarkGatewaySupport(t *testing.T) {
	const body = `{"models":{"gemini-current":{"displayName":"Gemini Current"},"gemini-old":{"displayName":"Legacy alias"},"claude-current":{"displayName":"Claude Current"},"internal":{"isInternal":true}},"agentModelSorts":[{"groups":[{"modelIds":["claude-current","gemini-current","internal","gemini-current"]}]}],"email":"private@example.test","access_token":"private-token"}`
	client := New(transportFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path != "/v1internal:fetchAvailableModels" || r.Header.Get("Authorization") != "Bearer test-access" {
			t.Error("wrong account model endpoint")
		}
		return reply(200, body), nil
	}))
	models, err := client.AccountModels(context.Background(), testCredentials())
	if err != nil || len(models) != 2 || models[0].ID != "gemini-current" || !models[0].GatewaySupported || models[1].GatewaySupported {
		t.Fatal("picker did not filter internal, duplicate and legacy entries")
	}
	encoded, _ := json.Marshal(models)
	for _, private := range []string{"private-token", "private@example.test", "test-project", "Legacy alias"} {
		if strings.Contains(string(encoded), private) {
			t.Fatal("unpublished data escaped model selection")
		}
	}
}

func TestAccountModelsRejectUnknownPickerStructure(t *testing.T) {
	for _, body := range []string{`{"models":{"gemini-test":{}}}`, `{"models":{},"agentModelSorts":[{"groups":[{"modelIds":["missing"]}]}]}`, `{"models":null,"agentModelSorts":[]}`, `{"models":{"bad model":{}},"agentModelSorts":[{"groups":[{"modelIds":["bad model"]}]}]}`} {
		client := New(transportFunc(func(*http.Request) (*http.Response, error) { return reply(200, body), nil }))
		if _, err := client.AccountModels(context.Background(), testCredentials()); err == nil {
			t.Fatal("invalid picker accepted")
		}
	}
}
