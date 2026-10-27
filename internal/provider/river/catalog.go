// Package river 接入 River 官方 OpenAI 兼容推理服务。
package river

import catalog "omniproxy/internal/core/modelcatalog"

const (
	// Kind 是 River 内置服务商的稳定标识。
	Kind = "river"
	// DefaultBaseURL 是官方推理入口。
	DefaultBaseURL = "https://api.river.ai/v1"
	// Qwen 是官方 Qwen 型号；FP8 是官方模型名的一部分，不能去掉。
	Qwen = "Qwen/Qwen3.8-27B-FP8"
	// DeepSeekFlash 保留官方仓库前缀和大小写，避免与直连 DeepSeek 混淆。
	DeepSeekFlash = "deepseek-ai/DeepSeek-V4.1-Flash"
	// DeepSeekFlash1017 是独立发布的固定版本，不能折叠到原版模型。
	DeepSeekFlash1017 = "deepseek-ai/DeepSeek-V4.1-Flash-1017"
	// KimiK3 是官方 Moonshot 型号，与直连 Kimi 的 kimi-k3 是不同计费入口。
	KimiK3 = "moonshotai/Kimi-K3"
	// NemotronVL 是官方多模态 Nemotron 模型。
	NemotronVL = "nvidia/NVIDIA-Nemotron-3.5-Super-VL-120B-A12B-NVFP4"
	// GLMFlash 是官方 GLM 模型。
	GLMFlash = "zai-org/GLM-5.3-Flash"
)

// Models 返回官方预设（顺序与官方推理模型页一致），不代表 Key 已通过认证。
func Models() []string {
	return []string{Qwen, DeepSeekFlash, DeepSeekFlash1017, KimiK3, NemotronVL, GLMFlash}
}

// SupportsModel 仅开放已经核对协议的官方模型。
func SupportsModel(model string) bool {
	for _, supported := range Models() {
		if model == supported {
			return true
		}
	}
	return false
}

// ModelMetadata 记录官方上下文和能力，不推断未公布的推理档位。
// 来源：https://docs.river.ai/guides/inference-models/（能力与上下限）、
// https://docs.river.ai/guides/inference-controls/（推理档位）。
func ModelMetadata(model string) catalog.Metadata {
	if !SupportsModel(model) {
		return catalog.Metadata{}
	}
	window, output, tools := int64(1048576), int64(131072), true
	var efforts []catalog.Effort
	description, defaultEffort := "", ""
	switch model {
	case Qwen:
		// 官方能力表：262,144 上下文、262,143 输出；档位为 none／low／medium／xhigh。
		window, output = 262144, 262143
		efforts = []catalog.Effort{{Value: "none", Description: "关闭思考"}, {Value: "low", Description: "低强度推理"}, {Value: "medium", Description: "中强度推理"}, {Value: "xhigh", Description: "超高强度推理"}}
	case DeepSeekFlash, DeepSeekFlash1017:
		output = 384000
		// 1017 于 2026-10-22 以有界合成请求确认三档接受、非法档位拒绝；不再沿用未知能力快照。
		efforts = []catalog.Effort{{Value: "none", Description: "关闭思考"}, {Value: "high", Description: "高强度推理"}, {Value: "max", Description: "最高强度推理"}}
		defaultEffort = "high"
	case KimiK3:
		// 官方：标准档位只开关思考，默认 max；low／high／max 分层需要 chat_template_kwargs，
		// 本适配器不发送该扩展，因此只发布官方标准字段能表达的取值。
		output = 32768
		efforts = []catalog.Effort{{Value: "none", Description: "关闭思考"}, {Value: "max", Description: "开启推理（官方默认）"}}
		defaultEffort = "max"
		description = "官方分层档位 low／high／max 由 chat_template_kwargs.thinking_effort 选择，本适配器只发送标准 reasoning 字段"
	case GLMFlash:
		efforts = []catalog.Effort{{Value: "low", Description: "低强度推理"}, {Value: "high", Description: "高强度推理"}, {Value: "max", Description: "最高强度推理"}}
		defaultEffort = "high"
	}
	// 官方能力表声明六款模型均支持文本与图片输入，适配器已按 image_url 块转发。
	return catalog.Metadata{
		Description:      description,
		ContextWindow:    &window,
		OutputTokenLimit: &output,
		ReasoningEfforts: efforts,
		DefaultEffort:    defaultEffort,
		InputModalities:  []string{"text", "image"},
		ToolUse:          &tools,
	}
}
