package workbuddyai

import (
	"slices"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
)

func TestModelsMatchStaticFallback(t *testing.T) {
	models := Models()
	if len(models) != 16 {
		t.Fatalf("models = %d, want 16", len(models))
	}
	displays := DisplayModels()
	if len(displays) != len(models) {
		t.Fatalf("displays = %d", len(displays))
	}
	if models[0] != "default-model" || displays[0] != "Default" || models[15] != "minimax-m3" || displays[15] != "MiniMax-M3" {
		t.Fatalf("static order = %v / %v", models[:1], models[len(models)-1:])
	}
	for index, id := range models {
		if !SupportsModel(id) {
			t.Errorf("SupportsModel(%q) = false", id)
		}
		if displays[index] == "" {
			t.Errorf("empty display name for %q", id)
		}
		if err := catalog.ValidateMetadata(ModelMetadata(id)); err != nil {
			t.Errorf("metadata for %q invalid: %v", id, err)
		}
	}
	if SupportsModel("unknown-model") || ModelName("unknown-model") != "unknown-model" {
		t.Fatal("unknown model accepted")
	}
	if len(ModelMetadata("unknown-model").InputModalities) != 0 {
		t.Fatal("unknown model got metadata")
	}
}

func TestModelMetadataDescribesStaticCapabilities(t *testing.T) {
	gpt := ModelMetadata("gpt-5.5")
	if gpt.ContextWindow == nil || *gpt.ContextWindow != 1000000 {
		t.Fatalf("gpt context = %v", gpt.ContextWindow)
	}
	if gpt.OutputTokenLimit != nil {
		t.Fatalf("gpt output = %v, want unknown", gpt.OutputTokenLimit)
	}
	if !slices.Contains(gpt.InputModalities, "image") {
		t.Fatalf("gpt modalities = %v", gpt.InputModalities)
	}
	if len(gpt.ReasoningEfforts) != 0 {
		t.Fatalf("gpt efforts = %v", gpt.ReasoningEfforts)
	}
	glm := ModelMetadata("glm-5.3")
	if slices.Contains(glm.InputModalities, "image") {
		t.Fatalf("glm modalities = %v", glm.InputModalities)
	}
	if glm.OutputTokenLimit != nil {
		t.Fatalf("glm output = %v, want unknown", glm.OutputTokenLimit)
	}
	if len(glm.ReasoningEfforts) != 3 || glm.ReasoningEfforts[0].Value != "low" || glm.ReasoningEfforts[2].Value != "max" {
		t.Fatalf("glm efforts = %v", glm.ReasoningEfforts)
	}
	if glm.DefaultEffort != "" {
		t.Fatalf("glm default effort = %q", glm.DefaultEffort)
	}
	hy3 := ModelMetadata("hy3")
	if hy3.ContextWindow == nil || *hy3.ContextWindow != 192000 || hy3.OutputTokenLimit == nil || *hy3.OutputTokenLimit != 64000 {
		t.Fatalf("hy3 limits = %v / %v", hy3.ContextWindow, hy3.OutputTokenLimit)
	}
	flash := ModelMetadata("gemini-3.5-flash")
	if flash.OutputTokenLimit == nil || *flash.OutputTokenLimit != 65536 {
		t.Fatalf("flash output = %v", flash.OutputTokenLimit)
	}
}

func TestModelListsAreIndependent(t *testing.T) {
	first := Models()
	first[0] = "mutated"
	if second := Models(); second[0] != "default-model" {
		t.Fatalf("Models returned shared slice: %v", second[0])
	}
	displays := DisplayModels()
	displays[0] = "mutated"
	if ModelName("default-model") != "Default" {
		t.Fatal("display name registry mutated")
	}
}
