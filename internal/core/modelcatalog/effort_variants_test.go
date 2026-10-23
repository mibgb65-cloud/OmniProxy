package modelcatalog

import "testing"

func TestFoldEffortVariantsCollapsesIntoBaseModel(t *testing.T) {
	entries := []Entry{
		{ID: "gemini-3.8-flash", Name: "Gemini 3.8 Flash"},
		{ID: "gemini-3.8-flash-high", Name: "Gemini 3.8 Flash (High)"},
		{ID: "gemini-3.8-flash-low", Name: "Gemini 3.8 Flash (Low)"},
		{ID: "gemini-3.8-flash-medium", Name: "Gemini 3.8 Flash (Medium)"},
		{ID: "gemini-3.8-flash-tts", Name: "Gemini 3.8 Flash TTS"},
	}
	folded := FoldEffortVariants(entries)
	if len(folded) != 2 {
		t.Fatalf("entries = %d, want base + tts only: %#v", len(folded), folded)
	}
	var base Entry
	for _, entry := range folded {
		if entry.ID == "gemini-3.8-flash" {
			base = entry
		}
	}
	if len(base.Metadata.EffortModels) != 3 {
		t.Fatalf("effort models = %v, want three levels", base.Metadata.EffortModels)
	}
	if variant, ok := EffortModel(base, "low"); !ok || variant != "gemini-3.8-flash-low" {
		t.Fatalf("low level = (%q,%t)", variant, ok)
	}
	levels := EffortLevels(base)
	if len(levels) != 3 || levels[0] != "low" || levels[1] != "medium" || levels[2] != "high" {
		t.Fatalf("levels = %v, want low/medium/high order", levels)
	}
	if base.Metadata.DefaultEffort != "high" {
		t.Fatalf("default effort = %q, want high", base.Metadata.DefaultEffort)
	}
	if ValidateMetadata(base.Metadata) != nil {
		t.Fatal("folded metadata rejected by validation")
	}
}

// 没有同名基础模型时保持原样，避免把不同型号误合并。
func TestFoldEffortVariantsKeepsOrphans(t *testing.T) {
	entries := []Entry{
		{ID: "gemini-9-flash-agent", Name: "Gemini 9 Flash (High)"},
		{ID: "gemini-3.5-flash", Name: "Gemini 3.5 Flash"},
	}
	folded := FoldEffortVariants(entries)
	if len(folded) != 2 {
		t.Fatalf("entries = %d, want both kept", len(folded))
	}
	if len(folded[0].Metadata.EffortModels) != 0 || len(folded[1].Metadata.EffortModels) != 0 {
		t.Fatal("unrelated names must not be merged")
	}
}

func TestFoldEffortVariantsKeepsFirstMappingPerLevel(t *testing.T) {
	entries := []Entry{
		{ID: "base", Name: "Model"},
		{ID: "variant-a", Name: "Model (High)"},
		{ID: "variant-b", Name: "Model (High)"},
	}
	folded := FoldEffortVariants(entries)
	if len(folded) != 1 {
		t.Fatalf("entries = %d, want one folded entry", len(folded))
	}
	if folded[0].Metadata.EffortModels["high"] != "variant-a" {
		t.Fatalf("high mapped to %q, want first variant", folded[0].Metadata.EffortModels["high"])
	}
}

// 折叠后旧的变体 ID 必须能还原成基础模型 + 档位，历史选择不会变成未知模型。
func TestEffortVariantBaseResolvesLegacySelection(t *testing.T) {
	entries := FoldEffortVariants([]Entry{
		{ID: "gemini-3.8-flash", Name: "Gemini 3.8 Flash"},
		{ID: "gemini-3.8-flash-low", Name: "Gemini 3.8 Flash (Low)"},
	})
	entry, level, ok := EffortVariantBase(entries, "gemini-3.8-flash-low")
	if !ok || entry.ID != "gemini-3.8-flash" || level != "low" {
		t.Fatalf("resolved (%q,%q,%t)", entry.ID, level, ok)
	}
	if _, _, ok := EffortVariantBase(entries, "gemini-3.8-flash"); ok {
		t.Fatal("base model must not resolve as a variant")
	}
}

func TestValidateMetadataRejectsUnknownEffortModelLevel(t *testing.T) {
	metadata := Metadata{EffortModels: map[string]string{"ultra": "model-x"}}
	if ValidateMetadata(metadata) == nil {
		t.Fatal("unknown effort level accepted")
	}
	metadata = Metadata{EffortModels: map[string]string{"high": "bad model"}}
	if ValidateMetadata(metadata) == nil {
		t.Fatal("invalid variant model id accepted")
	}
}
