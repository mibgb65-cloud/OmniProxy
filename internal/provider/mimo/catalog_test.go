package mimo

import (
	"slices"
	"testing"
)

func TestCatalogExcludesRetiredModels(t *testing.T) {
	want := []string{"mimo-v2.6-pro", "mimo-v2.6-flash", "mimo-v2.6-pro-ultraspeed"}
	if models := Models(); !slices.Equal(models, want) {
		t.Fatalf("Models() = %v, want %v", models, want)
	}
	for _, model := range []string{"mimo-v2.5-pro", "mimo-v2.5"} {
		if SupportsModel(model) || ModelMetadata(model).ContextWindow != nil {
			t.Errorf("旧模型仍被目录支持：%s", model)
		}
	}
}
