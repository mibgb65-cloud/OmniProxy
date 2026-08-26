package config

import (
	"encoding/json"
	"errors"
	"omniproxy/internal/storage"
	"os"
)

type Store struct {
	path string
	file *storage.JSONStore[Config]
}

func NewStore(path string) *Store {
	return &Store{path: path, file: storage.NewJSONStore[Config](path)}
}

func (s *Store) Load() (Config, error) {
	cfg := Default()

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if len(data) == 0 {
		return cfg, nil
	}

	var saved struct {
		ProxyPort                          *int           `json:"proxyPort"`
		ControlPort                        *int           `json:"controlPort"`
		SchedulingMode                     *string        `json:"schedulingMode"`
		WebSocketMode                      *string        `json:"websocketMode"`
		CheckBetaUpdates                   *bool          `json:"checkBetaUpdates"`
		TaskAutomationEnabled              *bool          `json:"taskAutomationEnabled"`
		TaskAutomationClients              *[]string      `json:"taskAutomationClients"`
		TaskAutomationLaunchMode           *string        `json:"taskAutomationLaunchMode"`
		TaskAutomationLaunchTarget         *string        `json:"taskAutomationLaunchTarget"`
		TaskAutomationFallbackURL          *string        `json:"taskAutomationFallbackUrl"`
		TaskAutomationBrowser              *string        `json:"taskAutomationBrowser"`
		TaskAutomationBrowserUserDataDir   *string        `json:"taskAutomationBrowserUserDataDir"`
		TaskAutomationBrowserProfile       *string        `json:"taskAutomationBrowserProfile"`
		TaskAutomationReturnToClient       *bool          `json:"taskAutomationReturnToClient"`
		TaskAutomationIdleSeconds          *int           `json:"taskAutomationIdleSeconds"`
		TaskAutomationReturnDelaySeconds   *int           `json:"taskAutomationReturnDelaySeconds"`
		OutboundProxyEnabled               *bool          `json:"outboundProxyEnabled"`
		OutboundProxyURL                   *string        `json:"outboundProxyUrl"`
		OutboundProxyProviders             *[]string      `json:"outboundProxyProviders"`
		OutboundProxyModels                *[]string      `json:"outboundProxyModels"`
		UpstreamBaseURL                    *string        `json:"upstreamBaseUrl"`
		OpenAIBaseURL                      *string        `json:"openaiBaseUrl"`
		AnthropicBaseURL                   *string        `json:"anthropicBaseUrl"`
		DeepSeekBaseURL                    *string        `json:"deepseekBaseUrl"`
		DeepSeekAnthropicBaseURL           *string        `json:"deepseekAnthropicBaseUrl"`
		KimiBaseURL                        *string        `json:"kimiBaseUrl"`
		ZhipuBaseURL                       *string        `json:"zhipuBaseUrl"`
		ZhipuAnthropicBaseURL              *string        `json:"zhipuAnthropicBaseUrl"`
		MiniMaxBaseURL                     *string        `json:"minimaxBaseUrl"`
		MiniMaxAnthropicBaseURL            *string        `json:"minimaxAnthropicBaseUrl"`
		GeminiBaseURL                      *string        `json:"geminiBaseUrl"`
		OpenRouterBaseURL                  *string        `json:"openrouterBaseUrl"`
		TokenRouterBaseURL                 *string        `json:"tokenrouterBaseUrl"`
		Sub2APIBaseURL                     *string        `json:"sub2apiBaseUrl"`
		NewAPIBaseURL                      *string        `json:"newapiBaseUrl"`
		AnyRouterBaseURL                   *string        `json:"anyrouterBaseUrl"`
		ZoBaseURL                          *string        `json:"zoBaseUrl"`
		PremBaseURL                        *string        `json:"premBaseUrl"`
		PremAutoStartPCCIProxy             *bool          `json:"premAutoStartPcciProxy"`
		CustomGatewayBaseURL               *string        `json:"customGatewayBaseUrl"`
		CustomGatewayAnthropicBaseURL      *string        `json:"customGatewayAnthropicBaseUrl"`
		XiaomiBaseURL                      *string        `json:"xiaomiBaseUrl"`
		XiaomiAPIBaseURL                   *string        `json:"xiaomiApiBaseUrl"`
		XiaomiAPIAnthropicBaseURL          *string        `json:"xiaomiApiAnthropicBaseUrl"`
		XiaomiTokenPlanBaseURL             *string        `json:"xiaomiTokenPlanBaseUrl"`
		XiaomiTokenPlanAnthropicBaseURL    *string        `json:"xiaomiTokenPlanAnthropicBaseUrl"`
		XiaomiTokenPlanSGPBaseURL          *string        `json:"xiaomiTokenPlanSgpBaseUrl"`
		XiaomiTokenPlanSGPAnthropicBaseURL *string        `json:"xiaomiTokenPlanSgpAnthropicBaseUrl"`
		XiaomiTokenPlanAMSBaseURL          *string        `json:"xiaomiTokenPlanAmsBaseUrl"`
		XiaomiTokenPlanAMSAnthropicBaseURL *string        `json:"xiaomiTokenPlanAmsAnthropicBaseUrl"`
		XiaomiCredentialPriority           *string        `json:"xiaomiCredentialPriority"`
		CodexBaseURL                       *string        `json:"codexBaseUrl"`
		GatewayRoutes                      *GatewayRoutes `json:"gatewayRoutes"`
		ModelRoutes                        *ModelRoutes   `json:"modelRoutes"`
		SwitchThreshold                    *int           `json:"switchThreshold"`
		MaxRetries                         *int           `json:"maxRetries"`
		HistoryRetentionDays               *int           `json:"historyRetentionDays"`
		HealthWatchThreshold               *int           `json:"healthWatchThreshold"`
		HealthRiskThreshold                *int           `json:"healthRiskThreshold"`
		LongRequestAlertSeconds            *int           `json:"longRequestAlertSeconds"`
		CodexUsageEndpoint                 *string        `json:"codexUsageEndpoint"`
		CodexAutoActivateUsage             *bool          `json:"codexAutoActivateUsage"`
		ClaudeSubscriptionUsageEnabled     *bool          `json:"claudeSubscriptionUsageEnabled"`
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		return cfg, err
	}

	if saved.ProxyPort != nil && *saved.ProxyPort > 0 {
		cfg.ProxyPort = *saved.ProxyPort
	}
	if saved.ControlPort != nil && *saved.ControlPort > 0 {
		cfg.ControlPort = *saved.ControlPort
	}
	if saved.SchedulingMode != nil {
		cfg.SchedulingMode = *saved.SchedulingMode
	}
	if saved.WebSocketMode != nil {
		cfg.WebSocketMode = *saved.WebSocketMode
	}
	if saved.CheckBetaUpdates != nil {
		cfg.CheckBetaUpdates = *saved.CheckBetaUpdates
	}
	if saved.TaskAutomationEnabled != nil {
		cfg.TaskAutomationEnabled = *saved.TaskAutomationEnabled
	}
	if saved.TaskAutomationClients != nil {
		cfg.TaskAutomationClients = append([]string(nil), (*saved.TaskAutomationClients)...)
	}
	if saved.TaskAutomationLaunchMode != nil {
		cfg.TaskAutomationLaunchMode = *saved.TaskAutomationLaunchMode
	}
	if saved.TaskAutomationLaunchTarget != nil {
		cfg.TaskAutomationLaunchTarget = *saved.TaskAutomationLaunchTarget
	}
	if saved.TaskAutomationFallbackURL != nil {
		cfg.TaskAutomationFallbackURL = *saved.TaskAutomationFallbackURL
	}
	if saved.TaskAutomationBrowser != nil {
		cfg.TaskAutomationBrowser = *saved.TaskAutomationBrowser
	}
	if saved.TaskAutomationBrowserUserDataDir != nil {
		cfg.TaskAutomationBrowserUserDataDir = *saved.TaskAutomationBrowserUserDataDir
	}
	if saved.TaskAutomationBrowserProfile != nil {
		cfg.TaskAutomationBrowserProfile = *saved.TaskAutomationBrowserProfile
	}
	if saved.TaskAutomationReturnToClient != nil {
		cfg.TaskAutomationReturnToClient = *saved.TaskAutomationReturnToClient
	}
	if saved.TaskAutomationIdleSeconds != nil {
		cfg.TaskAutomationIdleSeconds = *saved.TaskAutomationIdleSeconds
	}
	if saved.TaskAutomationReturnDelaySeconds != nil {
		cfg.TaskAutomationReturnDelaySeconds = *saved.TaskAutomationReturnDelaySeconds
	}
	if saved.OutboundProxyEnabled != nil {
		cfg.OutboundProxyEnabled = *saved.OutboundProxyEnabled
	}
	if saved.OutboundProxyURL != nil {
		cfg.OutboundProxyURL = *saved.OutboundProxyURL
	}
	if saved.OutboundProxyProviders != nil {
		cfg.OutboundProxyProviders = append([]string(nil), (*saved.OutboundProxyProviders)...)
	} else if saved.OutboundProxyModels != nil {
		cfg.OutboundProxyProviders = providersFromOutboundProxyModels(*saved.OutboundProxyModels)
	}
	if saved.OutboundProxyModels != nil {
		cfg.OutboundProxyModels = append([]string(nil), (*saved.OutboundProxyModels)...)
	}
	if saved.UpstreamBaseURL != nil && *saved.UpstreamBaseURL != "" {
		cfg.UpstreamBaseURL = *saved.UpstreamBaseURL
	}
	if saved.OpenAIBaseURL != nil && *saved.OpenAIBaseURL != "" {
		cfg.OpenAIBaseURL = *saved.OpenAIBaseURL
	}
	if saved.AnthropicBaseURL != nil && *saved.AnthropicBaseURL != "" {
		cfg.AnthropicBaseURL = *saved.AnthropicBaseURL
	}
	if saved.DeepSeekBaseURL != nil && *saved.DeepSeekBaseURL != "" {
		cfg.DeepSeekBaseURL = *saved.DeepSeekBaseURL
	}
	if saved.DeepSeekAnthropicBaseURL != nil && *saved.DeepSeekAnthropicBaseURL != "" {
		cfg.DeepSeekAnthropicBaseURL = *saved.DeepSeekAnthropicBaseURL
	}
	if saved.KimiBaseURL != nil && *saved.KimiBaseURL != "" {
		cfg.KimiBaseURL = *saved.KimiBaseURL
	}
	if saved.ZhipuBaseURL != nil && *saved.ZhipuBaseURL != "" {
		cfg.ZhipuBaseURL = *saved.ZhipuBaseURL
	}
	if saved.ZhipuAnthropicBaseURL != nil && *saved.ZhipuAnthropicBaseURL != "" {
		cfg.ZhipuAnthropicBaseURL = *saved.ZhipuAnthropicBaseURL
	}
	if saved.MiniMaxBaseURL != nil && *saved.MiniMaxBaseURL != "" {
		cfg.MiniMaxBaseURL = *saved.MiniMaxBaseURL
	}
	if saved.MiniMaxAnthropicBaseURL != nil && *saved.MiniMaxAnthropicBaseURL != "" {
		cfg.MiniMaxAnthropicBaseURL = *saved.MiniMaxAnthropicBaseURL
	}
	if saved.GeminiBaseURL != nil && *saved.GeminiBaseURL != "" {
		cfg.GeminiBaseURL = *saved.GeminiBaseURL
	}
	if saved.OpenRouterBaseURL != nil && *saved.OpenRouterBaseURL != "" {
		cfg.OpenRouterBaseURL = *saved.OpenRouterBaseURL
	}
	if saved.TokenRouterBaseURL != nil && *saved.TokenRouterBaseURL != "" {
		cfg.TokenRouterBaseURL = *saved.TokenRouterBaseURL
	}
	if saved.Sub2APIBaseURL != nil && *saved.Sub2APIBaseURL != "" {
		cfg.Sub2APIBaseURL = *saved.Sub2APIBaseURL
	}
	if saved.NewAPIBaseURL != nil && *saved.NewAPIBaseURL != "" {
		cfg.NewAPIBaseURL = *saved.NewAPIBaseURL
	}
	if saved.AnyRouterBaseURL != nil && *saved.AnyRouterBaseURL != "" {
		cfg.AnyRouterBaseURL = *saved.AnyRouterBaseURL
	}
	if saved.ZoBaseURL != nil && *saved.ZoBaseURL != "" {
		cfg.ZoBaseURL = *saved.ZoBaseURL
	}
	if saved.PremBaseURL != nil && *saved.PremBaseURL != "" {
		cfg.PremBaseURL = *saved.PremBaseURL
	}
	if saved.PremAutoStartPCCIProxy != nil {
		cfg.PremAutoStartPCCIProxy = *saved.PremAutoStartPCCIProxy
	}
	if saved.CustomGatewayBaseURL != nil {
		cfg.CustomGatewayBaseURL = *saved.CustomGatewayBaseURL
	}
	if saved.CustomGatewayAnthropicBaseURL != nil {
		cfg.CustomGatewayAnthropicBaseURL = *saved.CustomGatewayAnthropicBaseURL
	}
	if saved.XiaomiBaseURL != nil {
		cfg.XiaomiBaseURL = *saved.XiaomiBaseURL
	}
	if saved.XiaomiAPIBaseURL != nil && *saved.XiaomiAPIBaseURL != "" {
		cfg.XiaomiAPIBaseURL = *saved.XiaomiAPIBaseURL
	} else if cfg.XiaomiBaseURL != "" {
		cfg.XiaomiAPIBaseURL = cfg.XiaomiBaseURL
	}
	if saved.XiaomiAPIAnthropicBaseURL != nil && *saved.XiaomiAPIAnthropicBaseURL != "" {
		cfg.XiaomiAPIAnthropicBaseURL = *saved.XiaomiAPIAnthropicBaseURL
	}
	if saved.XiaomiTokenPlanBaseURL != nil && *saved.XiaomiTokenPlanBaseURL != "" {
		cfg.XiaomiTokenPlanBaseURL = *saved.XiaomiTokenPlanBaseURL
	}
	if saved.XiaomiTokenPlanAnthropicBaseURL != nil && *saved.XiaomiTokenPlanAnthropicBaseURL != "" {
		cfg.XiaomiTokenPlanAnthropicBaseURL = *saved.XiaomiTokenPlanAnthropicBaseURL
	}
	if saved.XiaomiTokenPlanSGPBaseURL != nil && *saved.XiaomiTokenPlanSGPBaseURL != "" {
		cfg.XiaomiTokenPlanSGPBaseURL = *saved.XiaomiTokenPlanSGPBaseURL
	}
	if saved.XiaomiTokenPlanSGPAnthropicBaseURL != nil && *saved.XiaomiTokenPlanSGPAnthropicBaseURL != "" {
		cfg.XiaomiTokenPlanSGPAnthropicBaseURL = *saved.XiaomiTokenPlanSGPAnthropicBaseURL
	}
	if saved.XiaomiTokenPlanAMSBaseURL != nil && *saved.XiaomiTokenPlanAMSBaseURL != "" {
		cfg.XiaomiTokenPlanAMSBaseURL = *saved.XiaomiTokenPlanAMSBaseURL
	}
	if saved.XiaomiTokenPlanAMSAnthropicBaseURL != nil && *saved.XiaomiTokenPlanAMSAnthropicBaseURL != "" {
		cfg.XiaomiTokenPlanAMSAnthropicBaseURL = *saved.XiaomiTokenPlanAMSAnthropicBaseURL
	}
	if saved.XiaomiCredentialPriority != nil {
		cfg.XiaomiCredentialPriority = *saved.XiaomiCredentialPriority
	}
	if saved.CodexBaseURL != nil && *saved.CodexBaseURL != "" {
		cfg.CodexBaseURL = *saved.CodexBaseURL
	}
	if saved.GatewayRoutes != nil {
		cfg.GatewayRoutes = *saved.GatewayRoutes
	}
	if saved.ModelRoutes != nil {
		cfg.ModelRoutes = *saved.ModelRoutes
	}
	if saved.SwitchThreshold != nil && *saved.SwitchThreshold > 0 {
		cfg.SwitchThreshold = *saved.SwitchThreshold
	}
	if saved.MaxRetries != nil && *saved.MaxRetries >= 0 {
		cfg.MaxRetries = *saved.MaxRetries
	}
	if saved.HistoryRetentionDays != nil && *saved.HistoryRetentionDays > 0 {
		cfg.HistoryRetentionDays = *saved.HistoryRetentionDays
	}
	if saved.HealthWatchThreshold != nil && *saved.HealthWatchThreshold > 0 {
		cfg.HealthWatchThreshold = *saved.HealthWatchThreshold
	}
	if saved.HealthRiskThreshold != nil && *saved.HealthRiskThreshold > 0 {
		cfg.HealthRiskThreshold = *saved.HealthRiskThreshold
	}
	if saved.LongRequestAlertSeconds != nil && *saved.LongRequestAlertSeconds > 0 {
		cfg.LongRequestAlertSeconds = *saved.LongRequestAlertSeconds
	}
	if saved.CodexUsageEndpoint != nil && *saved.CodexUsageEndpoint != "" {
		cfg.CodexUsageEndpoint = *saved.CodexUsageEndpoint
	}
	if saved.CodexAutoActivateUsage != nil {
		cfg.CodexAutoActivateUsage = *saved.CodexAutoActivateUsage
	}
	if saved.ClaudeSubscriptionUsageEnabled != nil {
		cfg.ClaudeSubscriptionUsageEnabled = *saved.ClaudeSubscriptionUsageEnabled
	}
	return Normalize(cfg), nil
}

func (s *Store) Save(cfg Config) error {
	return s.file.Save(Normalize(cfg))
}
