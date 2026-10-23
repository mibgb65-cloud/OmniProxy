package agentconfig

import (
	"context"
	"fmt"
	"github.com/pelletier/go-toml/v2"
	"omniproxy/internal/infrastructure/database"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type memoryStore struct{ values map[string][]byte }

func (s *memoryStore) Put(b []byte) (string, error) {
	id := fmt.Sprint(len(s.values) + 1)
	s.values[id] = append([]byte(nil), b...)
	return id, nil
}
func (s *memoryStore) Get(id string) ([]byte, error) {
	return append([]byte(nil), s.values[id]...), nil
}
func (s *memoryStore) Delete(id string) error { delete(s.values, id); return nil }

type testCatalog struct{}

func (testCatalog) AvailableModels(context.Context) ([]string, error) {
	return []string{"gpt-test"}, nil
}
func fixture(t *testing.T) *Service {
	t.Helper()
	root := t.TempDir()
	db, e := database.Open(database.Config{Path: filepath.Join(root, "db.sqlite")})
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { db.Close() })
	if e = database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); e != nil {
		t.Fatal(e)
	}
	return New(db, &memoryStore{map[string][]byte{}}, Paths{Codex: filepath.Join(root, "codex", "config.toml"), Claude: filepath.Join(root, "claude", "settings.json"), ClaudeDesktop: filepath.Join(root, "Claude-3p", "configLibrary", DesktopProfileID+".json")}, 3001, testCatalog{})
}
func TestCodexPreservesIdentityAndUnrelatedConfig(t *testing.T) {
	for _, original := range []string{"", "# original\nmodel_provider = \"openai\"\nmodel = \"old\"\n[projects.\"E:\\\\work\"]\ntrust_level=\"trusted\"\n", "model_provider='existing'\n[model_providers.existing]\nname='Existing'\nbase_url='https://old.invalid'\nenv_key='OLD_KEY'\n# retained\n[features]\nmulti_agent=true\n", "model_provider='existing'\n[model_providers.existing]\nbase_url = '''https://old.invalid\n/path'''\n[other]\nsecret = 'private-test-only'\n", "model_provider='openai'\n[features]\nmulti_agent=true\n"} {
		result, provider, err := buildCodex([]byte(original), []byte("local-test-key"), "http://127.0.0.1:3001/codex/v1", "gpt-test")
		if err != nil {
			t.Fatal(err)
		}
		var before, after map[string]any
		toml.Unmarshal([]byte(original), &before)
		if toml.Unmarshal(result, &after) != nil {
			t.Fatal("invalid TOML")
		}
		if after["model_provider"] != before["model_provider"] {
			t.Fatal("provider identity changed")
		}
		if provider == "openai" {
			if after["openai_base_url"] != "http://127.0.0.1:3001/codex/v1" {
				t.Fatal("built-in provider endpoint missing")
			}
			continue
		}
		entry := after["model_providers"].(map[string]any)[provider].(map[string]any)
		if entry["base_url"] != "http://127.0.0.1:3001/codex/v1" || entry["requires_openai_auth"] != false {
			t.Fatal("gateway not configured")
		}
		if before["features"] != nil && after["features"].(map[string]any)["multi_agent"] != true {
			t.Fatal("feature changed")
		}
		if strings.Contains(original, "# retained") && !strings.Contains(string(result), "# retained") {
			t.Fatal("unrelated comment lost")
		}
	}
}
func TestApplyBackupRestoreAndConcurrentEditProtection(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	os.MkdirAll(filepath.Dir(s.paths.Codex), 0700)
	original := []byte("model_provider='openai'\nmodel='old'\n")
	os.WriteFile(s.paths.Codex, original, 0600)
	session := filepath.Join(filepath.Dir(s.paths.Codex), "sessions.jsonl")
	os.WriteFile(session, []byte("official-history"), 0600)
	input := Input{Client: "codex", Model: "gpt-test", Revision: digest(original, true)}
	preview, err := s.Preview(ctx, input)
	if err != nil || strings.Contains(preview, "opk_") {
		t.Fatal("unsafe preview", err)
	}
	saved, err := s.Apply(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if !saved[0].CanRestore || !saved[0].Configured {
		t.Fatal("missing backup")
	}
	bytes, _ := os.ReadFile(session)
	if string(bytes) != "official-history" {
		t.Fatal("session modified")
	}
	key, err := s.Key(ctx, "codex")
	if err != nil || !s.Authorize(ctx, string(key)) || s.Authorize(ctx, strings.Repeat("x", 68)) {
		t.Fatal("bad authorization")
	}
	keyText := string(key)
	clear(key)
	applied, _ := os.ReadFile(s.paths.Codex)
	os.WriteFile(s.paths.Codex, append(applied, []byte("# external edit\n")...), 0600)
	if _, err = s.Restore(ctx, "codex"); err == nil {
		t.Fatal("external edit overwritten")
	}
	if _, err = s.Apply(ctx, Input{Client: "codex", Model: "gpt-test", Revision: digest(append(applied, []byte("# external edit\n")...), true)}); err == nil {
		t.Fatal("external edit accepted as owned")
	}
	os.WriteFile(s.paths.Codex, applied, 0600)
	if _, err = s.Restore(ctx, "codex"); err != nil {
		t.Fatal(err)
	}
	restored, _ := os.ReadFile(s.paths.Codex)
	if string(restored) != string(original) {
		t.Fatal("original not restored exactly")
	}
	if s.Authorize(ctx, keyText) {
		t.Fatal("restored key still enabled")
	}
}
func TestClaudeDesktopActivatesGatewayAndRestoresOfficialSelection(t *testing.T) {
	s := fixture(t)
	meta := filepath.Join(filepath.Dir(s.paths.ClaudeDesktop), "_meta.json")
	os.MkdirAll(filepath.Dir(meta), 0700)
	os.WriteFile(meta, []byte(`{"appliedId":"official","entries":[{"id":"official","name":"Official"}]}`), 0600)
	if _, err := s.Apply(context.Background(), Input{Client: "claude-desktop", Model: "gpt-test", Revision: digest(nil, false)}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(meta)
	if !strings.Contains(string(raw), `"appliedId": "`+DesktopProfileID+`"`) || !strings.Contains(string(raw), `"id": "official"`) {
		t.Fatal("gateway not selected or official entry lost")
	}
	if _, err := s.Restore(context.Background(), "claude-desktop"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(s.paths.ClaudeDesktop); !os.IsNotExist(err) {
		t.Fatal("new profile not removed")
	}
	raw, err := os.ReadFile(meta)
	if err != nil || !strings.Contains(string(raw), `"appliedId": "official"`) || strings.Contains(string(raw), DesktopProfileID) {
		t.Fatal("official selection not restored")
	}
}
