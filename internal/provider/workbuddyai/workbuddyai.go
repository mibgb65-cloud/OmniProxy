// Package workbuddyai 接入 WorkBuddy AI（腾讯 CodeBuddy 计划的国际版订阅，
// https://www.workbuddy.ai）的浏览器登录态、模型目录、Credits 额度与 Chat 转发。
//
// 协议事实来自社区插件 @magpie-community/opencode-workbuddy-auth 0.2.36
// （MIT，packages/workbuddy/index.mjs）的只读调研，**不是官方契约**，也没有用
// 真实账号在本仓库验证；上游随版本发版（尤其是客户端指纹与业务码）改动后这里
// 可能整体失效。失败一律给固定错误码，不静默降级。
//
// 本包只做固定端点的有界 HTTP 适配：续期的单飞、token 缓存、账号轮换与目录
// 快照都由调用方负责；除登录窗口内的 state 外，客户端不保存任何凭据状态。
package workbuddyai

import (
	"strings"

	catalog "omniproxy/internal/core/modelcatalog"
)

const (
	// Kind 是内置服务商的稳定标识。
	Kind = "workbuddy_ai"
	// Name 是界面展示名称。
	Name = "WorkBuddy AI"
	// DefaultBaseURL 是国际版固定基址；所有端点都相对它追加。
	DefaultBaseURL = "https://www.workbuddy.ai"
	// AuthMethod 是账号表的认证类型；登录令牌不能伪装成 API Key。
	AuthMethod = "workbuddy_login"
	// ChatProtocol 是上游唯一可确认的原生协议；其它协议由网关先转换。
	ChatProtocol = "chat"

	// AppVersion 是上游客户端指纹里的桌面版版本（社区插件 0.2.36 记录，
	// 对应 WorkBuddy AI 5.9.3.42398134）。
	AppVersion = "5.9.3"
	// CLIVersion 是桌面版内置 CLI 的版本；UA 与 X-IDE-Version 都带它。
	CLIVersion = "2.165.0"
)

// staticModel 是插件 AI_MODELS 的一条 fallback 记录。
//
// context 是上游 /v3/config 的 maxInputTokens，output 是 maxOutputTokens；
// 插件留空的位置在这里也是 0（未知，不是 0 容量）。efforts 是插件记录的推理
// 档位（已去掉 "none"）。全部数值来自插件记录，未在本仓库真机验证。
type staticModel struct {
	id      string
	name    string
	context int
	output  int
	efforts []string
}

// fallbackModels 是插件 AI_MODELS 的静态清单（16 条，顺序与插件一致）。
// 只作兜底：账号自己的 /v3/config 才是运行时的权威目录。插件从
// modelPromotions 现算「免费模型」，静态表没有这个信息，因此这里不声明 Free。
var fallbackModels = []staticModel{
	{id: "default-model", name: "Default", context: 176000, output: 24000},
	{id: "fast-model", name: "Fast", context: 200000, output: 32000},
	{id: "balanced-model", name: "Balanced", context: 256000},
	{id: "primary-model", name: "Primary", context: 272000, output: 72000},
	{id: "deep-model", name: "Deep", context: 176000},
	{id: "gpt-5.5", name: "GPT-5.5", context: 1000000},
	{id: "gpt-5.4", name: "GPT-5.4", context: 272000},
	{id: "gpt-5.3-codex", name: "GPT-5.3-Codex", context: 272000},
	{id: "gemini-3.1-pro", name: "Gemini-3.1-Pro", context: 400000},
	{id: "gemini-3.5-flash", name: "Gemini-3.5-Flash", context: 1000000, output: 65536},
	{id: "glm-5.3", name: "GLM-5.3", context: 1000000, efforts: []string{"low", "high", "max"}},
	{id: "glm-5.2", name: "GLM-5.2", context: 1000000, output: 48000, efforts: []string{"high", "xhigh"}},
	{id: "hy3", name: "Hy3", context: 192000, output: 64000, efforts: []string{"low", "high"}},
	{id: "kimi-k3", name: "Kimi-K3", context: 1000000, output: 32000},
	{id: "kimi-k2.6", name: "Kimi-K2.6", context: 256000},
	{id: "minimax-m3", name: "MiniMax-M3", context: 512000},
}

// Models 返回静态 fallback 清单的请求 ID；顺序与插件一致。
func Models() []string {
	models := make([]string, 0, len(fallbackModels))
	for _, model := range fallbackModels {
		models = append(models, model.id)
	}
	return models
}

// DisplayModels 返回与 Models 一一对应的展示名称。
func DisplayModels() []string {
	names := make([]string, 0, len(fallbackModels))
	for _, model := range fallbackModels {
		names = append(names, model.name)
	}
	return names
}

// SupportsModel 只接受静态清单里的 ID，不做模糊匹配。
func SupportsModel(model string) bool {
	_, found := staticModelOf(model)
	return found
}

// ModelName 返回静态清单里的展示名称；未知 ID 原样返回。
func ModelName(model string) string {
	if entry, found := staticModelOf(model); found {
		return entry.name
	}
	return model
}

// ModelMetadata 返回静态清单里可确认的能力：上下文与输出上限、输入模态与推理
// 档位。glm-* 的图片声明按上游实际行为置为 false（配置写 supportsImages=true，
// 但上游会把图片去掉）。工具调用与免费信息不在静态表里，保持未知。
func ModelMetadata(model string) catalog.Metadata {
	entry, found := staticModelOf(model)
	if !found {
		return catalog.Metadata{}
	}
	return metadataOf(entry)
}

func staticModelOf(model string) (staticModel, bool) {
	for _, entry := range fallbackModels {
		if entry.id == model {
			return entry, true
		}
	}
	return staticModel{}, false
}

// metadataOf 只填静态表能证明的字段；空值表示未知。
func metadataOf(entry staticModel) catalog.Metadata {
	result := catalog.Metadata{InputModalities: []string{"text"}}
	if !strings.HasPrefix(entry.id, "glm-") {
		result.InputModalities = []string{"text", "image"}
	}
	if entry.context > 0 && entry.context <= 10000000 {
		value := int64(entry.context)
		result.ContextWindow = &value
	}
	if entry.output > 0 && entry.output <= 10000000 {
		value := int64(entry.output)
		result.OutputTokenLimit = &value
	}
	for _, effort := range entry.efforts {
		result.ReasoningEfforts = append(result.ReasoningEfforts, catalog.Effort{Value: effort})
	}
	return result
}
