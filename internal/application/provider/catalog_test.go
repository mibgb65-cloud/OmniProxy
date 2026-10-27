package provider

import (
	"reflect"
	"testing"

	"omniproxy/internal/provider/river"
)

// 预设模型只是卡片预览（不代表账号权限），但仍固定住官方清单，
// 避免界面回退到已过时的型号或漏掉 River 新增型号。
func TestBuiltInPresetModelsMatchOfficialLists(t *testing.T) {
	openai, ok := definitionForKind("openai")
	if !ok || !reflect.DeepEqual(openai.Models, []string{"GPT-6 Astra", "GPT-6.1 Sol", "GPT-6.1 Terra", "GPT-6 Luna"}) {
		t.Fatalf("OpenAI preset drifted from the official featured models: %v", openai.Models)
	}
	definition, ok := definitionForKind(river.Kind)
	if !ok || !reflect.DeepEqual(definition.Models, river.DisplayModels()) || len(definition.Models) != len(river.Models()) {
		t.Fatalf("River preset drifted from the provider catalog: %v", definition.Models)
	}
}
