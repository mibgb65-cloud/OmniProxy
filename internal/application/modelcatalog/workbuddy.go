package modelcatalog

import (
	"strings"

	core "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/provider/workbuddyai"
)

// workbuddyMetadata 把 WorkBuddy 的 /v3/config 条目转成网关目录元数据。
// 上游没给的字段保持未知，不补 0，也不猜测上下文或输出上限。
//
// messagesNative 表示该构建把 Claude 模型放在原生 Anthropic Messages 上服务：
// 这类模型在 Messages 上接受 `max` 档位（社区插件实测 200 且思考更多），即使目录的
// supportedEfforts 只到 xhigh，因此补上 `max` 让客户端能选到它。
func workbuddyMetadata(model workbuddyai.Model, messagesNative bool) core.Metadata {
	metadata := core.Metadata{}
	if model.Context > 0 {
		window := int64(model.Context)
		metadata.ContextWindow = &window
	}
	if model.Output > 0 {
		output := int64(model.Output)
		metadata.OutputTokenLimit = &output
	}
	inputs := []string{"text"}
	if model.Images {
		inputs = append(inputs, "image")
	}
	metadata.InputModalities = inputs
	tools := true
	metadata.ToolUse = &tools
	seen := map[string]bool{}
	for _, effort := range model.Efforts {
		value := normalizeEffort(effort)
		if value == "" || value == "none" || seen[value] || len(metadata.ReasoningEfforts) >= 16 {
			continue
		}
		seen[value] = true
		metadata.ReasoningEfforts = append(metadata.ReasoningEfforts, core.Effort{Value: value})
	}
	if messagesNative && len(metadata.ReasoningEfforts) > 0 && !seen["max"] && len(metadata.ReasoningEfforts) < 16 &&
		strings.HasPrefix(strings.ToLower(model.ID), "claude-") {
		metadata.ReasoningEfforts = append(metadata.ReasoningEfforts, core.Effort{Value: "max"})
		seen["max"] = true
	}
	if len(metadata.ReasoningEfforts) > 0 && seen[normalizeEffort(model.DefaultEffort)] {
		metadata.DefaultEffort = normalizeEffort(model.DefaultEffort)
	}
	return metadata
}

// normalizeEffort 只接受网关目录允许的档位词表（小写字母、数字、下划线与连字符）。
// 上游若给出别的拼写就丢弃该档位，而不是让整份目录同步失败。
func normalizeEffort(value string) string {
	if len(value) == 0 || len(value) > 32 {
		return ""
	}
	for _, r := range value {
		if r >= 'A' && r <= 'Z' {
			r = r - 'A' + 'a'
		}
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' || r == '-') {
			return ""
		}
	}
	return strings.ToLower(value)
}
