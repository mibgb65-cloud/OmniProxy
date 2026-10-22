package agentconfig

import (
	catalog "omniproxy/internal/core/modelcatalog"
)

// desktopDefaultEffortSettings 写入 3p 配置的顶层 defaultModelEffort。
//
// 客户端用它决定默认模型在 Chat／Cowork／Code 的起手档位，而不是 Anthropic 的推荐档位。
// 只写该模型已确认、且客户端接受的档位；未确认时清除旧值，避免移除模型后继续套用旧档位。
func desktopDefaultEffortSettings(config map[string]any, entries []catalog.Entry, defaultModel string) {
	for _, entry := range entries {
		if entry.ID != defaultModel {
			continue
		}
		if level := catalog.DesktopDefaultEffort(entry); level != "" {
			config["defaultModelEffort"] = level
			return
		}
	}
	delete(config, "defaultModelEffort")
}
