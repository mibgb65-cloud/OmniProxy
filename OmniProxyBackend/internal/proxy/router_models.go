package proxy

import (
	"strings"

	"omniproxy/internal/token"
)

func modelRouteKey(model string) string {
	return strings.ToLower(strings.TrimSpace(model))
}

func providerForModel(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		return token.ProviderAnthropic
	}

	if strings.HasPrefix(model, "mimo-") {
		return token.ProviderXiaomi
	}
	if strings.HasPrefix(model, "deepseek-") {
		return token.ProviderDeepSeek
	}
	if strings.HasPrefix(model, "kimi-") {
		return token.ProviderKimi
	}
	if strings.HasPrefix(model, "glm-") || strings.HasPrefix(model, "zhipu-") {
		return token.ProviderZhipu
	}
	if strings.HasPrefix(model, "minimax-") {
		return token.ProviderMiniMax
	}
	if model == "claude-opus-4-7" || model == "claude-sonnet-4-6" {
		return token.ProviderZo
	}
	if strings.HasPrefix(model, "atria-") {
		return token.ProviderAtria
	}
	return token.ProviderAnthropic
}

func providerForOpenCodeModel(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	if model == "" {
		return token.ProviderOpenAI
	}
	if strings.HasPrefix(model, "mimo-") {
		return token.ProviderXiaomi
	}
	if strings.HasPrefix(model, "deepseek-") {
		return token.ProviderDeepSeek
	}
	if strings.HasPrefix(model, "kimi-") {
		return token.ProviderKimi
	}
	if strings.HasPrefix(model, "glm-") || strings.HasPrefix(model, "zhipu-") {
		return token.ProviderZhipu
	}
	if strings.HasPrefix(model, "minimax-") {
		return token.ProviderMiniMax
	}
	if strings.HasPrefix(model, "auto:") || strings.HasPrefix(model, "tokenrouter:") || strings.HasPrefix(model, "tokenrouter/") {
		return token.ProviderTokenRouter
	}
	if strings.Contains(model, "/") {
		return token.ProviderOpenRouter
	}
	if strings.HasPrefix(model, "atria-") {
		return token.ProviderAtria
	}
	if strings.HasPrefix(model, "custom-") {
		return token.ProviderCustom
	}
	return token.ProviderOpenAI
}

func providerForPiModel(model string) (string, string) {
	provider := providerForOpenCodeModel(model)
	credentialType := ""
	if provider == token.ProviderOpenAI {
		credentialType = token.CredentialTypeAPIKey
	}
	return provider, credentialType
}
