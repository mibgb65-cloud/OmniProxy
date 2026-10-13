package agentconfig

import (
	"strings"

	catalog "omniproxy/internal/core/modelcatalog"
)

const subscriptionCompactWindow = 850000

func hasConfirmedContext(entries []catalog.Entry) bool {
	for _, entry := range entries {
		if entry.Metadata.ContextVerified {
			return true
		}
	}
	return false
}

func subscriptionCompactSettings(config map[string]any, entries []catalog.Entry) error {
	if !hasConfirmedContext(entries) {
		return nil
	}
	settings, ok := config["modelSettings"].(map[string]any)
	if config["modelSettings"] != nil && !ok {
		return failure("invalid_config")
	}
	if settings == nil {
		settings = map[string]any{}
	}
	for _, entry := range entries {
		if !entry.Metadata.ContextVerified {
			continue
		}
		alias := catalog.DesktopAlias(entry.ID)
		// 本机官方核心按规范名匹配 [1m]；只设两个基础 ID，避免变体设置覆盖用户已有的基础模型阈值。
		for _, id := range []string{strings.TrimSuffix(entry.ID, catalog.ClaudeLongContextSuffix), strings.TrimSuffix(alias, catalog.ClaudeLongContextSuffix)} {
			model, ok := settings[id].(map[string]any)
			if settings[id] != nil && !ok {
				return failure("invalid_config")
			}
			if model == nil {
				model = map[string]any{}
			}
			limit := float64(subscriptionCompactWindow)
			if value, exists := model["autoCompactWindow"]; exists && value != "auto" {
				prior, ok := value.(float64)
				if !ok || prior < 100000 || prior > 1000000 || prior != float64(int64(prior)) {
					return failure("invalid_config")
				}
				if prior < limit {
					limit = prior
				}
			}
			model["autoCompactWindow"] = limit
			settings[id] = model
		}
	}
	// 真实线路通过 900K 输入、百万输入被拒绝；提前压缩为输出和客户端系统／工具文本留空间。
	config["modelSettings"] = settings
	return nil
}
