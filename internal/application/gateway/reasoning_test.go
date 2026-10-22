package gateway

import (
	"encoding/json"
	"testing"

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
