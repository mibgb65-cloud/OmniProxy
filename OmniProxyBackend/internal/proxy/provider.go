package proxy

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"

	"omniproxy/internal/config"
	"omniproxy/internal/token"
)

type providerURLField struct {
	Name  string
	Value string
}

func proxyBaseURLFields(cfg config.Config) []providerURLField {
	cfg = config.Normalize(cfg)
	return []providerURLField{
		{Name: "upstream", Value: cfg.UpstreamBaseURL},
		{Name: token.ProviderOpenAI, Value: cfg.OpenAIBaseURL},
		{Name: token.ProviderAnthropic, Value: cfg.AnthropicBaseURL},
		{Name: token.ProviderDeepSeek, Value: cfg.DeepSeekBaseURL},
		{Name: "deepseek_anthropic", Value: cfg.DeepSeekAnthropicBaseURL},
		{Name: token.ProviderKimi, Value: cfg.KimiBaseURL},
		{Name: token.ProviderZhipu, Value: cfg.ZhipuBaseURL},
		{Name: "zhipu_anthropic", Value: cfg.ZhipuAnthropicBaseURL},
		{Name: token.ProviderMiniMax, Value: cfg.MiniMaxBaseURL},
		{Name: "minimax_anthropic", Value: cfg.MiniMaxAnthropicBaseURL},
		{Name: token.ProviderGemini, Value: cfg.GeminiBaseURL},
		{Name: token.ProviderOpenRouter, Value: cfg.OpenRouterBaseURL},
		{Name: token.ProviderFeatherless, Value: cfg.FeatherlessBaseURL},
		{Name: token.ProviderTokenRouter, Value: cfg.TokenRouterBaseURL},
		{Name: token.ProviderSub2API, Value: cfg.Sub2APIBaseURL},
		{Name: token.ProviderNewAPI, Value: cfg.NewAPIBaseURL},
		{Name: token.ProviderAnyRouter, Value: cfg.AnyRouterBaseURL},
		{Name: token.ProviderZo, Value: cfg.ZoBaseURL},
		{Name: token.ProviderPrem, Value: cfg.PremBaseURL},
		{Name: token.ProviderAtria, Value: cfg.AtriaBaseURL},
		{Name: "custom_gateway", Value: cfg.CustomGatewayBaseURL},
		{Name: "custom_gateway_anthropic", Value: cfg.CustomGatewayAnthropicBaseURL},
		{Name: "xiaomi_api", Value: cfg.XiaomiAPIBaseURL},
		{Name: "xiaomi_api_anthropic", Value: cfg.XiaomiAPIAnthropicBaseURL},
		{Name: "xiaomi_token_plan", Value: cfg.XiaomiTokenPlanBaseURL},
		{Name: "xiaomi_token_plan_anthropic", Value: cfg.XiaomiTokenPlanAnthropicBaseURL},
		{Name: "xiaomi_token_plan_sgp", Value: cfg.XiaomiTokenPlanSGPBaseURL},
		{Name: "xiaomi_token_plan_sgp_anthropic", Value: cfg.XiaomiTokenPlanSGPAnthropicBaseURL},
		{Name: "xiaomi_token_plan_ams", Value: cfg.XiaomiTokenPlanAMSBaseURL},
		{Name: "xiaomi_token_plan_ams_anthropic", Value: cfg.XiaomiTokenPlanAMSAnthropicBaseURL},
		{Name: "codex", Value: cfg.CodexBaseURL},
	}
}

func validationURLFields(cfg config.Config) []providerURLField {
	cfg = config.Normalize(cfg)
	fields := proxyBaseURLFields(cfg)
	fields = append(fields, providerURLField{Name: "codex_usage", Value: cfg.CodexUsageEndpoint})
	return fields
}

func ValidateProxyBaseURLs(cfg config.Config) error {
	return validateURLFields(proxyBaseURLFields(cfg), "base url")
}

func ValidateValidationURLs(cfg config.Config) error {
	return validateURLFields(validationURLFields(cfg), "url")
}

func validateURLFields(fields []providerURLField, label string) error {
	for _, field := range fields {
		if strings.TrimSpace(field.Value) == "" {
			continue
		}
		if _, err := url.ParseRequestURI(field.Value); err != nil {
			return fmt.Errorf("invalid %s %s: %w", field.Name, label, err)
		}
	}
	return nil
}

func routeBaseURL(cfg config.Config, route routeInfo, selected token.Token) string {
	cfg = config.Normalize(cfg)
	if isCodexCredential(selected) {
		return cfg.CodexBaseURL
	}

	switch token.NormalizeProvider(route.Provider) {
	case token.ProviderAnthropic:
		return cfg.AnthropicBaseURL
	case token.ProviderDeepSeek:
		if route.Protocol == "anthropic" {
			return cfg.DeepSeekAnthropicBaseURL
		}
		return cfg.DeepSeekBaseURL
	case token.ProviderKimi:
		return cfg.KimiBaseURL
	case token.ProviderZhipu:
		if route.Protocol == "anthropic" {
			return cfg.ZhipuAnthropicBaseURL
		}
		return cfg.ZhipuBaseURL
	case token.ProviderMiniMax:
		if route.Protocol == "anthropic" {
			return cfg.MiniMaxAnthropicBaseURL
		}
		return cfg.MiniMaxBaseURL
	case token.ProviderGemini:
		return cfg.GeminiBaseURL
	case token.ProviderOpenRouter:
		return cfg.OpenRouterBaseURL
	case token.ProviderFeatherless:
		return cfg.FeatherlessBaseURL
	case token.ProviderTokenRouter:
		return cfg.TokenRouterBaseURL
	case token.ProviderSub2API:
		if strings.TrimSpace(selected.BaseURL) != "" {
			return selected.BaseURL
		}
		return cfg.Sub2APIBaseURL
	case token.ProviderNewAPI:
		if strings.TrimSpace(selected.BaseURL) != "" {
			return selected.BaseURL
		}
		return cfg.NewAPIBaseURL
	case token.ProviderAnyRouter:
		if strings.TrimSpace(selected.BaseURL) != "" {
			return selected.BaseURL
		}
		return cfg.AnyRouterBaseURL
	case token.ProviderZo:
		return cfg.ZoBaseURL
	case token.ProviderPrem:
		return premProxyBaseURL(cfg.PremBaseURL)
	case token.ProviderAtria:
		return cfg.AtriaBaseURL
	case token.ProviderCustom:
		if route.Protocol == "anthropic" && cfg.CustomGatewayAnthropicBaseURL != "" {
			return cfg.CustomGatewayAnthropicBaseURL
		}
		return cfg.CustomGatewayBaseURL
	case token.ProviderXiaomi:
		return xiaomiBaseURL(cfg, route.Protocol, selected)
	default:
		if cfg.OpenAIBaseURL != "" {
			return cfg.OpenAIBaseURL
		}
		return cfg.UpstreamBaseURL
	}
}

func validationBaseURL(cfg config.Config, selected token.Token) string {
	cfg = config.Normalize(cfg)
	switch token.NormalizeProvider(selected.Provider) {
	case token.ProviderAnthropic:
		return cfg.AnthropicBaseURL
	case token.ProviderDeepSeek:
		return cfg.DeepSeekBaseURL
	case token.ProviderKimi:
		return cfg.KimiBaseURL
	case token.ProviderZhipu:
		return cfg.ZhipuBaseURL
	case token.ProviderMiniMax:
		return cfg.MiniMaxBaseURL
	case token.ProviderGemini:
		return cfg.GeminiBaseURL
	case token.ProviderOpenRouter:
		return cfg.OpenRouterBaseURL
	case token.ProviderFeatherless:
		return cfg.FeatherlessBaseURL
	case token.ProviderTokenRouter:
		return cfg.TokenRouterBaseURL
	case token.ProviderSub2API:
		if strings.TrimSpace(selected.BaseURL) != "" {
			return selected.BaseURL
		}
		return cfg.Sub2APIBaseURL
	case token.ProviderNewAPI:
		if strings.TrimSpace(selected.BaseURL) != "" {
			return selected.BaseURL
		}
		return cfg.NewAPIBaseURL
	case token.ProviderAnyRouter:
		if strings.TrimSpace(selected.BaseURL) != "" {
			return selected.BaseURL
		}
		return cfg.AnyRouterBaseURL
	case token.ProviderZo:
		return cfg.ZoBaseURL
	case token.ProviderPrem:
		return premProxyBaseURL(cfg.PremBaseURL)
	case token.ProviderAtria:
		return cfg.AtriaBaseURL
	case token.ProviderCustom:
		return cfg.CustomGatewayBaseURL
	case token.ProviderXiaomi:
		return xiaomiBaseURL(cfg, "openai", selected)
	default:
		if cfg.OpenAIBaseURL != "" {
			return cfg.OpenAIBaseURL
		}
		return cfg.UpstreamBaseURL
	}
}

func xiaomiBaseURL(cfg config.Config, protocol string, selected token.Token) string {
	if selected.CredentialType == token.CredentialTypeMimoTokenPlan {
		if protocol == "anthropic" {
			switch selected.Region {
			case token.MimoRegionSGP:
				return cfg.XiaomiTokenPlanSGPAnthropicBaseURL
			case token.MimoRegionAMS:
				return cfg.XiaomiTokenPlanAMSAnthropicBaseURL
			}
			return cfg.XiaomiTokenPlanAnthropicBaseURL
		}
		switch selected.Region {
		case token.MimoRegionSGP:
			return cfg.XiaomiTokenPlanSGPBaseURL
		case token.MimoRegionAMS:
			return cfg.XiaomiTokenPlanAMSBaseURL
		}
		return cfg.XiaomiTokenPlanBaseURL
	}
	if protocol == "anthropic" {
		return cfg.XiaomiAPIAnthropicBaseURL
	}
	return cfg.XiaomiAPIBaseURL
}

func premProxyBaseURL(baseURL string) string {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	lower := strings.ToLower(baseURL)
	for _, suffix := range []string{"/openai/v1", "/anthropic/v1", "/v1"} {
		if strings.HasSuffix(lower, suffix) {
			return strings.TrimRight(baseURL[:len(baseURL)-len(suffix)], "/")
		}
	}
	return baseURL
}

func ProxyConfigChanged(oldCfg config.Config, nextCfg config.Config) bool {
	oldCfg = config.Normalize(oldCfg)
	nextCfg = config.Normalize(nextCfg)
	if oldCfg.ProxyPort != nextCfg.ProxyPort ||
		oldCfg.SchedulingMode != nextCfg.SchedulingMode ||
		oldCfg.WebSocketMode != nextCfg.WebSocketMode ||
		oldCfg.OutboundProxyEnabled != nextCfg.OutboundProxyEnabled ||
		oldCfg.OutboundProxyURL != nextCfg.OutboundProxyURL ||
		oldCfg.XiaomiCredentialPriority != nextCfg.XiaomiCredentialPriority ||
		oldCfg.MaxRetries != nextCfg.MaxRetries {
		return true
	}
	if !reflect.DeepEqual(oldCfg.OutboundProxyModels, nextCfg.OutboundProxyModels) {
		return true
	}
	if !reflect.DeepEqual(oldCfg.OutboundProxyProviders, nextCfg.OutboundProxyProviders) {
		return true
	}
	if !reflect.DeepEqual(oldCfg.GatewayRoutes, nextCfg.GatewayRoutes) {
		return true
	}
	return !reflect.DeepEqual(proxyBaseURLFields(oldCfg), proxyBaseURLFields(nextCfg))
}
