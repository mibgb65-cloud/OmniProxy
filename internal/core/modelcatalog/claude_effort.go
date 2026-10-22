package modelcatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ClaudeEffortPrefix 表示网关生成的明确档位选项，不伪装真实厂商型号。
const ClaudeEffortPrefix = "omniproxy-effort-"

func claudeEffortID(model, effort string) string {
	base := strings.TrimSuffix(model, ClaudeLongContextSuffix)
	digest := sha256.Sum256([]byte(base + "\x00" + effort))
	id := ClaudeEffortPrefix + hex.EncodeToString(digest[:20])
	if base != model {
		id += ClaudeLongContextSuffix
	}
	return id
}

func claudeEffortEntries(entries []Entry) []Entry {
	result := []Entry{}
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.ID] = true
	}
	for _, entry := range entries {
		// 目前只有 River 的 Messages 档位转换已实现；不替其他适配器猜测契约。
		if entry.ProviderKind != "river" {
			continue
		}
		for _, effort := range entry.Metadata.ReasoningEfforts {
			id := claudeEffortID(entry.ID, effort.Value)
			if seen[id] {
				continue
			}
			variant := entry
			variant.ID = id
			label := effort.Description
			if label == "" {
				label = effort.Value
			}
			variant.Name = entry.Name + " · " + label
			if !textValid(variant.Name, 200) {
				variant.Name = id
			}
			variant.Metadata.IsDefault = false
			variant.Metadata.ReasoningEfforts = nil
			variant.Metadata.DefaultEffort = ""
			result = append(result, variant)
			seen[id] = true
		}
	}
	return result
}

// ResolveClaudeEffort 只解析当前可用目录生成的选项，停用或删除目标后立即失效。
func ResolveClaudeEffort(entries []Entry, model string) (string, string, bool) {
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.ID] = true
	}
	for _, entry := range entries {
		if entry.ProviderKind != "river" {
			continue
		}
		for _, effort := range entry.Metadata.ReasoningEfforts {
			id := claudeEffortID(entry.ID, effort.Value)
			if seen[id] {
				continue
			}
			if _, ok := ResolveClaudeModel([]Entry{{ID: id, Metadata: entry.Metadata}}, model); ok {
				return entry.ID, effort.Value, true
			}
		}
	}
	return "", "", false
}
