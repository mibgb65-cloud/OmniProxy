package agentconfig

import (
	"strings"

	catalog "omniproxy/internal/core/modelcatalog"
)

// claudePickerSettings 让模型菜单每个模型只占一行，推理强度交回客户端原生 effort 控件。
//
// behavesAs 把网关路由名映射到客户端已认识的基础模型，客户端据此显示 effort 控件并送出
// output_config.effort；菜单标签仍是实际模型名，请求中的模型 ID 也不变。
// 不把第三方模型伪装成 Claude：这里只借用 effort／thinking 能力识别，不改变身份与展示名。
func claudePickerSettings(config map[string]any, entries []catalog.Entry) {
	options := make([]map[string]string, 0, len(entries))
	for _, entry := range entries {
		option := map[string]string{"model": entry.ID, "label": entry.Name}
		if behavior := effortBehavesAs(entry); behavior != "" {
			option["behavesAs"] = behavior
		}
		options = append(options, option)
	}
	// 显式替换内置列表，避免残留的档位行和不可路由的官方模型。
	config["modelPicker"] = map[string]any{"options": options, "replaceBuiltInOptions": true}
}

// effortBehavesAs 只对已确认 effort 枚举和默认档位的模型声明能力继承。
// 未确认能力的模型保持未知，不借其他模型的能力凭空显示 effort 控件。
func effortBehavesAs(entry catalog.Entry) string {
	if entry.ProviderKind != "river" || len(entry.Metadata.ReasoningEfforts) == 0 || entry.Metadata.DefaultEffort == "" {
		return ""
	}
	// 基础模型路由名是十进制摘要，不含厂商名，可安全映射到已知 Claude 模型。
	return claudeEffortBehavior
}

// claudeEffortBehavior 是客户端已知的 effort 能力来源模型；它的档位枚举
// （low／medium／high／xhigh／max）由网关按各模型真实枚举再次校验，不会透传非法档位。
const claudeEffortBehavior = "claude-opus-4-8"

// claudeEffortRange 是客户端为 behavesAs 模型提供的档位序；网关只接受各模型真实枚举的子集。
var claudeEffortRange = []string{"low", "medium", "high", "xhigh", "max"}

// claudeEffortSettings 为带 behavesAs 的模型写入客户端认识的收窄设置：
// 默认档位取上游公布的默认值，上限不超过该模型真实支持的最高档位。
func claudeEffortSettings(config map[string]any, entries []catalog.Entry) error {
	settings, ok := config["modelSettings"].(map[string]any)
	if config["modelSettings"] != nil && !ok {
		return failure("invalid_config")
	}
	if settings == nil {
		settings = map[string]any{}
	}
	for _, entry := range entries {
		if effortBehavesAs(entry) == "" {
			continue
		}
		ceiling := effortCeiling(entry)
		for _, id := range []string{strings.TrimSuffix(entry.ID, catalog.ClaudeLongContextSuffix), strings.TrimSuffix(catalog.DesktopAlias(entry.ID), catalog.ClaudeLongContextSuffix)} {
			model, ok := settings[id].(map[string]any)
			if settings[id] != nil && !ok {
				return failure("invalid_config")
			}
			if model == nil {
				model = map[string]any{}
			}
			// 用户已显式保存的档位优先，不覆盖。
			if _, exists := model["effortLevel"]; !exists && effortSupported(entry, entry.Metadata.DefaultEffort) {
				model["effortLevel"] = entry.Metadata.DefaultEffort
			}
			// 客户端会继承 behavesAs 模型的完整档位序，必须封顶到该模型真实支持的范围。
			if _, exists := model["maxEffortLevel"]; !exists && ceiling != "" {
				model["maxEffortLevel"] = ceiling
			}
			settings[id] = model
		}
	}
	config["modelSettings"] = settings
	return nil
}

func effortSupported(entry catalog.Entry, effort string) bool {
	return catalog.ClaudeEffort(entry, effort)
}

// effortCeiling 返回客户端档位序中该模型可用的最高档位。
func effortCeiling(entry catalog.Entry) string {
	ceiling := ""
	for _, level := range claudeEffortRange {
		if catalog.ClaudeEffort(entry, level) {
			ceiling = level
		}
	}
	return ceiling
}
