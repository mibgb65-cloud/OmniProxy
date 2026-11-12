package modelcatalog

import (
	"encoding/json"
	"slices"
	"strings"
)

// AntigravityModel 是 Antigravity 桌面端远端目录的一项。字段名由客户端读取：
// maxTokens 是上下文窗口，maxOutputTokens 是输出上限，未知时不发送。
type AntigravityModel struct {
	ModelID          string `json:"modelId"`
	DisplayName      string `json:"displayName"`
	MaxTokens        int64  `json:"maxTokens,omitempty"`
	MaxOutputTokens  int64  `json:"maxOutputTokens,omitempty"`
	SupportsImages   bool   `json:"supportsImages,omitempty"`
	SupportsThinking bool   `json:"supportsThinking,omitempty"`
}

// AntigravityJSON 生成 GET /v1/models?format=antigravity 的响应。
// 只发布 Google 订阅账号的 Gemini 文本模型；模型 ID 与网关实际可解析的 ID 一致。
func AntigravityJSON(entries []Entry) ([]byte, error) {
	list := make([]AntigravityModel, 0, len(entries))
	for _, entry := range entries {
		target := entry.TargetModel
		if target == "" {
			target = entry.ID
		}
		if entry.ProviderKind != "google" || !strings.HasPrefix(target, "gemini-") {
			continue
		}
		name := entry.Name
		if name == "" {
			name = entry.ID
		}
		item := AntigravityModel{
			ModelID: entry.ID, DisplayName: name,
			SupportsImages:   slices.Contains(entry.Metadata.InputModalities, "image"),
			SupportsThinking: len(entry.Metadata.ReasoningEfforts) > 0 || len(entry.Metadata.EffortModels) > 0,
		}
		if entry.Metadata.ContextWindow != nil {
			item.MaxTokens = *entry.Metadata.ContextWindow
		}
		if entry.Metadata.OutputTokenLimit != nil {
			item.MaxOutputTokens = *entry.Metadata.OutputTokenLimit
		}
		list = append(list, item)
	}
	return json.Marshal(struct {
		List []AntigravityModel `json:"list"`
	}{list})
}
