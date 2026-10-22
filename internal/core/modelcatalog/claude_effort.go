package modelcatalog

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ClaudeEffortPrefix 标识历史版本派生的档位模型 ID，仅用于兼容已保存的选择。
const ClaudeEffortPrefix = "omniproxy-effort-"

// claudeEffortID 只按历史上线时的算法复算，保证旧选择可被解码；不再用于生成菜单条目。
func claudeEffortID(model, effort string) string {
	base := strings.TrimSuffix(model, ClaudeLongContextSuffix)
	digest := sha256.Sum256([]byte(base + "\x00" + effort))
	id := ClaudeEffortPrefix + hex.EncodeToString(digest[:20])
	if base != model {
		id += ClaudeLongContextSuffix
	}
	return id
}

// ClaudeEffort 校验客户端原生 effort 是否属于该模型已确认的档位枚举。
// 模型不再按档位派生出多行菜单：推理强度由客户端自己的 effort 控件选择。
func ClaudeEffort(entry Entry, effort string) bool {
	for _, candidate := range entry.Metadata.ReasoningEfforts {
		if candidate.Value == effort {
			return true
		}
	}
	return false
}

// archivedEffortMatch 解码历史 omniproxy-effort-* 选择，识别出基础模型与档位。
// 该 ID 不再出现在菜单中，但用户此前保存的选择必须继续可用，不能退回 unsupported_model。
// DesktopAlias 的摘要形式同样接受，避免老会话在客户端归一化后失效。
func archivedEffortMatch(entries []Entry, model string) (Entry, string, bool) {
	base, long := strings.CutSuffix(model, ClaudeLongContextSuffix)
	for _, entry := range entries {
		if entry.ProviderKind != "river" {
			continue
		}
		if long && !supportsLongContext(entry) {
			continue
		}
		for _, effort := range entry.Metadata.ReasoningEfforts {
			id := claudeEffortID(entry.ID, effort.Value)
			if base == id || base == strings.TrimSuffix(DesktopAlias(id), ClaudeLongContextSuffix) {
				return entry, effort.Value, true
			}
		}
	}
	return Entry{}, "", false
}

// ResolveClaudeEffort 只解析历史派生 ID 中已确认的档位，停用或删除目标后立即失效。
func ResolveClaudeEffort(entries []Entry, model string) (string, string, bool) {
	entry, effort, ok := archivedEffortMatch(entries, model)
	if !ok {
		return "", "", false
	}
	return entry.ID, effort, true
}
