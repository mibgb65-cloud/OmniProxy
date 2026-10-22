package agentconfig

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
		if row["model"] != entry.ID || row["label"] != entry.Name {
			t.Fatal("CLI picker model IDs, labels or order changed")
		}
		// 每个模型只占一行：推理强度由客户端原生 effort 控件选择，不再派生档位模型。
		if strings.HasPrefix(entry.ID, catalog.ClaudeEffortPrefix) {
			t.Fatal("CLI picker still lists a derived effort model", entry.ID)
		}
		// 只有已确认档位枚举与默认值的模型才继承 effort 能力。
		wantBehaves := entry.ProviderKind == "river" && len(entry.Metadata.ReasoningEfforts) > 0 && entry.Metadata.DefaultEffort != ""
		if wantBehaves != (row["behavesAs"] != nil) {
			t.Fatal("effort capability grant does not match the confirmed metadata", entry.ID)
		}
	}
	return config
}

// 已确认档位的模型必须在客户端显示原生 effort 控件，且默认与上限不超过真实枚举。
func TestClaudePickerGrantsNativeEffortWithinConfirmedRange(t *testing.T) {
	entries := []catalog.Entry{
		{ID: "river-confirmed", Name: "River Confirmed", ProviderKind: "river", Metadata: catalog.Metadata{
			ReasoningEfforts: []catalog.Effort{{Value: "low"}, {Value: "high"}, {Value: "max"}}, DefaultEffort: "high"}},
		{ID: "river-unknown", Name: "River Unknown", ProviderKind: "river"},
		{ID: "other-confirmed", Name: "Other", ProviderKind: "other", Metadata: catalog.Metadata{
			ReasoningEfforts: []catalog.Effort{{Value: "high"}}, DefaultEffort: "high"}},
	}
	config := map[string]any{"modelSettings": map[string]any{"river-confirmed": map[string]any{"autoCompactWindow": 850000.0}}}
	if err := claudeEffortSettings(config, entries); err != nil {
		t.Fatal(err)
	}
	settings := config["modelSettings"].(map[string]any)
	model := settings["river-confirmed"].(map[string]any)
	// 用户保存的 autoCompactWindow 必须保留，同一条目只增补档位。
	if model["autoCompactWindow"] != 850000.0 || model["effortLevel"] != "high" || model["maxEffortLevel"] != "max" {
		t.Fatal("effort settings damaged or exceeded the confirmed range", model)
	}
	for _, id := range []string{"river-unknown", "other-confirmed"} {
		if settings[id] != nil {
			t.Fatal("settings written for a model without confirmed effort", id)
		}
	}
	// 用户显式保存的档位与上限不被覆盖。
	saved := map[string]any{"modelSettings": map[string]any{"river-confirmed": map[string]any{"effortLevel": "low", "maxEffortLevel": "high"}}}
	if err := claudeEffortSettings(saved, entries); err != nil {
		t.Fatal(err)
	}
	kept := saved["modelSettings"].(map[string]any)["river-confirmed"].(map[string]any)
	if kept["effortLevel"] != "low" || kept["maxEffortLevel"] != "high" {
		t.Fatal("explicit user effort overwritten", kept)
	}
	if err := claudeEffortSettings(map[string]any{"modelSettings": "broken"}, entries); err == nil {
		t.Fatal("malformed modelSettings accepted")
	}
}
