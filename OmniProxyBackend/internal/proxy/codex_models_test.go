package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func codexCatalogFixture(t *testing.T) []byte {
	t.Helper()
	return []byte(`{
  "fetched_at": 1750000000,
  "etag": "cache-etag-1",
  "client_version": "0.155.0",
  "identity": {"account_id": "acc-1"},
  "models": [
    {
      "slug": "gpt-5.6-sol",
      "display_name": "GPT-5.6-Sol",
      "description": "Reliable agentic workhorse for everyday tasks.",
      "default_reasoning_level": "medium",
      "supported_reasoning_levels": [{"effort": "low", "description": "fast"}],
      "shell_type": "unified_exec",
      "visibility": "list",
      "supported_in_api": true,
      "priority": 0,
      "service_tiers": [{"id": "priority", "name": "Fast"}],
      "model_messages": {"instructions_template": "You are Codex, an agent based on GPT-5."},
      "support_verbosity": true,
      "truncation_policy": {"mode": "tokens", "limit": 10000},
      "context_window": 272000,
      "max_context_window": 872000,
      "experimental_supported_tools": []
    }
  ]
}`)
}

func TestInjectCodexModelsAppendsMissingModel(t *testing.T) {
	out, added := injectCodexModels(codexCatalogFixture(t), []codexModelProfile{
		{Model: "Atria-Dawn-Preview", ContextWindow: 256000},
	})
	if added != 1 {
		t.Fatalf("added = %d, want 1", added)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("injected body is not valid JSON: %v", err)
	}
	models := doc["models"].([]any)
	if len(models) != 2 {
		t.Fatalf("models = %d, want 2", len(models))
	}
	entry := models[1].(map[string]any)
	if entry["slug"] != "Atria-Dawn-Preview" {
		t.Errorf("slug = %v", entry["slug"])
	}
	if entry["display_name"] != "Atria Dawn Preview" {
		t.Errorf("display_name = %v", entry["display_name"])
	}
	if entry["base_instructions"] != codexInjectedBaseInstructions {
		t.Errorf("base_instructions = %v", entry["base_instructions"])
	}
	if entry["context_window"] != float64(256000) {
		t.Errorf("context_window = %v", entry["context_window"])
	}
	if entry["max_context_window"] != float64(256000) {
		t.Errorf("max_context_window = %v", entry["max_context_window"])
	}
	if entry["visibility"] != "list" {
		t.Errorf("visibility = %v", entry["visibility"])
	}

	// Untouched catalog fields must survive the round trip.
	if doc["etag"] != "cache-etag-1" {
		t.Errorf("etag = %v", doc["etag"])
	}
	if doc["fetched_at"] != float64(1750000000) {
		t.Errorf("fetched_at = %v", doc["fetched_at"])
	}
	if ident, ok := doc["identity"].(map[string]any); !ok || ident["account_id"] != "acc-1" {
		t.Errorf("identity = %v", doc["identity"])
	}
}

func TestInjectCodexModelsSkipsExistingSlugs(t *testing.T) {
	profiles := []codexModelProfile{
		{Model: "GPT-5.6-Sol", ContextWindow: 272000},
		{Model: "Atria-Dawn-Preview", ContextWindow: 256000},
	}
	out, added := injectCodexModels(codexCatalogFixture(t), profiles)
	if added != 1 {
		t.Fatalf("added = %d, want 1 (existing slug must not be re-added)", added)
	}
	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("injected body is not valid JSON: %v", err)
	}
	for _, item := range doc["models"].([]any) {
		if item.(map[string]any)["slug"] == "GPT-5.6-Sol" {
			t.Fatal("existing slug was replaced instead of skipped")
		}
	}
}

func TestInjectCodexModelsFailsOpen(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		profiles []codexModelProfile
	}{
		{"malformed json", `{not json`, []codexModelProfile{{Model: "Atria-Dawn-Preview"}}},
		{"missing models key", `{"etag": "x"}`, []codexModelProfile{{Model: "Atria-Dawn-Preview"}}},
		{"models not an array", `{"models": "nope"}`, []codexModelProfile{{Model: "Atria-Dawn-Preview"}}},
		{"no profiles", `{"models": []}`, nil},
		{"empty profile model", `{"models": []}`, []codexModelProfile{{Model: "  "}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, added := injectCodexModels([]byte(tc.raw), tc.profiles)
			if added != 0 {
				t.Fatalf("added = %d, want 0", added)
			}
			if string(out) != tc.raw {
				t.Fatalf("body was modified: %q", string(out))
			}
		})
	}
}

func TestCodexModelDisplayName(t *testing.T) {
	cases := map[string]string{
		"Atria-Dawn-Preview": "Atria Dawn Preview",
		"deepseek-v4-flash":  "DeepSeek V4 Flash",
		"mimo-v2.5-pro[1m]":  "MiMo V2.5",
		"kimi-for-coding":    "Kimi for Coding",
		"custom-llama-3":     "Custom Llama 3",
	}
	for model, want := range cases {
		if got := codexModelDisplayName(model); got != want {
			t.Errorf("codexModelDisplayName(%q) = %q, want %q", model, got, want)
		}
	}
}

func TestParseCodexModelProfile(t *testing.T) {
	raw := `# omniproxy managed profile
model = "Atria-Dawn-Preview"
review_model = "Atria-Dawn-Preview"
model_provider = "openai"
model_reasoning_effort = "xhigh"
model_context_window = 256000
`
	profile := parseCodexModelProfile(raw)
	if profile.Model != "Atria-Dawn-Preview" {
		t.Errorf("Model = %q", profile.Model)
	}
	if profile.ContextWindow != 256000 {
		t.Errorf("ContextWindow = %d", profile.ContextWindow)
	}
}

func TestCodexModelProfilesInDir(t *testing.T) {
	dir := t.TempDir()
	profiles := map[string]string{
		"omniproxy-atria-dawn-preview.config.toml": "model = \"Atria-Dawn-Preview\"\nmodel_context_window = 256000\n",
		"omniproxy-deepseek-v4-pro.config.toml":    "model = \"deepseek-v4-pro\"\n",
		"omniproxy-incomplete.config.toml":         "# no model line\nmodel_provider = \"openai\"\n",
	}
	for name, content := range profiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	loaded, err := codexModelProfilesInDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 2 {
		t.Fatalf("loaded %d profiles, want 2 (incomplete profile must be skipped)", len(loaded))
	}
	byModel := map[string]int{}
	for _, profile := range loaded {
		byModel[profile.Model] = profile.ContextWindow
	}
	if byModel["Atria-Dawn-Preview"] != 256000 {
		t.Errorf("atria context window = %d", byModel["Atria-Dawn-Preview"])
	}
	if _, ok := byModel["deepseek-v4-pro"]; !ok {
		t.Error("deepseek profile missing")
	}
}

func TestIsCodexModelsListRequest(t *testing.T) {
	cases := []struct {
		method string
		path   string
		want   bool
	}{
		{http.MethodGet, "/codex/v1/models", true},
		{http.MethodGet, "/codex/v1/models?client_version=0.155.0", true},
		{http.MethodGet, "/backend-api/codex/v1/models", true},
		{http.MethodGet, "/codex/models", true},
		{http.MethodPost, "/codex/v1/models", false},
		{http.MethodGet, "/codex/v1/responses", false},
		{http.MethodGet, "/codex/v1/models/extra", false},
		{http.MethodGet, "/anthropic-router/v1/messages", false},
	}
	for _, tc := range cases {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if got := isCodexModelsListRequest(req); got != tc.want {
				t.Fatalf("isCodexModelsListRequest = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestApplyCodexModelsInjectionRewritesResponse(t *testing.T) {
	raw := codexCatalogFixture(t)
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		Header:        http.Header{"Content-Encoding": []string{"gzip"}, "Content-Length": []string{strconv.Itoa(len(raw))}},
		Body:          bodyOf(raw),
		ContentLength: int64(len(raw)),
	}
	profiles := []codexModelProfile{{Model: "Atria-Dawn-Preview", ContextWindow: 256000}}

	if added := applyCodexModelsInjection(resp, profiles); added != 1 {
		t.Fatalf("added = %d, want 1", added)
	}
	if resp.Header.Get("Content-Encoding") != "" {
		t.Error("Content-Encoding header was not cleared")
	}
	body := readAll(t, resp.Body)
	if got, want := resp.Header.Get("Content-Length"), strconv.Itoa(len(body)); got != want {
		t.Errorf("Content-Length = %s, want %s", got, want)
	}
	if resp.ContentLength != int64(len(body)) {
		t.Errorf("ContentLength = %d, want %d", resp.ContentLength, len(body))
	}
	if !strings.Contains(string(body), "Atria-Dawn-Preview") {
		t.Error("response body does not contain the injected model")
	}
	if strings.Contains(string(body), "You are Codex, an agent based on GPT-5") == false {
		t.Error("upstream model entry was dropped from the body")
	}
}

func TestApplyCodexModelsInjectionLeavesOtherResponsesAlone(t *testing.T) {
	raw := []byte(`{"error": "no active token available"}`)
	resp := &http.Response{
		StatusCode:    http.StatusServiceUnavailable,
		Header:        http.Header{},
		Body:          bodyOf(raw),
		ContentLength: int64(len(raw)),
	}
	if added := applyCodexModelsInjection(resp, []codexModelProfile{{Model: "Atria-Dawn-Preview"}}); added != 0 {
		t.Fatalf("added = %d, want 0 for non-200 response", added)
	}
	if got := readAll(t, resp.Body); string(got) != string(raw) {
		t.Fatalf("body changed: %q", string(got))
	}

	// Chunked (unknown length) responses must be passed through untouched.
	chunked := &http.Response{
		StatusCode:    http.StatusOK,
		Header:        http.Header{},
		Body:          bodyOf(raw),
		ContentLength: -1,
	}
	if added := applyCodexModelsInjection(chunked, []codexModelProfile{{Model: "Atria-Dawn-Preview"}}); added != 0 {
		t.Fatalf("added = %d, want 0 for chunked response", added)
	}
	if got := readAll(t, chunked.Body); string(got) != string(raw) {
		t.Fatalf("body changed: %q", string(got))
	}
}

func TestRewriteCodexModelsResponseDoesNotConsumeUnknownLengthBody(t *testing.T) {
	s := &Service{}
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		Header:        http.Header{},
		Body:          bodyOf(codexCatalogFixture(t)),
		ContentLength: 0,
	}
	req := httptest.NewRequest(http.MethodGet, "/codex/v1/models", nil)
	s.rewriteCodexModelsResponse(req, resp)
	if got := readAll(t, resp.Body); !strings.Contains(string(got), "gpt-5.6-sol") {
		t.Fatal("response body was consumed or replaced when it cannot be injected")
	}
}

func bodyOf(raw []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(raw))
}

func readAll(t *testing.T, body io.Reader) []byte {
	t.Helper()
	raw, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
