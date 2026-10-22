package agentconfig

import (
	"strings"

	catalog "omniproxy/internal/core/modelcatalog"
)

// claudePickerSettings 让模型菜单每个模型只占一行，推理强度交回客户端原生 effort 控件。
//
// behavesAs 让客户端把网关路由名当作它已认识的基础模型：用于上下文窗口、身份与默认档位解析。
// 菜单标签仍是实际模型名，请求中的模型 ID 也不变，不改变身份、计费与展示归属。
// 注意：官方文档只说明 behavesAs 应用「能力与 effort 默认值」，并未声明它能让 effort 控件
// 出现在未被识别的模型上；不要据此断言控件一定出现。
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

// claudeEffortBehavior 是客户端已知的 effort 能力来源模型。
const claudeEffortBehavior = "claude-opus-4-8"

// claudeEffortLevels 是客户端在 modelSettings 中接受的档位。
// 官方文档明确 max 不是 effortLevel／maxEffortLevel 的合法取值，不能写入这两个键。
var claudeEffortLevels = []string{"low", "medium", "high", "xhigh"}

// claudeEffortSettings 为带 behavesAs 的模型写入客户端接受的档位设置。
//
// 只写客户端真正接受的档位：max 既不能作为 effortLevel，也不能作为 maxEffortLevel，
// 写入会让档位解析异常并把实际发送值压回 high（本机 2.1.292 实测）。
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
		defaultLevel := effortDefaultLevel(entry)
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
			if _, exists := model["effortLevel"]; !exists && defaultLevel != "" {
				model["effortLevel"] = defaultLevel
			}
			// 客户端会继承 behavesAs 模型的完整档位序，必须封顶到该模型真实支持的范围；
			// 上限只写客户端接受的取值，写 max 会被忽略并导致档位解析异常。
			if _, exists := model["maxEffortLevel"]; !exists && ceiling != "" {
				model["maxEffortLevel"] = ceiling
			}
			settings[id] = model
		}
	}
	config["modelSettings"] = settings
	return nil
}

// effortDefaultLevel 把上游默认档位映射到客户端接受的取值。
// 上游默认 max 时退到客户端最高合法档位 xhigh；其余已确认档位原样使用。
func effortDefaultLevel(entry catalog.Entry) string {
	for _, level := range claudeEffortLevels {
		if level == entry.Metadata.DefaultEffort && effortSupported(entry, level) {
			return level
		}
	}
	if effortSupported(entry, entry.Metadata.DefaultEffort) {
		// 默认档位是客户端不接受的取值（如 max），用客户端最高合法档位表达。
		return claudeEffortLevels[len(claudeEffortLevels)-1]
	}
	return ""
}

func effortSupported(entry catalog.Entry, effort string) bool {
	return catalog.ClaudeEffort(entry, effort)
}

// effortCeiling 返回客户端档位序中该模型可用的最高档位。
// 只考虑客户端接受的取值；模型只支持 max 时没有可写上限，返回空。
func effortCeiling(entry catalog.Entry) string {
	ceiling := ""
	for _, level := range claudeEffortLevels {
		if catalog.ClaudeEffort(entry, level) {
			ceiling = level
		}
	}
	return ceiling
}
