// Package river 接入 River 官方 OpenAI 兼容推理服务。
package river

import catalog "omniproxy/internal/core/modelcatalog"

const (
	// Kind 是 River 内置服务商的稳定标识。
	Kind = "river"
	// DefaultBaseURL 是官方推理入口。
	DefaultBaseURL = "https://api.river.ai/v1"
	// DeepSeekFlash 保留官方仓库前缀和大小写，避免与直连 DeepSeek 混淆。
	DeepSeekFlash = "deepseek-ai/DeepSeek-V4.1-Flash"
	// DeepSeekFlash1017 是独立发布的固定版本，不能折叠到原版模型。
	DeepSeekFlash1017 = "deepseek-ai/DeepSeek-V4.1-Flash-1017"
	// NemotronVL 是官方多模态 Nemotron 模型。
	NemotronVL = "nvidia/NVIDIA-Nemotron-3.5-Super-VL-120B-A12B-NVFP4"
	// GLMFlash 是官方 GLM 模型。
	GLMFlash = "zai-org/GLM-5.3-Flash"
)

// Models 返回官方预设，不代表 Key 已通过认证。
func Models() []string { return []string{DeepSeekFlash, DeepSeekFlash1017, NemotronVL, GLMFlash} }

// SupportsModel 仅开放已经核对协议的官方模型。
func SupportsModel(model string) bool {
	return model == DeepSeekFlash || model == DeepSeekFlash1017 || model == NemotronVL || model == GLMFlash
}

// ModelMetadata 记录官方上下文和能力，不推断未公布的 Nemotron 推理档位。
// 来源：https://docs.river.ai/guides/inference-models/ 与 inference-controls/。
func ModelMetadata(model string) catalog.Metadata {
	if !SupportsModel(model) {
		return catalog.Metadata{}
	}
	window, output, tools := int64(1048576), int64(131072), true
	if model == DeepSeekFlash || model == DeepSeekFlash1017 {
		output = 384000
	}
	m := catalog.Metadata{ContextWindow: &window, OutputTokenLimit: &output, InputModalities: []string{"text"}, ToolUse: &tools}
	// 1017 于 2026-10-22 以有界合成请求确认三档接受、非法档位拒绝；不再沿用未知能力快照。
	if model == DeepSeekFlash || model == DeepSeekFlash1017 {
		m.ReasoningEfforts = []catalog.Effort{{Value: "none", Description: "关闭思考"}, {Value: "high", Description: "高强度推理"}, {Value: "max", Description: "最高强度推理"}}
		m.DefaultEffort = "high"
	}
	if model == GLMFlash {
		m.ReasoningEfforts = []catalog.Effort{{Value: "low", Description: "低强度推理"}, {Value: "high", Description: "高强度推理"}, {Value: "max", Description: "最高强度推理"}}
		m.DefaultEffort = "high"
	}
	return m
}
