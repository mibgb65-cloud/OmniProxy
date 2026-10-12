package modelcatalog

import "slices"

// ForProtocol 排除已确认不支持文本／工具或目标协议的模型，未知值不作负面推断。
func ForProtocol(entries []Entry, protocol string) []Entry {
	result := make([]Entry, 0, len(entries))
	for _, entry := range entries {
		if len(entry.Protocols) > 0 && !slices.Contains(entry.Protocols, protocol) {
			continue
		}
		if entry.Metadata.ToolUse != nil && !*entry.Metadata.ToolUse {
			continue
		}
		if len(entry.Metadata.InputModalities) > 0 && !slices.Contains(entry.Metadata.InputModalities, "text") {
			continue
		}
		result = append(result, entry)
	}
	return result
}
