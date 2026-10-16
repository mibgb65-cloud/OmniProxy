package agentconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

func TestClaudePickerApplyRefreshAndRestore(t *testing.T) {
	s, source := catalogFixture(t)
	ctx := context.Background()
	original := []byte(`{"theme":"dark","permissions":{"allow":[]},"availableModels":["gpt-default","coding-route"],"modelPicker":{"options":[{"model":"old","label":"Old"}]}}`)
	if err := os.MkdirAll(filepath.Dir(s.paths.Claude), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s.paths.Claude, original, 0600); err != nil {
		t.Fatal(err)
	}
	input := Input{Client: "claude", Revision: digest(original, true)}
	plan, err := s.PreviewPlan(ctx, input)
	if err != nil || plan.Mode != "cli-model-list" || len(plan.Models) != 2 {
		t.Fatal("CLI preview does not advertise the full picker", err)
	}
	if _, err := s.Apply(ctx, input); err != nil {
		t.Fatal(err)
	}
	config := checkClaudePicker(t, s.paths.Claude, plan.Models)
	if config["theme"] != "dark" || config["permissions"] == nil || len(config["availableModels"].([]any)) != 2 {
		t.Fatal("unrelated settings or user model restrictions changed")
	}
	selected := config["env"].(map[string]any)["ANTHROPIC_MODEL"]
	source.entries = []catalog.Entry{{ID: "gpt-default", Name: "Updated GPT"}, {ID: "new-route", Alias: true}}
	if err := s.RefreshCatalog(ctx); err != nil {
		t.Fatal(err)
	}
	entries, err := s.clientEntries(ctx, "claude")
	if err != nil {
		t.Fatal(err)
	}
	config = checkClaudePicker(t, s.paths.Claude, entries)
	if config["env"].(map[string]any)["ANTHROPIC_MODEL"] != selected || config["theme"] != "dark" {
		t.Fatal("refresh changed the default model or unrelated settings")
	}
	if _, err := s.Restore(ctx, "claude"); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(s.paths.Claude)
	if err != nil || !bytes.Equal(restored, original) {
		t.Fatal("original model picker and settings were not restored", err)
	}
}

func TestClaudePickerRefreshProtectsExternalChangesAndUnmanagedFiles(t *testing.T) {
	for _, managed := range []bool{false, true} {
		t.Run(map[bool]string{false: "unmanaged", true: "externally-edited"}[managed], func(t *testing.T) {
			s, _ := catalogFixture(t)
			ctx := context.Background()
			if err := os.MkdirAll(filepath.Dir(s.paths.Claude), 0700); err != nil {
				t.Fatal(err)
			}
			if managed {
				if _, err := s.Apply(ctx, Input{Client: "claude", Revision: digest(nil, false)}); err != nil {
					t.Fatal(err)
				}
			}
			external := []byte(`{"model":"my-choice","modelPicker":{"options":[{"model":"private"}]}}`)
			if err := os.WriteFile(s.paths.Claude, external, 0600); err != nil {
				t.Fatal(err)
			}
			err := s.RefreshCatalog(ctx)
			if managed && (err == nil || err.Error() != "agent: config_changed") || !managed && err != nil {
				t.Fatal("unexpected refresh protection result", err)
			}
			actual, err := os.ReadFile(s.paths.Claude)
			if err != nil || !bytes.Equal(actual, external) {
				t.Fatal("external settings were overwritten", err)
			}
		})
	}
}

func checkClaudePicker(t *testing.T, path string, entries []catalog.Entry) map[string]any {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(body, &config); err != nil {
		t.Fatal(err)
	}
	picker, ok := config["modelPicker"].(map[string]any)
	if !ok || picker["replaceBuiltInOptions"] != true {
		t.Fatal("CLI still shows built-in options instead of the gateway catalog")
	}
	options := picker["options"].([]any)
	if len(options) != len(entries) {
		t.Fatal("CLI picker differs from the preview model count")
	}
	for i, entry := range entries {
		row := options[i].(map[string]any)
		if row["model"] != entry.ID || row["label"] != entry.Name || row["behavesAs"] != nil {
			t.Fatal("CLI picker model IDs, labels, order or capabilities changed")
		}
	}
	return config
}
