package modelcatalog

import (
	"sort"
	"strings"
)

// 上游用独立模型 ID 表达推理档位时（例如 Google 的 “… (High)／(Low)／(Medium)”），
// 客户端看到的是多个“模型”。这里把变体折叠回基础模型：目录只发布一行，
// 档位由客户端原生 effort 控件选择，网关在请求时换成实际存在的变体 ID。
var variantEffortOrder = []string{"minimal", "low", "medium", "high", "xhigh", "ultra"}

// variantSuffixes 按长度无关的固定顺序匹配，避免依赖 map 迭代顺序。
var variantSuffixes = []struct {
	suffix string
	level  string
}{
	{" (minimal)", "minimal"},
	{" (low)", "low"},
	{" (medium)", "medium"},
	{" (high)", "high"},
	{" (xhigh)", "xhigh"},
	{" (ultra)", "ultra"},
	{"（低）", "low"},
	{"（中）", "medium"},
	{"（高）", "high"},
}

// FoldEffortVariants 把同名档位变体折叠进基础模型，变体条目不再单独发布。
//
// 只有同时存在基础模型时才会折叠：只有变体、没有基础模型的型号保持原样，
// 避免把不同型号误合并。折叠后基础模型的 EffortModels 保留档位到变体 ID 的映射，
// 已保存的旧选择仍可按 ID 解析。
func FoldEffortVariants(entries []Entry) []Entry {
	byName := make(map[string]int, len(entries))
	for index, entry := range entries {
		byName[strings.ToLower(entry.Name)] = index
	}
	folded := make([]Entry, len(entries))
	copy(folded, entries)
	removed := make([]bool, len(entries))
	for index := range folded {
		effort, base := splitEffortVariant(folded[index].Name)
		if effort == "" {
			continue
		}
		target, ok := byName[strings.ToLower(base)]
		if !ok || target == index || removed[target] {
			continue
		}
		candidate := &folded[target]
		if candidate.Metadata.EffortModels == nil {
			candidate.Metadata.EffortModels = map[string]string{}
		}
		// 同一档位出现两次时保留首个，不覆盖已确认的映射。
		if _, exists := candidate.Metadata.EffortModels[effort]; !exists {
			candidate.Metadata.EffortModels[effort] = folded[index].ID
		}
		removed[index] = true
	}
	result := make([]Entry, 0, len(folded))
	for index, entry := range folded {
		if removed[index] {
			continue
		}
		if len(entry.Metadata.EffortModels) > 0 {
			entry.Metadata.ReasoningEfforts = variantEfforts(entry.Metadata.EffortModels)
			if entry.Metadata.DefaultEffort == "" {
				entry.Metadata.DefaultEffort = defaultVariantEffort(entry.Metadata.EffortModels)
			}
		}
		result = append(result, entry)
	}
	return result
}

// splitEffortVariant 拆出 “模型名 (High)” 形式的档位与基础名称。
func splitEffortVariant(name string) (effort, base string) {
	lower := strings.ToLower(name)
	for _, item := range variantSuffixes {
		if !strings.HasSuffix(lower, strings.ToLower(item.suffix)) {
			continue
		}
		trimmed := strings.TrimSpace(name[:len(name)-len(item.suffix)])
		if trimmed == "" {
			return "", ""
		}
		return item.level, trimmed
	}
	return "", ""
}

// variantEfforts 按固定档位顺序生成枚举；变体本身不带档位说明，不臆造描述。
func variantEfforts(models map[string]string) []Effort {
	result := make([]Effort, 0, len(models))
	for _, level := range variantEffortOrder {
		if _, ok := models[level]; ok {
			result = append(result, Effort{Value: level})
		}
	}
	return result
}

// defaultVariantEffort 只在高档已确认时给出默认档位，其余交给客户端默认。
func defaultVariantEffort(models map[string]string) string {
	if _, ok := models["high"]; ok {
		return "high"
	}
	return ""
}

// EffortModel 返回该档位对应的上游变体模型 ID。
func EffortModel(entry Entry, level string) (string, bool) {
	model, ok := entry.Metadata.EffortModels[level]
	return model, ok && model != ""
}

// EffortLevels 返回折叠后已确认的档位，供请求期校验。
func EffortLevels(entry Entry) []string {
	levels := make([]string, 0, len(entry.Metadata.EffortModels))
	for level := range entry.Metadata.EffortModels {
		levels = append(levels, level)
	}
	sort.Slice(levels, func(i, j int) bool { return variantRank(levels[i]) < variantRank(levels[j]) })
	return levels
}

// EffortVariantBase 把旧的档位变体 ID（例如 “…-low”）还原为基础模型与档位。
// 目录折叠后变体不再单独发布，历史选择必须继续可用。
func EffortVariantBase(entries []Entry, model string) (Entry, string, bool) {
	for _, entry := range entries {
		for level, variant := range entry.Metadata.EffortModels {
			if variant == model {
				return entry, level, true
			}
		}
	}
	return Entry{}, "", false
}

func variantRank(level string) int {
	for index, candidate := range variantEffortOrder {
		if candidate == level {
			return index
		}
	}
	return len(variantEffortOrder)
}
