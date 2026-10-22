package gateway

import (
	"encoding/json"
	"strings"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/provider/river"
)

func TestRiverMessagesEffortPreservedAndValidated(t *testing.T) {
	for _, model := range []string{river.DeepSeekFlash, river.DeepSeekFlash1017, river.GLMFlash} {
		for _, effort := range river.ModelMetadata(model).ReasoningEfforts {
			body := []byte(`{"output_config":{"effort":"` + effort.Value + `"}}`)
			raw, err := riverMessagesEffort(body, model)
			if err != nil || string(raw) != `"`+effort.Value+`"` {
				t.Fatal("effort lost", model, effort, err)
			}
		}
	}
	for _, bad := range []string{`null`, `[]`, `{"effort":null}`, `{"effort":10}`, `{"effort":"medium"}`} {
		if _, err := riverMessagesEffort([]byte(`{"output_config":`+bad+`}`), river.DeepSeekFlash1017); err == nil {
			t.Fatal("invalid effort accepted", bad)
		}
	}
	if raw, err := riverMessagesEffort([]byte(`{"thinking":{"type":"disabled"}}`), river.GLMFlash); err != nil || raw != nil {
		t.Fatal("missing effort changed legacy default", err)
	}
	if _, err := riverMessagesEffort([]byte(`{"output_config":{"effort":"none"}}`), river.GLMFlash); err == nil {
		t.Fatal("GLM reasoning was disabled")
	}
	if _, err := riverMessagesEffort([]byte(`{"output_config":{"effort":"high"}}`), river.NemotronVL); err == nil {
		t.Fatal("unverified Nemotron effort accepted")
	}
}

// 客户端原生 effort：模型 ID 只有一行，强度来自 output_config.effort，且必须属于该模型真实枚举。
func TestClaudeSelectionUsesNativeEffortOnSingleModel(t *testing.T) {
	window := int64(1048576)
	confirmed := catalog.Entry{ID: river.DeepSeekFlash1017, Name: "Flash 1017", ProviderKind: "river",
		Metadata: catalog.Metadata{ContextWindow: &window, ReasoningEfforts: []catalog.Effort{{Value: "none"}, {Value: "high"}, {Value: "max"}}, DefaultEffort: "high"}}
	unknown := catalog.Entry{ID: river.NemotronVL, Name: "Nemotron", ProviderKind: "river", Metadata: catalog.Metadata{ContextWindow: &window}}
	entries, err := catalog.ClaudeEntries([]catalog.Entry{confirmed, unknown})
	if err != nil {
		t.Fatal(err)
	}
	for _, model := range []string{confirmed.ID, confirmed.ID + catalog.ClaudeLongContextSuffix, catalog.DesktopAlias(confirmed.ID)} {
		want := confirmed.ID
		if strings.HasSuffix(model, catalog.ClaudeLongContextSuffix) {
			want += catalog.ClaudeLongContextSuffix
		}
		for _, effort := range []string{"none", "high", "max"} {
			got, gotEffort, found := catalog.ResolveClaudeSelection(entries, model, effort)
			if !found || got != want || gotEffort != effort {
				t.Fatal("native effort not applied to the single model", model, effort, got, gotEffort)
			}
		}
		// 未确认的档位不能被透传，解析结果为空档位，由 riverMessagesEffort 明确拒绝。
		if got, gotEffort, found := catalog.ResolveClaudeSelection(entries, model, "medium"); !found || got != want || gotEffort != "" {
			t.Fatal("unconfirmed effort accepted", model, gotEffort, got)
		}
		if got, gotEffort, found := catalog.ResolveClaudeSelection(entries, model, ""); !found || got != want || gotEffort != "" {
			t.Fatal("missing effort invented a level", model, gotEffort, got)
		}
	}
	// 未确认档位的模型：即使客户端继承别的模型的档位也不会显示或被接受。
	if _, gotEffort, found := catalog.ResolveClaudeSelection(entries, unknown.ID, "high"); !found || gotEffort != "" {
		t.Fatal("effort invented for unconfirmed model", gotEffort)
	}
	// 客户端原生 effort 解析必须读到字段本身，不猜测其他字段。
	for _, body := range []string{`{}`, `{"output_config":null}`, `{"output_config":{}}`, `{"output_config":"high"}`} {
		if effort := claudeRequestedEffort([]byte(body)); effort != "" {
			t.Fatal("non-native effort source accepted", body, effort)
		}
	}
	if effort := claudeRequestedEffort([]byte(`{"output_config":{"format":"kept","effort":"max"},"thinking":{"type":"disabled"}}`)); effort != "max" {
		t.Fatal("native effort not read", effort)
	}
}

func TestExplicitClaudeEffortKeepsOtherFieldsAndOverridesStaleThinking(t *testing.T) {
	for _, effort := range []string{"none", "high", "max"} {
		body, err := withClaudeEffort([]byte(`{"model":"route","output_config":{"format":"kept","effort":"medium"},"thinking":{"type":"disabled"},"messages":[{"role":"user","content":"QA"}]}`), effort)
		if err != nil {
			t.Fatal(err)
		}
		var got struct {
			Output   map[string]string `json:"output_config"`
			Thinking map[string]string `json:"thinking"`
			Messages []any             `json:"messages"`
		}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Fatal(err)
		}
		mode := "adaptive"
		if effort == "none" {
			mode = "disabled"
		}
		if got.Output["effort"] != effort || got.Output["format"] != "kept" || got.Thinking["type"] != mode || len(got.Messages) != 1 {
			t.Fatal("effort selection damaged request", got)
		}
	}
	if _, err := withClaudeEffort([]byte(`{"output_config":[]}`), "high"); err == nil {
		t.Fatal("malformed output config accepted")
	}
}
