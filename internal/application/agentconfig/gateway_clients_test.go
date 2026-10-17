package agentconfig

import (
	"context"
	"encoding/json"
	catalog "omniproxy/internal/core/modelcatalog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func gatewayClientFixture(t *testing.T) (*Service, *changingCatalog) {
	t.Helper()
	s, source := catalogFixture(t)
	root := filepath.Dir(filepath.Dir(s.paths.Codex))
	s.paths.OpenCode = filepath.Join(root, "opencode", "opencode.jsonc")
	s.paths.KiloCode = filepath.Join(root, "kilo", "kilo.jsonc")
	s.paths.QwenCode = filepath.Join(root, "qwen", "settings.json")
	s.paths.MiMoCode = filepath.Join(root, "mimo", "mimocode.jsonc")
	s.paths.HarnessWeb = filepath.Join(root, "harness", "profiles", "web", "cordis.patch.yml")
	s.paths.HarnessDesktop = filepath.Join(root, "harness", "profiles", "desktop", "cordis.patch.yml")
	s.paths.HarnessCredentials = filepath.Join(root, "harness", ".credentials.yaml")
	source.entries = []catalog.Entry{{ID: "deepseek-flash", Name: "Flash", Protocols: []string{"chat_completions"}}, {ID: "only-responses", Protocols: []string{"responses"}}}
	return s, source
}

func writeGatewayFixture(t *testing.T, path, body string) {
	t.Helper()
	if os.MkdirAll(filepath.Dir(path), 0700) != nil || os.WriteFile(path, []byte(body), 0600) != nil {
		t.Fatal("fixture write failed")
	}
}

func TestGatewayClientsPreviewApplyRefreshAndRestore(t *testing.T) {
	for _, client := range []string{"opencode", "kilocode", "mimo-code", "harness-web", "harness-desktop"} {
		t.Run(client, func(t *testing.T) {
			s, source := gatewayClientFixture(t)
			ctx := context.Background()
			path, _ := s.path(client)
			original := "// preserve comment\n{\"permission\":{\"bash\":\"ask\"},\"provider\":{\"existing\":{\"options\":{\"apiKey\":\"private-test-placeholder\"}}},}\n"
			if isHarness(client) {
				original = "# preserve comment\n- id: llm-pi-ai\n  config:\n    providers:\n      existing:\n        displayName: private-test-placeholder\n- id: other-plugin\n  config:\n    enabled: true\n"
			}
			writeGatewayFixture(t, path, original)
			before, err := s.snapshotClient(client, path)
			if err != nil {
				t.Fatal(err)
			}
			input := Input{Client: client, Revision: before.revision}
			plan, err := s.PreviewPlan(ctx, input)
			if err != nil || len(plan.Models) != 1 || plan.Models[0].ID != "deepseek-flash" || strings.Contains(plan.Summary, "private-test-placeholder") {
				t.Fatal("unsafe or incorrect plan", err)
			}
			var connections int
			if s.db.QueryRow(`SELECT COUNT(*) FROM agent_connections`).Scan(&connections) != nil || connections != 0 {
				t.Fatal("preview created credentials")
			}
			if _, err := s.Apply(ctx, input); err != nil {
				t.Fatal(err)
			}
			key, err := s.Key(ctx, client)
			if err != nil || !s.Authorize(ctx, string(key)) {
				t.Fatal("new client key cannot authorize", err)
			}
			config, err := os.ReadFile(path)
			if err != nil || !strings.Contains(string(config), "preserve comment") || !strings.Contains(string(config), "existing") || !strings.Contains(string(config), "deepseek-flash") || strings.Contains(string(config), "only-responses") {
				t.Fatal("user fields or protocols changed", err)
			}
			if isHarness(client) && strings.Contains(string(config), string(key)) {
				t.Fatal("Harness config contains a credential instead of a reference")
			}
			source.entries = append(source.entries, catalog.Entry{ID: "another-chat", Name: "Another", Protocols: []string{"chat_completions"}})
			if err := s.RefreshCatalog(ctx); err != nil {
				t.Fatal(err)
			}
			config, _ = os.ReadFile(path)
			if !strings.Contains(string(config), "another-chat") {
				t.Fatal("managed catalog not refreshed")
			}
			if _, err := s.Restore(ctx, client); err != nil {
				t.Fatal(err)
			}
			restored, err := os.ReadFile(path)
			if err != nil || string(restored) != original || s.Authorize(ctx, string(key)) {
				t.Fatal("restore lost original config or did not disable key", err)
			}
			clear(key)
		})
	}
}

func TestHarnessProfilesShareCredentialsWithoutOverwritingEachOther(t *testing.T) {
	s, _ := gatewayClientFixture(t)
	ctx := context.Background()
	writeGatewayFixture(t, s.paths.HarnessCredentials, "# credentials\nversion: 1\nrefs:\n  EXISTING_KEY: original-test-secret\nrecords:\n  plugin/account:\n    kind: grant\n    payload: {access: original-test-access}\n")
	for _, client := range []string{"harness-web", "harness-desktop"} {
		path, _ := s.path(client)
		snapshot, err := s.snapshotClient(client, path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.Apply(ctx, Input{Client: client, Revision: snapshot.revision}); err != nil {
			t.Fatal(err)
		}
	}
	webKey, _ := s.Key(ctx, "harness-web")
	defer clear(webKey)
	desktopKey, _ := s.Key(ctx, "harness-desktop")
	defer clear(desktopKey)
	if string(webKey) == string(desktopKey) {
		t.Fatal("profiles share an access key")
	}
	if _, err := s.Restore(ctx, "harness-web"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(s.paths.HarnessCredentials)
	if err != nil {
		t.Fatal(err)
	}
	key, err := harnessCredential(data, harnessRef("harness-desktop"))
	if err != nil || key == nil || *key != string(desktopKey) || !strings.Contains(string(data), "original-test-secret") || !strings.Contains(string(data), "original-test-access") {
		t.Fatal("restore overwrote another profile or credential", err)
	}
	if _, err := os.Stat(s.paths.HarnessWeb); !os.IsNotExist(err) {
		t.Fatal("new profile config not removed")
	}
	if !s.Authorize(ctx, string(desktopKey)) || s.Authorize(ctx, string(webKey)) {
		t.Fatal("profile key lifecycle not isolated")
	}
}

func TestGatewayClientsRefuseExternalEditsAndUnsafeInput(t *testing.T) {
	for _, client := range []string{"opencode", "kilocode", "mimo-code", "qwen-code", "harness-web"} {
		s, _ := gatewayClientFixture(t)
		ctx := context.Background()
		path, _ := s.path(client)
		before, _ := s.snapshotClient(client, path)
		if _, err := s.Apply(ctx, Input{Client: client, Revision: before.revision}); err != nil {
			t.Fatal(err)
		}
		body, _ := os.ReadFile(path)
		body = append(body, []byte("\n# external change\n")...)
		if !isHarness(client) {
			body = append(body[:len(body)-len("# external change\n")], []byte("// external change\n")...)
		}
		if os.WriteFile(path, body, 0600) != nil {
			t.Fatal("write failed")
		}
		if s.RefreshCatalog(ctx) == nil {
			t.Fatal("external edit overwritten on refresh")
		}
		if _, err := s.Restore(ctx, client); err == nil {
			t.Fatal("external edit overwritten on restore")
		}
	}
	for _, body := range []string{`{"provider":{},"provider":{}}`, `{"provider":{"a":{},"a":{}}}`, `[]`, `null`} {
		if _, _, err := openCodeConfig([]byte(body)); err == nil {
			t.Fatal("invalid JSONC accepted")
		}
	}
	for _, body := range []string{"- &loop {config: *loop}", "- id: duplicate\n  id: again", "---\n[]\n---\n[]"} {
		bundle, _ := json.Marshal(harnessBundle{Config: backup{[]byte(body), true}})
		if _, _, err := buildHarness("harness-web", bundle, []byte("test-key"), 3001, Plan{}); err == nil {
			t.Fatal("unsafe YAML accepted")
		}
	}
}
