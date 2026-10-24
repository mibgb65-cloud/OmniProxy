package modelcatalog

import "strings"

// UniqueNames 把重名的目录条目退回使用模型 ID 作为显示名。
//
// 上游目录里不同 ID 可能共用同一个显示名（实测 Gemini 3.5 Flash Lite 有四个不同型号），
// 客户端菜单会出现多行完全相同的选项，无法区分。重名时改用 ID，保证每一行可辨识；
// 不重名的条目保持上游给出的显示名。
func UniqueNames(entries []Entry) []Entry {
	counts := make(map[string]int, len(entries))
	for _, entry := range entries {
		counts[strings.ToLower(entry.Name)]++
	}
	result := make([]Entry, len(entries))
	copy(result, entries)
	for index, entry := range result {
		if counts[strings.ToLower(entry.Name)] > 1 && entry.ID != "" {
			result[index].Name = entry.ID
		}
	}
	return result
}
