package agentconfig

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

func TestConfirmedSubscriptionApplyAndRestore(t *testing.T) {
	for _, client := range []string{"claude", "claude-desktop"} {
		t.Run(client, func(t *testing.T) {
			s, source := catalogFixture(t)
			window := catalog.ClaudeLongContextWindow
			source.entries = []catalog.Entry{{ID: "gpt-6.1-sol", Name: "GPT-6.1-Sol", Metadata: catalog.Metadata{ContextWindow: &window, ContextVerified: true}}, {ID: "unverified"}}
			ctx := context.Background()
			input := Input{Client: client, Revision: digest(nil, false), Model: "gpt-6.1-sol"}
			plan, err := s.PreviewPlan(ctx, input)
			if err != nil || plan.DefaultModel != "gpt-6.1-sol[1m]" || len(plan.Models) != 2 {
				t.Fatal("confirmed default not upgraded", err)
			}
			if _, err := s.Apply(ctx, input); err != nil {
				t.Fatal(err)
			}
			path, _ := s.path(client)
			body, err := os.ReadFile(path)
			var config map[string]any
			if err != nil || json.Unmarshal(body, &config) != nil {
				t.Fatal("cannot read confirmed config", err)
			}
			if client == "claude" {
				env := config["env"].(map[string]any)
				settings := config["modelSettings"].(map[string]any)
				if env["ANTHROPIC_MODEL"] != plan.DefaultModel || env["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] != nil || settings[strings.TrimSuffix(plan.DefaultModel, catalog.ClaudeLongContextSuffix)].(map[string]any)["autoCompactWindow"] != float64(850000) {
					t.Fatal("long window or input margin lost")
				}
				if _, found := settings["unverified"]; found {
					t.Fatal("unverified model received subscription policy")
				}
			} else {
				models := config["inferenceModels"].([]any)
				first := models[0].(map[string]any)
				if first["name"] != catalog.DesktopAlias(plan.DefaultModel) || first["supports1m"] != true || first["prefer1m"] != true {
					t.Fatal("native long-context capability missing")
				}
				if err := s.RefreshCatalog(ctx); err != nil {
					t.Fatal(err)
				}
				// 官方默认变化不能覆盖仍可用的客户端默认。
				source.entries[1].Metadata.IsDefault = true
				if err := s.RefreshCatalog(ctx); err != nil {
					t.Fatal(err)
				}
				body, err = os.ReadFile(path)
				if err != nil || json.Unmarshal(body, &config) != nil || config["inferenceModels"].([]any)[0].(map[string]any)["name"] != catalog.DesktopAlias(plan.DefaultModel) {
					t.Fatal("directory refresh replaced current desktop default", err)
				}
				preview, err := s.PreviewPlan(ctx, Input{Client: client, Revision: digest(body, true)})
				if err != nil || preview.DefaultModel != plan.DefaultModel {
					t.Fatal("one-click configuration replaced current desktop default", err)
				}
			}
			if _, err := s.Restore(ctx, client); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("managed file not restored")
			}
		})
	}
}

func TestCompactMarginPreservesSmallerUserLimitAndRejectsMalformedValues(t *testing.T) {
	entries := []catalog.Entry{{ID: "gpt-6.1-sol[1m]", Metadata: catalog.Metadata{ContextVerified: true}}, {ID: "deepseek-flash[1m]"}}
	base := "gpt-6.1-sol"
	for _, value := range []any{float64(200000), float64(950000), "auto", "bad", float64(-1), float64(0), float64(1000001), float64(200000.5)} {
		config := map[string]any{"env": map[string]any{"CLAUDE_CODE_AUTO_COMPACT_WINDOW": "200000"}, "modelSettings": map[string]any{base: map[string]any{"autoCompactWindow": value, "effortLevel": "high"}}}
		err := subscriptionCompactSettings(config, entries)
		if value == float64(200000) || value == float64(950000) || value == "auto" {
			if err != nil {
				t.Fatal(err)
			}
			wanted := value
			if value == float64(950000) || value == "auto" {
				wanted = float64(850000)
			}
			settings := config["modelSettings"].(map[string]any)
			model := settings[base].(map[string]any)
			if model["autoCompactWindow"] != wanted || model["effortLevel"] != "high" || settings[entries[0].ID] != nil || settings[entries[1].ID] != nil || config["env"].(map[string]any)["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] != "200000" {
				t.Fatal("unsafe compact threshold or user limit overwritten")
			}
		} else if err == nil {
			t.Fatal("malformed compact threshold accepted")
		}
	}
}
