package modelcatalog

import (
	"encoding/json"
	"slices"
	"testing"
)

// River 官方已公布图片输入；Claude 菜单必须据此显示视觉能力。
// Codex 继续只承认原生订阅线路，不能因为其他适配器有图片就放宽。
func TestClaudeImageCapabilityFollowsVerifiedAdapterOnly(t *testing.T) {
	window := ClaudeLongContextWindow
	entries := []Entry{
		{ID: "deepseek-flash", Name: "DeepSeek-V4.1-Flash", ProviderKind: "river", Metadata: Metadata{ContextWindow: &window, InputModalities: []string{"text", "image"}}},
		{ID: "text-only", Name: "Text Only", ProviderKind: "river", Metadata: Metadata{ContextWindow: &window, InputModalities: []string{"text"}}},
		{ID: "gpt-6.1-sol", Name: "GPT", ProviderKind: "openai", Metadata: Metadata{InputModalities: []string{"text", "image"}}},
	}
	variants, err := ClaudeEntries(entries)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range variants {
		wantImage := entry.ProviderKind == "river" && entry.ID != "text-only" && entry.ID != "text-only"+ClaudeLongContextSuffix || entry.ID == "gpt-6.1-sol"
		gotImage := slices.Contains(entry.Metadata.InputModalities, "image")
		if gotImage != wantImage {
			t.Errorf("%s image modality = %v, want %v", entry.ID, gotImage, wantImage)
		}
		if !slices.Contains(entry.Metadata.InputModalities, "text") {
			t.Errorf("%s lost text modality", entry.ID)
		}
	}

	// River 适配器已按官方契约转发 Responses input_image 与 Chat image_url，
	// 因此 Codex 目录同样声明图片；未声明能力的 River 条目仍只发布文本。
	body, err := CodexJSON([]Entry{
		{ID: "deepseek-flash", ProviderKind: "river", Metadata: Metadata{InputModalities: []string{"text", "image"}}},
		{ID: "text-only", ProviderKind: "river", Metadata: Metadata{InputModalities: []string{"text"}}},
		{ID: "legacy-chat", ProviderKind: "deepseek", Metadata: Metadata{InputModalities: []string{"text", "image"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var codex struct {
		Models []struct {
			Modalities []string `json:"input_modalities"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &codex); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(codex.Models[0].Modalities, []string{"text", "image"}) {
		t.Fatalf("verified river image input not advertised: %v", codex.Models[0].Modalities)
	}
	if !slices.Equal(codex.Models[1].Modalities, []string{"text"}) {
		t.Fatalf("text-only river entry gained image input: %v", codex.Models[1].Modalities)
	}
	// 其他适配器仍未验证跨协议图片转换，不能因元数据声明就放宽。
	if !slices.Equal(codex.Models[2].Modalities, []string{"text"}) {
		t.Fatalf("unverified adapter advertised image input: %v", codex.Models[2].Modalities)
	}
}
