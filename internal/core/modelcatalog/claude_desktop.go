package modelcatalog

import (
	"crypto/sha256"
	"math/big"
	"strings"
)

const DesktopAliasPrefix = "claude-omniproxy-"

// DesktopAlias 为已校验的目录模型生成稳定路由名，客户端标签继续显示实际模型名。
func DesktopAlias(modelID string) string {
	digest := sha256.Sum256([]byte(modelID))
	// 客户端拒绝路由名中出现其他厂商名称；十进制摘要不会误命中名称过滤。
	// 固定宽度也避免客户端按名称长度选择默认模型时改变目录顺序。
	value := new(big.Int).SetBytes(digest[:]).Text(10)
	return DesktopAliasPrefix + strings.Repeat("0", 78-len(value)) + value
}

// ResolveDesktopAlias 只解析当前目录，模型或路由移除后旧客户端配置不能继续使用。
func ResolveDesktopAlias(entries []Entry, alias string) (string, bool) {
	if !strings.HasPrefix(alias, DesktopAliasPrefix) {
		return "", false
	}
	for _, entry := range entries {
		// 保留目录中既有的同名模型／用户路由，不改变已公开 ID 的含义。
		if entry.ID == alias {
			return entry.ID, true
		}
	}
	for _, entry := range entries {
		if DesktopAlias(entry.ID) == alias {
			return entry.ID, true
		}
	}
	return "", false
}
