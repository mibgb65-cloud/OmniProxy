// Package howtofix 提供 HowToFix 官方 OpenAI 兼容网关的有界 HTTP 适配器。
//
// 依据官方 handbook（https://howtofix.id/api/handbook/docs，读取日期 2026-11-15）：
// Base URL 为 https://api.howtofix.id/v1，提供 OpenAI 兼容的 /chat/completions 与 /models，
// 使用 Authorization: Bearer sk-... 认证；余额可由 /dashboard/billing/subscription 与
// /dashboard/billing/usage 读取。handbook 另称 Anthropic／Gemini 路由为原生直出，
// 但本适配器在真实 Key 验证前只按 Chat 转发，Responses／Messages 由网关既有转换链路处理。
package howtofix

import catalog "omniproxy/internal/core/modelcatalog"

const (
	// Kind 是提供商内部标识，与 migration 049 的 providers.kind 一致。
	Kind = "howtofix"
	// DefaultBaseURL 已包含 /v1 版本前缀，请求路径不能再追加 /v1。
	DefaultBaseURL = "https://api.howtofix.id/v1"
	// KeyPrefix 是官方文档示例中的 API Key 前缀，仅用于表单提示，不作强制校验。
	KeyPrefix = "sk-"
	// maxCatalogBytes 限制模型目录正文，避免异常上游拖垮解析。
	maxCatalogBytes = 1 << 20
	// maxCatalogModels 与 core/modelcatalog 的上限保持一致。
	maxCatalogModels = 1000
	// maxBalanceBytes 限制余额正文；两个余额接口都只是少量 JSON。
	maxBalanceBytes = 64 << 10
	// maxDisplayAmount 限制余额数值文本长度，防止异常上游输出超长数字。
	maxDisplayAmount = 32
)

// Models 不内置固定清单：目录按账号所属分组发放（例如 vip 专属模型），
// 只能通过 GET /v1/models 同步得到，因此这里返回空切片，不猜测模型 ID。
func Models() []string { return []string{} }

// ModelMetadata 只对已同步模型提供兜底空值：上下文窗口与能力必须来自官方目录。
func ModelMetadata(string) catalog.Metadata { return catalog.Metadata{} }
