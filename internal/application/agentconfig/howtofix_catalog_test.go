package agentconfig

import (
	"context"
	"encoding/json"
	"testing"

	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/provider/howtofix"
)

// TestHowToFixDiscoveredModelReachesManagedClients 断言 HowToFix 账号同步到的模型会进入
// 受管客户端目录：Chat 客户端（OpenCode／Harness／Qwen…）走 chat 协议，Claude 走 messages，
// Codex 走 responses。模型能力未知不影响发布，只是不声明图片、工具与长上下文变体。
func TestHowToFixDiscoveredModelReachesManagedClients(t *testing.T) {
	s, c := catalogFixture(t)
	window := int64(1000000)
	c.entries = append(c.entries, catalog.Entry{
		ID: "glm-5.3-flash", Name: "glm-5.3-flash", ProviderKind: howtofix.Kind,
		Protocols: []string{"chat_completions", "responses", "messages"},
		Metadata:  catalog.Metadata{ContextWindow: &window},
	})
	for _, client := range []string{"claude", "codex", "harness-web", "harness-desktop", "opencode", "qwen-code"} {
		entries, err := s.clientEntries(context.Background(), client)
		if err != nil {
			t.Fatalf("%s: %v", client, err)
		}
		found := false
		for _, entry := range entries {
			if entry.ID == "glm-5.3-flash" {
				found = true
				break
			}
		}
		if !found {
			ids := make([]string, 0, len(entries))
			for _, entry := range entries {
				ids = append(ids, entry.ID)
			}
			t.Fatalf("%s 的受管目录缺少 HowToFix 模型: %v", client, ids)
		}
	}
	// Codex 目录文件按 slug 发布模型；HowToFix 模型必须出现在同一份 JSON 里。
	body, err := catalog.CodexJSON(c.entries)
	if err != nil {
		t.Fatal(err)
	}
	var stored struct {
		Models []struct {
			Slug string `json:"slug"`
		} `json:"models"`
	}
	if json.Unmarshal(body, &stored) != nil {
		t.Fatal("codex catalog is not valid JSON")
	}
	for _, model := range stored.Models {
		if model.Slug == "glm-5.3-flash" {
			return
		}
	}
	t.Fatalf("Codex 目录缺少 HowToFix 模型: %s", body)
}
