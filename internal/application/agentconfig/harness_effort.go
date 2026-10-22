package agentconfig

import (
	catalog "omniproxy/internal/core/modelcatalog"
)

// harnessReasoningLevels 是 Harness Effort 菜单可用的档位顺序。
// Harness 自带 off／low／medium／high，并接受在映射里追加自定义档位（如上游的 max）。
// 键是菜单档位，值是线路上的 reasoning_effort 拼写；网关按各模型已确认枚举复核。
var harnessReasoningLevels = []string{"low", "medium", "high", "max"}

// harnessReasoningEfforts 输出 pi-ai 的 reasoningEfforts 映射。
//
// Harness 官方文档：模型只有声明了 reasoning levels，模型选择器才出现 Effort 菜单；
// 未声明时菜单里没有该项，由端点自身默认值决定是否思考。因此这里只为上游已确认
// 档位枚举的模型输出，未确认的模型由调用方写 false，不凭空宣告能力。
//
// 键是菜单档位，值是线路上发送的 reasoning_effort 拼写。
func harnessReasoningEfforts(entry catalog.Entry) map[string]any {
	// 只有 River 的 Messages／Chat 档位链路已确认可实现，不替其他适配器猜测契约。
	if entry.ProviderKind != "river" {
		return nil
	}
	efforts := map[string]any{}
	for _, level := range harnessReasoningLevels {
		if catalog.ClaudeEffort(entry, level) {
			efforts[level] = level
		}
	}
	// 上游用 none 表示关闭思考。这里必须显式发送 none 而不是留空：留空等于不发参数，
	// 而 River 的 DeepSeek／GLM 在不带参数时默认仍在思考，用户选 off 会无效。
	if catalog.ClaudeEffort(entry, "none") {
		efforts["off"] = "none"
	}
	// 至少要有两个可选项，单个档位的声明没有意义。
	if len(efforts) < 2 {
		return nil
	}
	return efforts
}

// harnessThinkingFormat 返回该模型需要的思考格式兼容开关。
//
// 思考默认开启的上游（River 的 DeepSeek 系与 GLM 系）在没有 reasoning_effort 时仍会思考。
// pi-ai 需要 thinkingFormat: deepseek 才会在档位切换时同步发送 thinking: {type: enabled/disabled}，
// 否则勾掉思考仍可能有隐式思考。只对能取得关闭档位的 River 模型返回。
func harnessThinkingFormat(entry catalog.Entry, efforts map[string]any) string {
	if efforts == nil || entry.ProviderKind != "river" {
		return ""
	}
	if _, hasOff := efforts["off"]; !hasOff {
		return ""
	}
	return "deepseek"
}
