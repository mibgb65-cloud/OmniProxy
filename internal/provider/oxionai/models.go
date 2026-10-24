// Package oxionai 提供 Oxion AI 官方 Chat Completions 的有界 HTTP 适配器。
//
// 依据官方文档（https://oxionai.com/docs）：Base URL 为 https://oxionai.com/v1，
// 只提供 OpenAI 兼容的 /chat/completions 与 /models，没有 Responses／Messages 原生协议，
// 也没有 reasoning_effort 参数；思考内容通过 content 之外的 reasoning_content 返回。
package oxionai

import catalog "omniproxy/internal/core/modelcatalog"

const (
	// Kind 是提供商内部标识，与 migration 043 的 providers.kind 一致。
	Kind = "oxionai"
	// DefaultBaseURL 已包含 /v1 版本前缀，请求路径不能再追加 /v1。
	DefaultBaseURL = "https://oxionai.com/v1"
	// KeyPrefix 是官方文档示例中的 API Key 前缀，仅用于表单提示，不作强制校验。
	KeyPrefix = "cc_"
	// maxCatalogBytes 限制模型目录正文，避免异常上游拖垮解析。
	maxCatalogBytes = 1 << 20
	// maxCatalogModels 与 core/modelcatalog 的上限保持一致。
	maxCatalogModels = 1000
)

// Models 不内置固定清单：官方目录按账号发放，只能通过 GET /v1/models 同步得到，
// 因此这里返回空切片，不猜测模型 ID。
func Models() []string { return []string{} }

// ModelMetadata 只对已同步模型提供兜底空值：上下文窗口与能力必须来自官方目录。
func ModelMetadata(string) catalog.Metadata { return catalog.Metadata{} }
