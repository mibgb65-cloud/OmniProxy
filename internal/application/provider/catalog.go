// Package provider manages provider configurations independently of the desktop shell.
package provider

import (
	"omniproxy/internal/provider/atria"
	"omniproxy/internal/provider/kimi"
	"omniproxy/internal/provider/mimo"
	"omniproxy/internal/provider/minimax"
	"omniproxy/internal/provider/oxionai"
	"omniproxy/internal/provider/river"
	"omniproxy/internal/provider/stepfun"
)

// Definition describes a built-in provider kind and its authentication options.
type Definition struct {
	Kind               string   `json:"kind"`
	Name               string   `json:"name"`
	DefaultBaseURL     string   `json:"defaultBaseUrl"`
	Models             []string `json:"models"`
	AuthMethods        []string `json:"authMethods"`
	PlannedAuthMethods []string `json:"plannedAuthMethods"`
	// DiscoveryOnly 表示目录按账号发放，没有可内置的预设模型；添加账号并同步后才有模型。
	DiscoveryOnly bool `json:"discoveryOnly,omitempty"`
}

const (
	AuthAPIKey            = "api_key"
	AuthSubscriptionLogin = "subscription_login"
	KindOpenAICompatible  = "openai-compatible"
	SourceBuiltIn         = "builtin"
	SourceCustom          = "custom"
)

var definitions = []Definition{
	{Kind: "openai", Name: "OpenAI", DefaultBaseURL: "https://api.openai.com/v1", Models: []string{"GPT-5.4", "GPT-5.4 Mini", "GPT-4.1", "o3"}, AuthMethods: []string{AuthAPIKey, AuthSubscriptionLogin}},
	{Kind: "anthropic", Name: "Anthropic", DefaultBaseURL: "https://api.anthropic.com/v1", Models: []string{"Claude Opus 5.5", "Claude Sonnet 5", "Claude Haiku 4.5", "Claude Fable 5.1"}, AuthMethods: []string{AuthAPIKey}},
	{Kind: "google", Name: "Google", DefaultBaseURL: "https://generativelanguage.googleapis.com/v1beta", Models: []string{"Gemini 3.8 Flash", "Gemini 3.7 Flash", "Gemini 3.1 Pro", "Gemini Embedding"}, AuthMethods: []string{AuthAPIKey, AuthSubscriptionLogin}},
	{Kind: "deepseek", Name: "DeepSeek", DefaultBaseURL: "https://api.deepseek.com", Models: []string{"DeepSeek V4.1 Flash", "DeepSeek V4 Pro"}, AuthMethods: []string{AuthAPIKey}},
	{Kind: "zhipu", Name: "Zhipu AI", DefaultBaseURL: "https://open.bigmodel.cn/api/paas/v4", Models: []string{"GLM-5.3", "GLM-5.3-Flash", "GLM-5.2", "GLM-4.7"}, AuthMethods: []string{AuthAPIKey}},
	{Kind: kimi.Kind, Name: "Moonshot AI", DefaultBaseURL: kimi.DefaultBaseURL, Models: kimi.Models(), AuthMethods: []string{AuthAPIKey}},
	{Kind: minimax.Kind, Name: "MiniMax", DefaultBaseURL: minimax.DefaultBaseURL, Models: append([]string{minimax.PreviewModel}, minimax.Models()...), AuthMethods: []string{AuthAPIKey}},
	{Kind: mimo.Kind, Name: "Xiaomi MiMo", DefaultBaseURL: mimo.DefaultBaseURL, Models: mimo.Models(), AuthMethods: []string{AuthAPIKey}},
	{Kind: stepfun.Kind, Name: "StepFun", DefaultBaseURL: stepfun.DefaultBaseURL, Models: stepfun.Models(), AuthMethods: []string{AuthAPIKey}},
	{Kind: atria.Kind, Name: "Atria", DefaultBaseURL: atria.DefaultBaseURL, Models: atria.Models(), AuthMethods: []string{AuthAPIKey}},
	{Kind: river.Kind, Name: "River", DefaultBaseURL: river.DefaultBaseURL, Models: river.DisplayModels(), AuthMethods: []string{AuthAPIKey}},
	{Kind: oxionai.Kind, Name: "Oxion AI", DefaultBaseURL: oxionai.DefaultBaseURL, Models: oxionai.Models(), AuthMethods: []string{AuthAPIKey}, DiscoveryOnly: true},
}

// Catalog returns an independent snapshot of the built-in provider kinds.
func Catalog() []Definition {
	result := make([]Definition, len(definitions))
	for i, definition := range definitions {
		result[i] = definition
		result[i].Models = append([]string(nil), definition.Models...)
		result[i].AuthMethods = append([]string(nil), definition.AuthMethods...)
		result[i].PlannedAuthMethods = append([]string(nil), definition.PlannedAuthMethods...)
	}
	return result
}

func definitionForKind(kind string) (Definition, bool) {
	for _, definition := range definitions {
		if definition.Kind == kind {
			return definition, true
		}
	}
	return Definition{}, false
}

// DefinitionFor 供其他应用服务读取内置提供商的默认端点（例如代理出站的主机匹配）。
func DefinitionFor(kind string) (Definition, bool) { return definitionForKind(kind) }

func knownKind(kind string) bool {
	for _, definition := range definitions {
		if definition.Kind == kind {
			return true
		}
	}
	return false
}
