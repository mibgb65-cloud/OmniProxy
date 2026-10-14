// Package mimo 提供小米 MiMo 按量 API 的原生协议适配器。
package mimo

import catalog "omniproxy/internal/core/modelcatalog"

const (
	// Kind 是已有内置提供商的标识，不新建重复提供商。
	Kind = "mimo"
	// DefaultBaseURL 是普通 API Key 的官方入口，Token Plan 使用独立入口。
	DefaultBaseURL  = "https://api.xiaomimimo.com/v1"
	maxOutputTokens = 131072
)

// Models 返回官方对话模型预设，不包含独立 ASR／TTS 模型。
func Models() []string {
	return []string{"mimo-v2.6-pro", "mimo-v2.6-flash", "mimo-v2.6-pro-ultraspeed", "mimo-v2.5-pro", "mimo-v2.5"}
}

// SupportsModel 限定已核对原生对话协议的模型，不能把语音模型误用于客户端路由。
func SupportsModel(model string) bool {
	for _, id := range Models() {
		if id == model {
			return true
		}
	}
	return false
}

// ModelMetadata 保存官方 1M 上下文、输出上限和工具能力；本版只公布已支持的文本输入。
// 来源：https://mimo.mi.com/models/zh-CN/mimo-v2.6-pro 与官方 Chat／Responses 接口文档。
func ModelMetadata(model string) catalog.Metadata {
	if !SupportsModel(model) {
		return catalog.Metadata{}
	}
	window, output, tools := int64(1000000), int64(maxOutputTokens), true
	return catalog.Metadata{ContextWindow: &window, OutputTokenLimit: &output, InputModalities: []string{"text"}, ToolUse: &tools,
		ReasoningEfforts: []catalog.Effort{{Value: "none", Description: "关闭思考"}, {Value: "medium", Description: "启用思考；官方暂不区分推理强度"}}, DefaultEffort: "medium"}
}
