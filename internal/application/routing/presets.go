package routing

import (
	"omniproxy/internal/provider/atria"
	"omniproxy/internal/provider/deepseek"
	"omniproxy/internal/provider/dumate"
	"omniproxy/internal/provider/howtofix"
	"omniproxy/internal/provider/mimo"
	"omniproxy/internal/provider/oxionai"
	"omniproxy/internal/provider/river"
	"omniproxy/internal/provider/workbuddyai"
)

// presetModels 只提供已有适配器支持的官方预设，不代表实时账号发现。
func presetModels(kind string) []string {
	switch kind {
	case "deepseek":
		return deepseek.Models()
	case atria.Kind:
		return atria.Models()
	case river.Kind:
		return river.Models()
	case mimo.Kind:
		return mimo.Models()
	case oxionai.Kind:
		// Oxion AI 的目录按账号发放，没有任何可内置的预设模型。
		return oxionai.Models()
	case howtofix.Kind:
		// HowToFix 的目录按账号分组发放，同样没有可内置的预设模型。
		return howtofix.Models()
	case dumate.Kind:
		// DuMate 上游没有 /models，只公布参考实现确认的两条别名。
		return dumate.Models()
	case workbuddyai.Kind, workbuddyai.ChinaKind:
		// WorkBuddy 的目录按账号的 /v3/config 发放；同步前没有任何可路由模型。
		return nil
	default:
		return nil
	}
}

func supportedProvider(kind string) bool {
	return kind == river.Kind || kind == "minimax" || kind == "kimi" || kind == "stepfun" || kind == "deepseek" || kind == "openai" || kind == atria.Kind || kind == "google" || kind == mimo.Kind || kind == oxionai.Kind || kind == howtofix.Kind || kind == dumate.Kind || kind == workbuddyai.Kind || kind == workbuddyai.ChinaKind
}

func hasPresetAccount(data snapshot, providerID string) bool {
	for _, item := range data.accounts {
		if item.ProviderID != providerID {
			continue
		}
		if item.AuthMethod == "api_key" || item.AuthMethod == dumate.AuthMethod || item.AuthMethod == workbuddyai.AuthMethod {
			return true
		}
	}
	return false
}
