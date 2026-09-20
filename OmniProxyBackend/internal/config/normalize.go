package config

import (
	"strings"
)

func Normalize(cfg Config) Config {
	defaults := Default()
	if cfg.ProxyPort <= 0 {
		cfg.ProxyPort = defaults.ProxyPort
	}
	if cfg.ControlPort <= 0 {
		cfg.ControlPort = defaults.ControlPort
	}
	switch strings.ToLower(strings.TrimSpace(cfg.SchedulingMode)) {
	case SchedulingModeBalanced:
		cfg.SchedulingMode = SchedulingModeBalanced
	case SchedulingModeQueue, "":
		cfg.SchedulingMode = SchedulingModeQueue
	default:
		cfg.SchedulingMode = defaults.SchedulingMode
	}
	switch strings.ToLower(strings.TrimSpace(cfg.WebSocketMode)) {
	case WebSocketModeDisabled:
		cfg.WebSocketMode = WebSocketModeDisabled
	case WebSocketModeEnabled, "":
		cfg.WebSocketMode = WebSocketModeEnabled
	default:
		cfg.WebSocketMode = defaults.WebSocketMode
	}
	if cfg.TaskAutomationClients == nil {
		cfg.TaskAutomationClients = append([]string(nil), defaults.TaskAutomationClients...)
	} else {
		cfg.TaskAutomationClients = normalizeTaskAutomationClients(cfg.TaskAutomationClients)
	}
	cfg.TaskAutomationLaunchMode = normalizeTaskAutomationLaunchMode(cfg.TaskAutomationLaunchMode)
	cfg.TaskAutomationLaunchTarget = strings.TrimSpace(cfg.TaskAutomationLaunchTarget)
	cfg.TaskAutomationFallbackURL = strings.TrimSpace(cfg.TaskAutomationFallbackURL)
	if cfg.TaskAutomationFallbackURL == "" {
		cfg.TaskAutomationFallbackURL = defaults.TaskAutomationFallbackURL
	}
	cfg.TaskAutomationBrowser = normalizeTaskAutomationBrowser(cfg.TaskAutomationBrowser)
	cfg.TaskAutomationBrowserUserDataDir = strings.TrimSpace(cfg.TaskAutomationBrowserUserDataDir)
	cfg.TaskAutomationBrowserProfile = strings.TrimSpace(cfg.TaskAutomationBrowserProfile)
	if cfg.TaskAutomationIdleSeconds <= 0 {
		cfg.TaskAutomationIdleSeconds = defaults.TaskAutomationIdleSeconds
	}
	if cfg.TaskAutomationIdleSeconds > 600 {
		cfg.TaskAutomationIdleSeconds = 600
	}
	if cfg.TaskAutomationReturnDelaySeconds <= 0 {
		cfg.TaskAutomationReturnDelaySeconds = defaults.TaskAutomationReturnDelaySeconds
	}
	if cfg.TaskAutomationReturnDelaySeconds > 600 {
		cfg.TaskAutomationReturnDelaySeconds = 600
	}
	cfg.OutboundProxyURL = normalizeOutboundProxyURL(cfg.OutboundProxyURL)
	if cfg.OutboundProxyURL == "" {
		cfg.OutboundProxyURL = defaults.OutboundProxyURL
	}
	if cfg.OutboundProxyProviders == nil {
		cfg.OutboundProxyProviders = append([]string(nil), defaults.OutboundProxyProviders...)
	} else {
		cfg.OutboundProxyProviders = normalizeOutboundProxyProviders(cfg.OutboundProxyProviders)
	}
	if cfg.OutboundProxyModels == nil {
		cfg.OutboundProxyModels = append([]string(nil), defaults.OutboundProxyModels...)
	} else {
		cfg.OutboundProxyModels = normalizeOutboundProxyModels(cfg.OutboundProxyModels)
	}
	if cfg.UpstreamBaseURL == "" {
		cfg.UpstreamBaseURL = defaults.UpstreamBaseURL
	}
	if cfg.OpenAIBaseURL == "" {
		cfg.OpenAIBaseURL = cfg.UpstreamBaseURL
	}
	if cfg.AnthropicBaseURL == "" {
		cfg.AnthropicBaseURL = defaults.AnthropicBaseURL
	}
	if cfg.DeepSeekBaseURL == "" {
		cfg.DeepSeekBaseURL = defaults.DeepSeekBaseURL
	}
	if cfg.DeepSeekAnthropicBaseURL == "" {
		cfg.DeepSeekAnthropicBaseURL = defaults.DeepSeekAnthropicBaseURL
	}
	if cfg.KimiBaseURL == "" {
		cfg.KimiBaseURL = defaults.KimiBaseURL
	}
	if cfg.ZhipuBaseURL == "" {
		cfg.ZhipuBaseURL = defaults.ZhipuBaseURL
	}
	if cfg.ZhipuAnthropicBaseURL == "" {
		cfg.ZhipuAnthropicBaseURL = defaults.ZhipuAnthropicBaseURL
	}
	if cfg.MiniMaxBaseURL == "" {
		cfg.MiniMaxBaseURL = defaults.MiniMaxBaseURL
	}
	if cfg.MiniMaxAnthropicBaseURL == "" {
		cfg.MiniMaxAnthropicBaseURL = defaults.MiniMaxAnthropicBaseURL
	}
	if cfg.GeminiBaseURL == "" {
		cfg.GeminiBaseURL = defaults.GeminiBaseURL
	}
	if cfg.OpenRouterBaseURL == "" {
		cfg.OpenRouterBaseURL = defaults.OpenRouterBaseURL
	}
	if cfg.FeatherlessBaseURL == "" {
		cfg.FeatherlessBaseURL = defaults.FeatherlessBaseURL
	}
	if cfg.TokenRouterBaseURL == "" {
		cfg.TokenRouterBaseURL = defaults.TokenRouterBaseURL
	}
	if cfg.Sub2APIBaseURL == "" {
		cfg.Sub2APIBaseURL = defaults.Sub2APIBaseURL
	}
	if cfg.NewAPIBaseURL == "" {
		cfg.NewAPIBaseURL = defaults.NewAPIBaseURL
	}
	if cfg.AnyRouterBaseURL == "" {
		cfg.AnyRouterBaseURL = defaults.AnyRouterBaseURL
	}
	if cfg.ZoBaseURL == "" {
		cfg.ZoBaseURL = defaults.ZoBaseURL
	}
	if cfg.PremBaseURL == "" {
		cfg.PremBaseURL = defaults.PremBaseURL
	}
	if cfg.AtriaBaseURL == "" {
		cfg.AtriaBaseURL = defaults.AtriaBaseURL
	}
	if cfg.CodexBaseURL == "" {
		cfg.CodexBaseURL = defaults.CodexBaseURL
	}
	cfg.GatewayRoutes = normalizeGatewayRoutes(cfg.GatewayRoutes, defaults.GatewayRoutes)
	cfg.ModelRoutes = normalizeModelRoutes(cfg.ModelRoutes)
	if cfg.XiaomiAPIBaseURL == "" {
		if cfg.XiaomiBaseURL != "" {
			cfg.XiaomiAPIBaseURL = cfg.XiaomiBaseURL
		} else {
			cfg.XiaomiAPIBaseURL = defaults.XiaomiAPIBaseURL
		}
	}
	if cfg.XiaomiAPIAnthropicBaseURL == "" {
		cfg.XiaomiAPIAnthropicBaseURL = defaults.XiaomiAPIAnthropicBaseURL
	}
	if cfg.XiaomiTokenPlanBaseURL == "" {
		cfg.XiaomiTokenPlanBaseURL = defaults.XiaomiTokenPlanBaseURL
	}
	if cfg.XiaomiTokenPlanAnthropicBaseURL == "" {
		cfg.XiaomiTokenPlanAnthropicBaseURL = defaults.XiaomiTokenPlanAnthropicBaseURL
	}
	if cfg.XiaomiTokenPlanSGPBaseURL == "" {
		cfg.XiaomiTokenPlanSGPBaseURL = defaults.XiaomiTokenPlanSGPBaseURL
	}
	if cfg.XiaomiTokenPlanSGPAnthropicBaseURL == "" {
		cfg.XiaomiTokenPlanSGPAnthropicBaseURL = defaults.XiaomiTokenPlanSGPAnthropicBaseURL
	}
	if cfg.XiaomiTokenPlanAMSBaseURL == "" {
		cfg.XiaomiTokenPlanAMSBaseURL = defaults.XiaomiTokenPlanAMSBaseURL
	}
	if cfg.XiaomiTokenPlanAMSAnthropicBaseURL == "" {
		cfg.XiaomiTokenPlanAMSAnthropicBaseURL = defaults.XiaomiTokenPlanAMSAnthropicBaseURL
	}
	switch strings.ToLower(strings.TrimSpace(cfg.XiaomiCredentialPriority)) {
	case MimoCredentialPriorityAPIKey, "api":
		cfg.XiaomiCredentialPriority = MimoCredentialPriorityAPIKey
	case MimoCredentialPriorityTokenPlan, "tokenplan", "token_plan", "token-plan":
		cfg.XiaomiCredentialPriority = MimoCredentialPriorityTokenPlan
	case "":
		cfg.XiaomiCredentialPriority = defaults.XiaomiCredentialPriority
	default:
		cfg.XiaomiCredentialPriority = defaults.XiaomiCredentialPriority
	}
	if cfg.SwitchThreshold <= 0 {
		cfg.SwitchThreshold = defaults.SwitchThreshold
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = defaults.MaxRetries
	}
	if cfg.HistoryRetentionDays <= 0 {
		cfg.HistoryRetentionDays = defaults.HistoryRetentionDays
	}
	if cfg.HistoryRetentionDays > 365 {
		cfg.HistoryRetentionDays = 365
	}
	if cfg.HealthWatchThreshold <= 0 {
		cfg.HealthWatchThreshold = defaults.HealthWatchThreshold
	}
	if cfg.HealthWatchThreshold > 100 {
		cfg.HealthWatchThreshold = 100
	}
	if cfg.HealthRiskThreshold <= 0 {
		cfg.HealthRiskThreshold = defaults.HealthRiskThreshold
	}
	if cfg.HealthRiskThreshold >= cfg.HealthWatchThreshold {
		cfg.HealthRiskThreshold = cfg.HealthWatchThreshold - 1
	}
	if cfg.HealthRiskThreshold < 1 {
		cfg.HealthRiskThreshold = 1
	}
	if cfg.LongRequestAlertSeconds <= 0 {
		cfg.LongRequestAlertSeconds = defaults.LongRequestAlertSeconds
	}
	if cfg.LongRequestAlertSeconds > 3600 {
		cfg.LongRequestAlertSeconds = 3600
	}
	if cfg.CodexUsageEndpoint == "" {
		cfg.CodexUsageEndpoint = defaults.CodexUsageEndpoint
	}
	return cfg
}
