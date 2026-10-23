package agentconfig

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

func TestQwenModelProvidersLifecycle(t *testing.T) {
	s, source := gatewayClientFixture(t)
	ctx := context.Background()
	original := `// original settings
{"$version":4,"modelProviders":{"openai":[{"id":"deepseek-flash","baseUrl":"https://existing.invalid/v1","envKey":"EXISTING_KEY"}],"anthropic":[{"id":"other"}]},"env":{"EXISTING_KEY":"test-placeholder"},"security":{"auth":{"selectedType":"qwen-oauth"},"other":true},"model":{"name":"old","other":true},"permissions":{"allow":["read"]}}`
	writeGatewayFixture(t, s.paths.QwenCode, original)
	oauth := filepath.Join(filepath.Dir(s.paths.QwenCode), "oauth_creds.json")
	writeGatewayFixture(t, oauth, "original-oauth-placeholder")
	before, _ := s.snapshotClient("qwen-code", s.paths.QwenCode)
	input := Input{Client: "qwen-code", Revision: before.revision}
	plan, err := s.PreviewPlan(ctx, input)
	if err != nil || len(plan.Models) != 1 || strings.Contains(plan.Summary, "test-placeholder") {
		t.Fatal("unsafe preview", err)
	}
	if _, err := s.Apply(ctx, input); err != nil {
		t.Fatal(err)
	}
	key, err := s.Key(ctx, "qwen-code")
	defer clear(key)
	if err != nil || !s.Authorize(ctx, string(key)) {
		t.Fatal("key not authorized", err)
	}
	source.entries = append(source.entries, catalog.Entry{ID: "new-chat", Protocols: []string{"chat_completions"}})
	s.port = 4099
	if err := s.RefreshCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(s.paths.QwenCode)
	var config map[string]any
	if json.Unmarshal(body, &config) != nil {
		t.Fatal("Qwen needs standard JSON without trailing commas")
	}
	providers := config["modelProviders"].(map[string]any)
	models := providers["openai"].([]any)
	if len(models) != 3 || len(providers["anthropic"].([]any)) != 1 || config["permissions"] == nil || strings.Contains(string(body), "only-responses") {
		t.Fatal("provider or settings preservation failed")
	}
	for _, raw := range models[:2] {
		entry := raw.(map[string]any)
		if entry["envKey"] != qwenCredentialRef || entry["baseUrl"] != "http://127.0.0.1:4099/qwen-code/v1" || entry["wireApi"] != "chat-completions" {
			t.Fatal("managed models not refreshed")
		}
	}
	if config["env"].(map[string]any)[qwenCredentialRef] != string(key) || config["security"].(map[string]any)["auth"].(map[string]any)["selectedType"] != "openai" {
		t.Fatal("dedicated credential or auth selection lost")
	}
	if _, err := s.Restore(ctx, "qwen-code"); err != nil {
		t.Fatal(err)
	}
	restored, _ := os.ReadFile(s.paths.QwenCode)
	login, _ := os.ReadFile(oauth)
	if string(restored) != original || string(login) != "original-oauth-placeholder" || s.Authorize(ctx, string(key)) {
		t.Fatal("restore changed original content, login or authorization")
	}
}

func TestCodingClientsIsolateKeysAndRejectReservedSlots(t *testing.T) {
	s, _ := gatewayClientFixture(t)
	ctx := context.Background()
	keys := map[string]bool{}
	for _, client := range []string{"opencode", "kilocode", "qwen-code", "mimo-code"} {
		path, _ := s.path(client)
		snapshot, _ := s.snapshotClient(client, path)
		if _, err := s.Apply(ctx, Input{Client: client, Revision: snapshot.revision}); err != nil {
			t.Fatal(client, err)
		}
		key, _ := s.Key(ctx, client)
		if keys[string(key)] || !s.Authorize(ctx, string(key)) {
			t.Fatal("clients share a key or new key not whitelisted", client)
		}
		keys[string(key)] = true
		clear(key)
	}
	for _, client := range []string{"kilocode", "mimo-code"} {
		if _, _, err := buildCompatibleClient(client, []byte(`{"provider":{"omniproxy":{"options":{"baseURL":"https://existing.invalid"}}}}`), nil, 3001, Plan{}, false); err == nil {
			t.Fatal("unowned provider overwritten", client)
		}
	}
	for _, body := range []string{`{"$version":4,"env":{"OMNIPROXY_QWEN_CODE_KEY":"existing"}}`, `{"$version":4,"modelProviders":{"openai":[{"envKey":"OMNIPROXY_QWEN_CODE_KEY"}]}}`, `{"$version":4,"modelProviders":{"openai":{}}}`, `{"$version":4,"security":[]}`} {
		if _, _, err := buildQwenCode([]byte(body), nil, 3001, Plan{}, false); err == nil {
			t.Fatal("reserved or malformed Qwen configuration overwritten")
		}
	}
}

func TestQwenRefusesUnmigratedOrFutureSettings(t *testing.T) {
	for _, body := range []string{`{"authType":"openai"}`, `{"$version":2,"model":{}}`, `{"$version":5}`} {
		if _, _, err := buildQwenCode([]byte(body), nil, 3001, Plan{}, false); err == nil || err.Error() != "agent: client_upgrade_required" {
			t.Fatal("unknown settings version skipped official migration", err)
		}
	}
}
