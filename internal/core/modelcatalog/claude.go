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

// ClaudeEntries 为已确认支持 1M 的目标增加客户端选项，普通模型保持原默认。
func ClaudeEntries(entries []Entry) ([]Entry, error) {
	result := append([]Entry(nil), entries...)
	seen := make(map[string]bool, len(entries))
	for _, entry := range entries {
		seen[entry.ID] = true
	}
	for _, entry := range entries {
		id := entry.ID + ClaudeLongContextSuffix
		if !supportsLongContext(entry) || strings.HasSuffix(entry.ID, ClaudeLongContextSuffix) || seen[id] || !ValidClaudeModel(id) {
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
		entry.Metadata.IsDefault = false
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
	return "", false
}
