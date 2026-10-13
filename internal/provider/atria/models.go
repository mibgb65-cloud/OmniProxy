// Package atria 提供 Atria 官方文本模型的原生 HTTP 适配器。
package atria

import catalog "omniproxy/internal/core/modelcatalog"

const (
	// Kind 是提供商内部标识。
	Kind = "atria"
	// ModelID 保留官方模型标识的大小写。
	ModelID = "Atria-Dawn-Preview"
	// DefaultBaseURL 包含版本前缀，协议路径不能再次追加 /v1。
	DefaultBaseURL  = "https://api.atria-asi.ai/v1"
	maxOutputTokens = 65536
)

// Models 返回官方预设，不代表账号认证或额度已经验证。
func Models() []string { return []string{ModelID} }

// ModelMetadata 仅记录官方确认的上下文和输入模态。
// 来源：https://api.atria-asi.ai/docs；推理档位和具体工具能力保持未知。
func ModelMetadata(model string) catalog.Metadata {
	if model != ModelID {
		return catalog.Metadata{}
	}
	window := int64(256000)
	return catalog.Metadata{ContextWindow: &window, InputModalities: []string{"text"}}
}
