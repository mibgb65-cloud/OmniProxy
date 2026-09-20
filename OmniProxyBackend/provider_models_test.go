package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"omniproxy/internal/token"
)

func TestParseProviderCatalogModelsSupportsOpenAIAndGeminiShapes(t *testing.T) {
	body := []byte(`{
		"data": [
			{"id": "gpt-test", "name": "GPT Test", "context_length": 128000},
			{"id": ""}
		]
	}`)
	models, err := parseProviderCatalogModels(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].ID != "gpt-test" || models[0].ContextLength != 128000 {
		t.Fatalf("unexpected OpenAI-compatible models: %#v", models)
	}

	body = []byte(`{
		"models": [
			{"name": "models/gemini-test", "displayName": "Gemini Test", "inputTokenLimit": 1048576}
		]
	}`)
	models, err = parseProviderCatalogModels(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].ID != "gemini-test" || models[0].Name != "Gemini Test" {
		t.Fatalf("unexpected Gemini models: %#v", models)
	}
}

func TestFetchFeatherlessCatalogModelsUsesPlanAndActiveFilters(t *testing.T) {
	var queryValues map[string]string
	var authorization string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queryValues = map[string]string{
			"available_on_current_plan": r.URL.Query().Get("available_on_current_plan"),
			"status":                    r.URL.Query().Get("status"),
			"sort":                      r.URL.Query().Get("sort"),
			"per_page":                  r.URL.Query().Get("per_page"),
		}
		authorization = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"data":[{"id":"Qwen/Qwen3-32B","name":"Qwen 3 32B","context_length":32768}]}`))
	}))
	defer upstream.Close()

	models, err := fetchOpenAICompatibleCatalogModels(
		context.Background(),
		upstream.Client(),
		token.ProviderFeatherless,
		upstream.URL+"/v1",
		"featherless-api-key-token",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 || models[0].ID != "Qwen/Qwen3-32B" || models[0].ContextLength != 32768 {
		t.Fatalf("unexpected Featherless models: %#v", models)
	}
	if authorization != "Bearer featherless-api-key-token" {
		t.Fatalf("unexpected authorization: %q", authorization)
	}
	if queryValues["available_on_current_plan"] != "true" || queryValues["status"] != "active" || queryValues["sort"] != "-popularity" || queryValues["per_page"] != "1000" {
		t.Fatalf("unexpected Featherless model filters: %#v", queryValues)
	}
}
