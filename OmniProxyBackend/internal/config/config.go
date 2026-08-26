package config

import (
	"omniproxy/internal/token"
)

const (
	SchedulingModeQueue    = "queue"
	SchedulingModeBalanced = "balanced"

	WebSocketModeEnabled  = "enabled"
	WebSocketModeDisabled = "disabled"

	TaskAutomationLaunchModeMedia   = "media"
	TaskAutomationLaunchModeLinuxDO = "linuxdo"

	TaskAutomationBrowserDefault = "default"
	TaskAutomationBrowserEdge    = "edge"
	TaskAutomationBrowserChrome  = "chrome"
	TaskAutomationBrowserFirefox = "firefox"

	MimoCredentialPriorityAPIKey    = "api_key"
	MimoCredentialPriorityTokenPlan = "mimo_token_plan"
)

const (
	GatewayRouteCodex  = "codex"
	GatewayRouteClaude = "claude"
	GatewayRouteOpenAI = "openai"
	GatewayRouteGemini = "gemini"
)

var defaultOutboundProxyModels = []string{
	"gpt-*",
	"claude-*",
	"gemini-*",
	"*/*",
}

var defaultOutboundProxyProviders = []string{
	"openai",
	"anthropic",
	"gemini",
	"openrouter",
	"zo",
	"prem",
}

type Config struct {
	ProxyPort                          int           `json:"proxyPort"`
	ControlPort                        int           `json:"controlPort"`
	SchedulingMode                     string        `json:"schedulingMode"`
	WebSocketMode                      string        `json:"websocketMode"`
	CheckBetaUpdates                   bool          `json:"checkBetaUpdates"`
	TaskAutomationEnabled              bool          `json:"taskAutomationEnabled"`
	TaskAutomationClients              []string      `json:"taskAutomationClients"`
	TaskAutomationLaunchMode           string        `json:"taskAutomationLaunchMode"`
	TaskAutomationLaunchTarget         string        `json:"taskAutomationLaunchTarget"`
	TaskAutomationFallbackURL          string        `json:"taskAutomationFallbackUrl"`
	TaskAutomationBrowser              string        `json:"taskAutomationBrowser"`
	TaskAutomationBrowserUserDataDir   string        `json:"taskAutomationBrowserUserDataDir"`
	TaskAutomationBrowserProfile       string        `json:"taskAutomationBrowserProfile"`
	TaskAutomationReturnToClient       bool          `json:"taskAutomationReturnToClient"`
	TaskAutomationIdleSeconds          int           `json:"taskAutomationIdleSeconds"`
	TaskAutomationReturnDelaySeconds   int           `json:"taskAutomationReturnDelaySeconds"`
	OutboundProxyEnabled               bool          `json:"outboundProxyEnabled"`
	OutboundProxyURL                   string        `json:"outboundProxyUrl"`
	OutboundProxyProviders             []string      `json:"outboundProxyProviders"`
	OutboundProxyModels                []string      `json:"outboundProxyModels"`
	UpstreamBaseURL                    string        `json:"upstreamBaseUrl"`
	OpenAIBaseURL                      string        `json:"openaiBaseUrl"`
	AnthropicBaseURL                   string        `json:"anthropicBaseUrl"`
	DeepSeekBaseURL                    string        `json:"deepseekBaseUrl"`
	DeepSeekAnthropicBaseURL           string        `json:"deepseekAnthropicBaseUrl"`
	KimiBaseURL                        string        `json:"kimiBaseUrl"`
	ZhipuBaseURL                       string        `json:"zhipuBaseUrl"`
	ZhipuAnthropicBaseURL              string        `json:"zhipuAnthropicBaseUrl"`
	MiniMaxBaseURL                     string        `json:"minimaxBaseUrl"`
	MiniMaxAnthropicBaseURL            string        `json:"minimaxAnthropicBaseUrl"`
	GeminiBaseURL                      string        `json:"geminiBaseUrl"`
	OpenRouterBaseURL                  string        `json:"openrouterBaseUrl"`
	TokenRouterBaseURL                 string        `json:"tokenrouterBaseUrl"`
	Sub2APIBaseURL                     string        `json:"sub2apiBaseUrl"`
	NewAPIBaseURL                      string        `json:"newapiBaseUrl"`
	AnyRouterBaseURL                   string        `json:"anyrouterBaseUrl"`
	ZoBaseURL                          string        `json:"zoBaseUrl"`
	PremBaseURL                        string        `json:"premBaseUrl"`
	PremAutoStartPCCIProxy             bool          `json:"premAutoStartPcciProxy"`
	CustomGatewayBaseURL               string        `json:"customGatewayBaseUrl"`
	CustomGatewayAnthropicBaseURL      string        `json:"customGatewayAnthropicBaseUrl"`
	XiaomiBaseURL                      string        `json:"xiaomiBaseUrl"`
	XiaomiAPIBaseURL                   string        `json:"xiaomiApiBaseUrl"`
	XiaomiAPIAnthropicBaseURL          string        `json:"xiaomiApiAnthropicBaseUrl"`
	XiaomiTokenPlanBaseURL             string        `json:"xiaomiTokenPlanBaseUrl"`
	XiaomiTokenPlanAnthropicBaseURL    string        `json:"xiaomiTokenPlanAnthropicBaseUrl"`
	XiaomiTokenPlanSGPBaseURL          string        `json:"xiaomiTokenPlanSgpBaseUrl"`
	XiaomiTokenPlanSGPAnthropicBaseURL string        `json:"xiaomiTokenPlanSgpAnthropicBaseUrl"`
	XiaomiTokenPlanAMSBaseURL          string        `json:"xiaomiTokenPlanAmsBaseUrl"`
	XiaomiTokenPlanAMSAnthropicBaseURL string        `json:"xiaomiTokenPlanAmsAnthropicBaseUrl"`
	XiaomiCredentialPriority           string        `json:"xiaomiCredentialPriority"`
	CodexBaseURL                       string        `json:"codexBaseUrl"`
	GatewayRoutes                      GatewayRoutes `json:"gatewayRoutes"`
	ModelRoutes                        ModelRoutes   `json:"modelRoutes,omitempty"`
	SwitchThreshold                    int           `json:"switchThreshold"`
	MaxRetries                         int           `json:"maxRetries"`
	HistoryRetentionDays               int           `json:"historyRetentionDays"`
	HealthWatchThreshold               int           `json:"healthWatchThreshold"`
	HealthRiskThreshold                int           `json:"healthRiskThreshold"`
	LongRequestAlertSeconds            int           `json:"longRequestAlertSeconds"`
	CodexUsageEndpoint                 string        `json:"codexUsageEndpoint"`
	CodexAutoActivateUsage             bool          `json:"codexAutoActivateUsage"`
	ClaudeSubscriptionUsageEnabled     bool          `json:"claudeSubscriptionUsageEnabled"`
}

type GatewayRouteConfig struct {
	Provider       string               `json:"provider"`
	CredentialType string               `json:"credentialType,omitempty"`
	Model          string               `json:"model,omitempty"`
	Fallbacks      []GatewayRouteConfig `json:"fallbacks,omitempty"`
}

type GatewayRoutes struct {
	Codex  GatewayRouteConfig `json:"codex"`
	Claude GatewayRouteConfig `json:"claude"`
	OpenAI GatewayRouteConfig `json:"openai"`
	Gemini GatewayRouteConfig `json:"gemini"`
}

type ModelRoutes map[string]GatewayRouteConfig

func Default() Config {
	return Config{
		ProxyPort:                          DefaultProxyPort(),
		ControlPort:                        DefaultControlPort(),
		SchedulingMode:                     SchedulingModeQueue,
		WebSocketMode:                      WebSocketModeEnabled,
		CheckBetaUpdates:                   false,
		TaskAutomationEnabled:              false,
		TaskAutomationClients:              []string{"codex", "claude", "claude-desktop"},
		TaskAutomationLaunchMode:           TaskAutomationLaunchModeMedia,
		TaskAutomationLaunchTarget:         "",
		TaskAutomationFallbackURL:          "https://www.douyin.com",
		TaskAutomationBrowser:              TaskAutomationBrowserDefault,
		TaskAutomationBrowserUserDataDir:   "",
		TaskAutomationBrowserProfile:       "",
		TaskAutomationReturnToClient:       true,
		TaskAutomationIdleSeconds:          5,
		TaskAutomationReturnDelaySeconds:   3,
		OutboundProxyEnabled:               false,
		OutboundProxyURL:                   "http://127.0.0.1:10808",
		OutboundProxyProviders:             append([]string(nil), defaultOutboundProxyProviders...),
		OutboundProxyModels:                append([]string(nil), defaultOutboundProxyModels...),
		UpstreamBaseURL:                    "https://api.openai.com",
		OpenAIBaseURL:                      "https://api.openai.com",
		AnthropicBaseURL:                   "https://api.anthropic.com",
		DeepSeekBaseURL:                    "https://api.deepseek.com",
		DeepSeekAnthropicBaseURL:           "https://api.deepseek.com/anthropic",
		KimiBaseURL:                        "https://api.kimi.com/coding",
		ZhipuBaseURL:                       "https://open.bigmodel.cn/api/paas/v4",
		ZhipuAnthropicBaseURL:              "https://open.bigmodel.cn/api/anthropic",
		MiniMaxBaseURL:                     "https://api.minimaxi.com/v1",
		MiniMaxAnthropicBaseURL:            "https://api.minimaxi.com/anthropic",
		GeminiBaseURL:                      "https://generativelanguage.googleapis.com",
		OpenRouterBaseURL:                  "https://openrouter.ai/api/v1",
		TokenRouterBaseURL:                 "https://api.tokenrouter.io",
		Sub2APIBaseURL:                     "https://aiapi.aicnio.com",
		NewAPIBaseURL:                      "http://127.0.0.1:3000",
		AnyRouterBaseURL:                   "https://anyrouter.top",
		ZoBaseURL:                          "https://api.zo.computer",
		PremBaseURL:                        "http://127.0.0.1:3100",
		PremAutoStartPCCIProxy:             true,
		CustomGatewayBaseURL:               "",
		CustomGatewayAnthropicBaseURL:      "",
		XiaomiBaseURL:                      "",
		XiaomiAPIBaseURL:                   "https://api.xiaomimimo.com/v1",
		XiaomiAPIAnthropicBaseURL:          "https://api.xiaomimimo.com/anthropic",
		XiaomiTokenPlanBaseURL:             "https://token-plan-cn.xiaomimimo.com/v1",
		XiaomiTokenPlanAnthropicBaseURL:    "https://token-plan-cn.xiaomimimo.com/anthropic",
		XiaomiTokenPlanSGPBaseURL:          "https://token-plan-sgp.xiaomimimo.com/v1",
		XiaomiTokenPlanSGPAnthropicBaseURL: "https://token-plan-sgp.xiaomimimo.com/anthropic",
		XiaomiTokenPlanAMSBaseURL:          "https://token-plan-ams.xiaomimimo.com/v1",
		XiaomiTokenPlanAMSAnthropicBaseURL: "https://token-plan-ams.xiaomimimo.com/anthropic",
		XiaomiCredentialPriority:           MimoCredentialPriorityTokenPlan,
		CodexBaseURL:                       "https://chatgpt.com/backend-api/codex",
		GatewayRoutes: GatewayRoutes{
			Codex:  GatewayRouteConfig{Provider: token.ProviderOpenAI, Model: "gpt-5.6-sol"},
			Claude: GatewayRouteConfig{Provider: token.ProviderAnthropic, Model: "default"},
			OpenAI: GatewayRouteConfig{Provider: token.ProviderOpenAI, Model: "gpt-5.6-terra"},
			Gemini: GatewayRouteConfig{Provider: token.ProviderGemini, Model: "gemini-3-pro-preview"},
		},
		SwitchThreshold:         15,
		MaxRetries:              2,
		HistoryRetentionDays:    14,
		HealthWatchThreshold:    80,
		HealthRiskThreshold:     50,
		LongRequestAlertSeconds: 120,
		CodexUsageEndpoint:      "https://chatgpt.com/backend-api/wham/usage",
		CodexAutoActivateUsage:  false,
	}
}
