package modelcatalog

import (
	"strings"

	core "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/provider/workbuddyai"
)

// workbuddyMetadata 把 WorkBuddy AI 的 /v3/config 条目转成网关目录元数据。
// 上游没给的字段保持未知，不补 0，也不猜测上下文或输出上限。
func workbuddyMetadata(model workbuddyai.Model) core.Metadata {
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
