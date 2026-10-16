package antigravity

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestAgentDiscoveryRequiresPickerForThirdPartyModels(t *testing.T) {
	body := `{"models":{"gemini-current":{},"gemini-legacy":{},"claude-opus-4-6-thinking":{},"claude-sonnet-4-6":{},"gpt-oss-120b-medium":{},"claude-future":{},"gemini-internal":{"isInternal":true}},"agentModelSorts":[{"groups":[{"modelIds":["gemini-current","claude-opus-4-6-thinking","claude-sonnet-4-6","gpt-oss-120b-medium","claude-future","gemini-internal"]}]}]}`
	client := New(transportFunc(func(*http.Request) (*http.Response, error) { return reply(200, body), nil }))
	models, err := client.Models(context.Background(), testCredentials())
	if err != nil || len(models) != 5 {
		t.Fatal("supported models or legacy Gemini missing", err)
	}
	for _, model := range models {
		if len(model.Metadata.InputModalities) != 1 || model.Metadata.InputModalities[0] != "text" || model.Metadata.ContextWindow != nil {
			t.Fatal("fabricated model capability")
		}
	}
	picker, err := client.AccountModels(context.Background(), testCredentials())
	if err != nil || len(picker) != 5 {
		t.Fatal("picker changed or included internal entry", err)
	}
	for _, model := range picker {
		if model.GatewaySupported != (model.ID != "claude-future") {
			t.Fatal("adapter status inconsistent", model.ID)
		}
	}
	body = `{"models":{"gemini-legacy":{},"claude-opus-4-6-thinking":{}}}`
	models, err = client.Models(context.Background(), testCredentials())
	if err != nil || len(models) != 1 || models[0].ID != "gemini-legacy" {
		t.Fatal("third-party model published without official picker")
	}
}

func TestToolSignatureCacheIsolationExpiryAndBounds(t *testing.T) {
	client := New(nil)
	cred := testCredentials()
	args := object{"city": "Paris"}
	call := object{"id": "call_a", "function": object{"name": "weather", "arguments": `{"city":"Paris"}`}, "extra_content": object{"google": object{"thought_signature": "opaque-signature"}}}
	client.rememberSignatures(cred, "gemini-current", []object{call})
	body := []byte(`{"project":"test-project","request":{"contents":[{"role":"model","parts":[{"functionCall":{"id":"call_a","name":"weather","args":{"city":"Paris"}}}]}]}}`)
	for _, mode := range []string{"same", "subject", "project", "authorization", "model", "arguments", "expired"} {
		current, model, input := cred, "gemini-current", body
		switch mode {
		case "subject":
			current.Subject = "other"
		case "project":
			current.Project = "other"
		case "authorization":
			current.RefreshToken = "other"
		case "model":
			model = "other"
		case "arguments":
			input = []byte(strings.ReplaceAll(string(body), "Paris", "London"))
		case "expired":
			key := signatureKey(cred, model, "call_a", "weather", args)
			client.signatures.entries[key] = signatureEntry{"opaque-signature", time.Now().Add(-time.Second)}
		}
		out, err := client.restoreSignatures(input, current, model)
		if err != nil || strings.Contains(string(out), "opaque-signature") != (mode == "same") {
			t.Fatal("signature isolation failed", mode, err)
		}
	}
	for i := range signatureLimit + 20 {
		call["id"] = strings.Repeat("x", i+1)
		client.rememberSignatures(cred, "gemini-current", []object{call})
	}
	if len(client.signatures.entries) != signatureLimit {
		t.Fatal("unbounded signature cache")
	}
}

func TestOtherFinishRequiresCompleteClaudeToolCall(t *testing.T) {
	for _, model := range []string{"claude-sonnet-4-6", "gemini-current"} {
		for _, tool := range []bool{false, true} {
			parts := `{"text":"incomplete"}`
			if tool {
				parts = `{"functionCall":{"name":"weather","args":{"city":"Paris"}}}`
			}
			state := responseState{model: model}
			_, err := state.accept([]byte(`{"candidates":[{"content":{"parts":[` + parts + `]},"finishReason":"OTHER"}]}`))
			if (err == nil) != (tool && model == "claude-sonnet-4-6") {
				t.Fatal("invalid OTHER terminal handling", model, tool)
			}
		}
	}
}
