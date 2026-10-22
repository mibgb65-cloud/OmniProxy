package modelcatalog

import (
	"crypto/sha256"
	"math/big"
	"strings"
)

const DesktopAliasPrefix = "claude-omniproxy-"

// DesktopAlias 为已校验的目录模型生成稳定路由名，客户端标签继续显示实际模型名。
func DesktopAlias(modelID string) string {
	if strings.HasSuffix(modelID, ClaudeLongContextSuffix) {
		// Desktop 会去掉后缀并合并同基础 ID 的选项；长上下文必须有独立基础 ID。
		return desktopDigest(modelID) + ClaudeLongContextSuffix
	}
	return desktopDigest(modelID)
}

func desktopDigest(modelID string) string {
	digest := sha256.Sum256([]byte(modelID))
	// 客户端拒绝路由名中出现其他厂商名称；十进制摘要不会误命中名称过滤。
	// 固定宽度也避免客户端按名称长度选择默认模型时改变目录顺序。
	value := new(big.Int).SetBytes(digest[:]).Text(10)
	return DesktopAliasPrefix + strings.Repeat("0", 78-len(value)) + value
}

// DesktopEffortLevels 是客户端 inferenceModels[].maxEffort 接受的档位，由低到高。
// 取值域与 Claude Code 的 effortLevel／maxEffortLevel 不同：这里允许 max。
var DesktopEffortLevels = []string{"low", "medium", "high", "xhigh", "max"}

// DesktopMaxEffort 返回该模型在客户端档位序中已确认的最高档位。
// 只做已确认枚举与客户端档位序的交集；未确认能力的模型返回空，不写上限。
func DesktopMaxEffort(entry Entry) string {
	ceiling := ""
	for _, level := range DesktopEffortLevels {
		if ClaudeEffort(entry, level) {
			ceiling = level
		}
	}
	return ceiling
}

// DesktopDefaultEffort 只返回客户端接受的、上游公布的默认档位。
func DesktopDefaultEffort(entry Entry) string {
	for _, level := range DesktopEffortLevels {
		if level == entry.Metadata.DefaultEffort {
			return level
		}
	}
	return ""
}

// DesktopModel 使用客户端原生长上下文标志，Code 菜单和实际 CLI 模型选择保持一致。
type DesktopModel struct {
	Name          string `json:"name"`
	LabelOverride string `json:"labelOverride"`
	Supports1m    bool   `json:"supports1m,omitempty"`
	Prefer1m      bool   `json:"prefer1m,omitempty"`
	MaxEffort     string `json:"maxEffort,omitempty"`
}

// DesktopModels 为普通和长上下文生成独立选项，客户端归一化后仍可区分。
func DesktopModels(entries []Entry, defaultModel string) []DesktopModel {
	models := make([]DesktopModel, 0, len(entries))
	for _, entry := range entries {
		long := strings.HasSuffix(entry.ID, ClaudeLongContextSuffix) && supportsLongContext(entry)
		item := DesktopModel{Name: DesktopAlias(entry.ID), LabelOverride: entry.Name, Supports1m: long, Prefer1m: long, MaxEffort: DesktopMaxEffort(entry)}
		if entry.ID == defaultModel {
			models = append([]DesktopModel{item}, models...)
		} else {
			models = append(models, item)
		}
	}
	return models
}

// ResolveDesktopAlias 只解析当前目录，模型或路由移除后旧客户端配置不能继续使用。
func ResolveDesktopAlias(entries []Entry, alias string) (string, bool) {
	if !strings.HasPrefix(alias, DesktopAliasPrefix) {
		return "", false
	}
	return ResolveClaudeModel(entries, alias)
}
