package modelcatalog

import (
	"strings"

	"omniproxy/internal/core/routing"
)

// ClaudeLongContextSuffix 是 Claude Code 的百万上下文选项标记。
const ClaudeLongContextSuffix = "[1m]"

// ClaudeLongContextWindow 是生成长上下文选项所需的已确认目标能力下限。
const ClaudeLongContextWindow int64 = 1000000

// ValidClaudeModel 只额外接受 Claude 的精确上下文后缀，保持原长度和模型名边界。
func ValidClaudeModel(model string) bool {
	return len(model) <= 128 && routing.ValidModel(strings.TrimSuffix(model, ClaudeLongContextSuffix))
}

func supportsLongContext(entry Entry) bool {
	return entry.Metadata.ContextWindow != nil && *entry.Metadata.ContextWindow >= ClaudeLongContextWindow
}

// ClaudeEntries 保留普通与可用的 1M 选项；已实测的订阅目标沿用独立的仅 1M 规则。
func ClaudeEntries(entries []Entry) ([]Entry, error) {
	result := make([]Entry, 0, len(entries))
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		seen[entry.ID] = true
	}
	for _, entry := range entries {
		id := entry.ID + ClaudeLongContextSuffix
		available := supportsLongContext(entry) && !strings.HasSuffix(entry.ID, ClaudeLongContextSuffix) && ValidClaudeModel(id)
		onlyLong := available && entry.Metadata.ContextVerified
		if !onlyLong {
			result = append(result, claudeModalities(entry))
		}
		if !available || seen[id] {
			continue
		}
		if entry.Name == "" {
			entry.Name = entry.ID
		}
		entry.ID = id
		entry.Name += ClaudeLongContextSuffix
		if !textValid(entry.Name, 200) {
			entry.Name = id
		}
		if !onlyLong {
			entry.Metadata.IsDefault = false
		}
		result = append(result, claudeModalities(entry))
	}
	// 推理强度沿用客户端原生 effort 控件，不派生出 N×M 的档位模型条目。
	return Normalize(result)
}

// claudeModalities 让 Claude 菜单按上游已确认的能力显示图片输入。
// 这里只公布元数据；实际图片转换仍由各适配器的协议转换负责。
func claudeModalities(entry Entry) Entry {
	if len(entry.Metadata.InputModalities) > 0 {
		return entry
	}
	if entry.ProviderKind == "river" {
		entry.Metadata.InputModalities = []string{"text", "image"}
	}
	return entry
}

// ResolveClaudeSelection 解析客户端选择的模型与推理强度。
// 模型 ID 只承载模型本身；effort 来自客户端 effort 控件（历史档位 ID 仍可解码）。
func ResolveClaudeSelection(entries []Entry, model, requestedEffort string) (string, string, bool) {
	resolved, found := ResolveClaudeModel(entries, model)
	if !found {
		return "", "", false
	}
	if effort, ok := decodeClaudeEffortID(entries, model); ok {
		// 历史选择自带档位，不能被客户端沿用的旧档位覆盖。
		return resolved, effort, true
	}
	if requestedEffort == "" {
		return resolved, "", true
	}
	for _, entry := range entries {
		if entry.ID == resolved {
			if ClaudeEffort(entry, requestedEffort) {
				return resolved, requestedEffort, true
			}
			break
		}
	}
	return resolved, "", true
}

// decodeClaudeEffortID 解码历史 omniproxy-effort-* 选择，返回其中的档位。
func decodeClaudeEffortID(entries []Entry, model string) (string, bool) {
	if entry, effort, ok := archivedEffortMatch(entries, model); ok && entry.ID != "" {
		return effort, true
	}
	return "", false
}

// ResolveClaudeModel 保留目录的精确 ID 优先级，只对已确认能力的目标去除后缀。
func ResolveClaudeModel(entries []Entry, model string) (string, bool) {
	if !ValidClaudeModel(model) {
		return "", false
	}
	for _, entry := range entries {
		if entry.ID == model {
			return entry.ID, true
		}
	}
	base := strings.TrimSuffix(model, ClaudeLongContextSuffix)
	for _, entry := range entries {
		if entry.ID != base && DesktopAlias(entry.ID) != base {
			continue
		}
		if base != model && !supportsLongContext(entry) {
			return "", false
		}
		return entry.ID, true
	}
	for _, entry := range entries {
		if !supportsLongContext(entry) {
			continue
		}
		// Desktop 的 prefer1m 选择和 CLI 的去后缀请求都使用独立的长上下文基础 ID。
		alias := DesktopAlias(entry.ID + ClaudeLongContextSuffix)
		if model == alias || model == strings.TrimSuffix(alias, ClaudeLongContextSuffix) {
			return entry.ID, true
		}
	}
	return "", false
}
