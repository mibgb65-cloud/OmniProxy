package routing

import (
	"omniproxy/internal/provider/atria"
	"omniproxy/internal/provider/deepseek"
)

// presetModels 只提供已有适配器支持的官方预设，不代表实时账号发现。
func presetModels(kind string) []string {
	switch kind {
	case "deepseek":
		return deepseek.Models()
	case atria.Kind:
		return atria.Models()
	default:
		return nil
	}
}

func supportedProvider(kind string) bool {
	return kind == "deepseek" || kind == "openai" || kind == atria.Kind || kind == "google"
}

func hasPresetAccount(data snapshot, providerID string) bool {
	for _, item := range data.accounts {
		if item.ProviderID == providerID && item.AuthMethod == "api_key" {
			return true
		}
	}
	return false
}
