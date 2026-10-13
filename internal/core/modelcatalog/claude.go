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

// ClaudeEntries 的 DeepSeek 目标只提供已确认的 1M，其他目标保持原默认和可用变体。
func ClaudeEntries(entries []Entry) ([]Entry, error) {
	result := make([]Entry, 0, len(entries))
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		seen[entry.ID] = true
	}
	for _, entry := range entries {
		id := entry.ID + ClaudeLongContextSuffix
		available := supportsLongContext(entry) && !strings.HasSuffix(entry.ID, ClaudeLongContextSuffix) && ValidClaudeModel(id)
		onlyLong := available && entry.ProviderKind == "deepseek"
		if !onlyLong {
			result = append(result, entry)
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
		result = append(result, entry)
	}
	return Normalize(result)
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
