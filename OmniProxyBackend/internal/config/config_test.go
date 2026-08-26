package config

import (
	"os"
	"path/filepath"
	"testing"

	"omniproxy/internal/token"
)

func TestNormalizeSchedulingAndWebSocketModes(t *testing.T) {
	cfg := Normalize(Config{})
	if cfg.SchedulingMode != SchedulingModeQueue {
		t.Fatalf("expected default queue scheduling, got %q", cfg.SchedulingMode)
	}
	if cfg.WebSocketMode != WebSocketModeEnabled {
		t.Fatalf("expected websocket enabled by default, got %q", cfg.WebSocketMode)
	}
	if cfg.OutboundProxyEnabled {
		t.Fatal("expected outbound proxy disabled by default")
	}
	if cfg.CheckBetaUpdates {
		t.Fatal("expected beta update checks disabled by default")
	}
	if cfg.CodexAutoActivateUsage {
		t.Fatal("expected Codex usage auto-activation disabled by default")
	}
	if cfg.OutboundProxyURL != "http://127.0.0.1:10808" {
		t.Fatalf("expected default outbound proxy URL, got %q", cfg.OutboundProxyURL)
	}
	if len(cfg.OutboundProxyModels) != len(defaultOutboundProxyModels) {
		t.Fatalf("expected default outbound proxy models, got %#v", cfg.OutboundProxyModels)
	}
	if len(cfg.OutboundProxyProviders) != len(defaultOutboundProxyProviders) {
		t.Fatalf("expected default outbound proxy providers, got %#v", cfg.OutboundProxyProviders)
	}
	if cfg.XiaomiCredentialPriority != MimoCredentialPriorityTokenPlan {
		t.Fatalf("expected default MiMo token plan priority, got %q", cfg.XiaomiCredentialPriority)
	}
	if cfg.XiaomiTokenPlanAMSBaseURL != "https://token-plan-ams.xiaomimimo.com/v1" || cfg.XiaomiTokenPlanAMSAnthropicBaseURL != "https://token-plan-ams.xiaomimimo.com/anthropic" {
		t.Fatalf("expected default MiMo AMS token plan urls, got openai=%q anthropic=%q", cfg.XiaomiTokenPlanAMSBaseURL, cfg.XiaomiTokenPlanAMSAnthropicBaseURL)
	}
	if cfg.TaskAutomationEnabled {
		t.Fatal("expected task automation disabled by default")
	}
	if len(cfg.TaskAutomationClients) != 3 || cfg.TaskAutomationClients[0] != "codex" || cfg.TaskAutomationClients[1] != "claude" || cfg.TaskAutomationClients[2] != "claude-desktop" {
		t.Fatalf("expected default task automation clients, got %#v", cfg.TaskAutomationClients)
	}
	if cfg.TaskAutomationLaunchMode != TaskAutomationLaunchModeMedia || cfg.TaskAutomationBrowser != TaskAutomationBrowserDefault {
		t.Fatalf("expected default task automation media/default browser, got mode=%q browser=%q", cfg.TaskAutomationLaunchMode, cfg.TaskAutomationBrowser)
	}
	if cfg.TaskAutomationFallbackURL != "https://www.douyin.com" || cfg.TaskAutomationIdleSeconds != 5 || cfg.TaskAutomationReturnDelaySeconds != 3 {
		t.Fatalf("expected default task automation timing, got fallback=%q idle=%d delay=%d", cfg.TaskAutomationFallbackURL, cfg.TaskAutomationIdleSeconds, cfg.TaskAutomationReturnDelaySeconds)
	}
	if cfg.ZhipuBaseURL == "" || cfg.MiniMaxBaseURL == "" || cfg.GeminiBaseURL == "" || cfg.OpenRouterBaseURL == "" || cfg.TokenRouterBaseURL == "" || cfg.Sub2APIBaseURL == "" || cfg.NewAPIBaseURL == "" || cfg.AnyRouterBaseURL == "" || cfg.ZoBaseURL == "" || cfg.PremBaseURL == "" {
		t.Fatalf("expected new provider default base urls, got zhipu=%q minimax=%q gemini=%q openrouter=%q tokenrouter=%q sub2api=%q newapi=%q anyrouter=%q zo=%q prem=%q", cfg.ZhipuBaseURL, cfg.MiniMaxBaseURL, cfg.GeminiBaseURL, cfg.OpenRouterBaseURL, cfg.TokenRouterBaseURL, cfg.Sub2APIBaseURL, cfg.NewAPIBaseURL, cfg.AnyRouterBaseURL, cfg.ZoBaseURL, cfg.PremBaseURL)
	}
	if cfg.GatewayRoutes.Codex.CredentialType != "" || cfg.GatewayRoutes.Claude.CredentialType != "" || cfg.GatewayRoutes.OpenAI.CredentialType != "" || cfg.GatewayRoutes.Gemini.CredentialType != "" {
		t.Fatalf("expected default gateway routes to use automatic credential matching, got %#v", cfg.GatewayRoutes)
	}
	if cfg.GatewayRoutes.Codex.Model != "gpt-5.6-sol" || cfg.GatewayRoutes.OpenAI.Model != "gpt-5.6-terra" {
		t.Fatalf("expected GPT-5.6 role-aware gateway defaults, got %#v", cfg.GatewayRoutes)
	}
	if !Default().PremAutoStartPCCIProxy {
		t.Fatal("expected Prem pcci-proxy auto-start enabled by default")
	}

	cfg = Normalize(Config{
		SchedulingMode:                   "BALANCED",
		WebSocketMode:                    "DISABLED",
		OutboundProxyEnabled:             true,
		OutboundProxyURL:                 "mixed:10808",
		OutboundProxyProviders:           []string{"codex", " ", "OPENROUTER", "zo-computer", "any-router", "prem-ai", "codex"},
		OutboundProxyModels:              []string{"gpt-5.4", " ", "GPT-5.4", "claude-*"},
		XiaomiCredentialPriority:         "api",
		TaskAutomationClients:            []string{"Codex", "claudecode", "claude-code-desktop", "codex", "unknown"},
		TaskAutomationLaunchMode:         "linux.do",
		TaskAutomationBrowser:            "msedge",
		TaskAutomationBrowserUserDataDir: "  %LOCALAPPDATA%\\Microsoft\\Edge\\User Data  ",
		TaskAutomationBrowserProfile:     "  Profile 1  ",
		TaskAutomationIdleSeconds:        900,
		TaskAutomationReturnDelaySeconds: 900,
	})
	if cfg.SchedulingMode != SchedulingModeBalanced {
		t.Fatalf("expected balanced scheduling, got %q", cfg.SchedulingMode)
	}
	if cfg.WebSocketMode != WebSocketModeDisabled {
		t.Fatalf("expected websocket disabled, got %q", cfg.WebSocketMode)
	}
	if cfg.XiaomiCredentialPriority != MimoCredentialPriorityAPIKey {
		t.Fatalf("expected MiMo API priority, got %q", cfg.XiaomiCredentialPriority)
	}
	if !cfg.OutboundProxyEnabled || cfg.OutboundProxyURL != "http://127.0.0.1:10808" {
		t.Fatalf("expected normalized outbound proxy, enabled=%v url=%q", cfg.OutboundProxyEnabled, cfg.OutboundProxyURL)
	}
	if len(cfg.OutboundProxyModels) != 2 || cfg.OutboundProxyModels[0] != "gpt-5.4" || cfg.OutboundProxyModels[1] != "claude-*" {
		t.Fatalf("expected normalized outbound proxy models, got %#v", cfg.OutboundProxyModels)
	}
	if len(cfg.OutboundProxyProviders) != 5 || cfg.OutboundProxyProviders[0] != "openai" || cfg.OutboundProxyProviders[1] != "openrouter" || cfg.OutboundProxyProviders[2] != "zo" || cfg.OutboundProxyProviders[3] != "anyrouter" || cfg.OutboundProxyProviders[4] != "prem" {
		t.Fatalf("expected normalized outbound proxy providers, got %#v", cfg.OutboundProxyProviders)
	}
	if len(cfg.TaskAutomationClients) != 3 || cfg.TaskAutomationClients[0] != "codex" || cfg.TaskAutomationClients[1] != "claude" || cfg.TaskAutomationClients[2] != "claude-desktop" {
		t.Fatalf("expected normalized task automation clients, got %#v", cfg.TaskAutomationClients)
	}
	if cfg.TaskAutomationLaunchMode != TaskAutomationLaunchModeLinuxDO || cfg.TaskAutomationBrowser != TaskAutomationBrowserEdge {
		t.Fatalf("expected normalized task automation browser mode, got mode=%q browser=%q", cfg.TaskAutomationLaunchMode, cfg.TaskAutomationBrowser)
	}
	if cfg.TaskAutomationBrowserUserDataDir != `%LOCALAPPDATA%\Microsoft\Edge\User Data` || cfg.TaskAutomationBrowserProfile != "Profile 1" {
		t.Fatalf("expected trimmed browser profile config, got data=%q profile=%q", cfg.TaskAutomationBrowserUserDataDir, cfg.TaskAutomationBrowserProfile)
	}
	if cfg.TaskAutomationIdleSeconds != 600 {
		t.Fatalf("expected capped task automation idle seconds, got %d", cfg.TaskAutomationIdleSeconds)
	}
	if cfg.TaskAutomationReturnDelaySeconds != 600 {
		t.Fatalf("expected capped task automation return delay seconds, got %d", cfg.TaskAutomationReturnDelaySeconds)
	}
}

func TestStoreLoadPreservesPremAutoStartDisabled(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"premAutoStartPcciProxy":false}`), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := NewStore(path).Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.PremAutoStartPCCIProxy {
		t.Fatal("expected saved Prem pcci-proxy auto-start=false to be preserved")
	}
}

func TestStoreLoadClaudeSubscriptionUsageToggle(t *testing.T) {
	dir := t.TempDir()

	defaultPath := filepath.Join(dir, "default.json")
	if err := os.WriteFile(defaultPath, []byte(`{}`), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := NewStore(defaultPath).Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.ClaudeSubscriptionUsageEnabled {
		t.Fatal("expected Claude subscription usage to stay opt-in by default")
	}

	enabledPath := filepath.Join(dir, "enabled.json")
	if err := os.WriteFile(enabledPath, []byte(`{"claudeSubscriptionUsageEnabled":true}`), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err = NewStore(enabledPath).Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.ClaudeSubscriptionUsageEnabled {
		t.Fatal("expected saved Claude subscription usage=true to be preserved")
	}
}

func TestStoreLoadCodexAutoActivateUsageToggle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"codexAutoActivateUsage":true}`), 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := NewStore(path).Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if !cfg.CodexAutoActivateUsage {
		t.Fatal("expected saved Codex usage auto-activation=true to be preserved")
	}
}

func TestStoreLoadLegacyGatewayRoutesWithoutFallbacks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	raw := []byte(`{
  "gatewayRoutes": {
    "openai": {
      "provider": "DeepSeek",
      "model": "deepseek-v4-pro[1m]"
    }
  }
}`)
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := NewStore(path).Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.GatewayRoutes.OpenAI.Provider != token.ProviderDeepSeek || cfg.GatewayRoutes.OpenAI.CredentialType != "" {
		t.Fatalf("expected legacy OpenAI route to normalize to DeepSeek without forced credential, got %#v", cfg.GatewayRoutes.OpenAI)
	}
	if cfg.GatewayRoutes.OpenAI.Model != "deepseek-v4-pro" {
		t.Fatalf("expected legacy route model to normalize to current DeepSeek model, got %q", cfg.GatewayRoutes.OpenAI.Model)
	}
	if cfg.GatewayRoutes.OpenAI.Fallbacks != nil {
		t.Fatalf("expected omitted legacy fallbacks to stay empty, got %#v", cfg.GatewayRoutes.OpenAI.Fallbacks)
	}
	if cfg.GatewayRoutes.Codex.Provider == "" || cfg.GatewayRoutes.Claude.Provider == "" || cfg.GatewayRoutes.Gemini.Provider == "" {
		t.Fatalf("expected missing gateway routes to receive defaults, got %#v", cfg.GatewayRoutes)
	}
}

func TestNormalizeGatewayRouteFallbacks(t *testing.T) {
	cfg := Normalize(Config{
		GatewayRoutes: GatewayRoutes{
			OpenAI: GatewayRouteConfig{
				Provider:       token.ProviderOpenAI,
				CredentialType: token.CredentialTypeAPIKey,
				Model:          "gpt-route",
				Fallbacks: []GatewayRouteConfig{
					{Provider: "DeepSeek"},
					{Provider: token.ProviderGemini},
					{Provider: token.ProviderOpenAI, CredentialType: token.CredentialTypeAPIKey},
					{Provider: token.ProviderZhipu, CredentialType: token.CredentialTypeCodingPlan, Model: "glm-route"},
				},
			},
		},
	})

	fallbacks := cfg.GatewayRoutes.OpenAI.Fallbacks
	if len(fallbacks) != 2 {
		t.Fatalf("expected two normalized fallbacks, got %#v", fallbacks)
	}
	if fallbacks[0].Provider != token.ProviderDeepSeek || fallbacks[0].CredentialType != "" || fallbacks[0].Model != "gpt-route" {
		t.Fatalf("unexpected first fallback: %#v", fallbacks[0])
	}
	if fallbacks[1].Provider != token.ProviderZhipu || fallbacks[1].CredentialType != token.CredentialTypeCodingPlan || fallbacks[1].Model != "glm-route" {
		t.Fatalf("unexpected second fallback: %#v", fallbacks[1])
	}
}

func TestNormalizeGatewayRouteFallbacksDropsNestedAndDuplicateEntries(t *testing.T) {
	cfg := Normalize(Config{
		GatewayRoutes: GatewayRoutes{
			OpenAI: GatewayRouteConfig{
				Provider:       token.ProviderOpenAI,
				CredentialType: token.CredentialTypeAPIKey,
				Model:          "gpt-route",
				Fallbacks: []GatewayRouteConfig{
					{
						Provider: token.ProviderDeepSeek,
						Fallbacks: []GatewayRouteConfig{
							{Provider: token.ProviderZhipu},
						},
					},
					{Provider: " DEEPSEEK ", Model: "duplicate-should-drop"},
					{Provider: token.ProviderZhipu, CredentialType: token.CredentialTypeAPIKey},
				},
			},
		},
	})

	fallbacks := cfg.GatewayRoutes.OpenAI.Fallbacks
	if len(fallbacks) != 2 {
		t.Fatalf("expected duplicate fallback to be dropped, got %#v", fallbacks)
	}
	if fallbacks[0].Provider != token.ProviderDeepSeek || len(fallbacks[0].Fallbacks) != 0 {
		t.Fatalf("expected nested fallback chain to be cleared, got %#v", fallbacks[0])
	}
	if fallbacks[1].Provider != token.ProviderZhipu || fallbacks[1].CredentialType != token.CredentialTypeAPIKey {
		t.Fatalf("unexpected second fallback: %#v", fallbacks[1])
	}
}

func TestNormalizeModelRoutesPreservesBackendOrder(t *testing.T) {
	cfg := Normalize(Config{
		ModelRoutes: ModelRoutes{
			" deepseek-v4-pro[1m] ": GatewayRouteConfig{
				Provider: " DeepSeek ",
				Fallbacks: []GatewayRouteConfig{
					{Provider: token.ProviderPrem, CredentialType: token.CredentialTypeAPIKey},
					{Provider: token.ProviderDeepSeek},
				},
			},
		},
	})

	route := cfg.ModelRoutes["deepseek-v4-pro[1m]"]
	if route.Provider != token.ProviderDeepSeek || route.Model != "deepseek-v4-pro[1m]" {
		t.Fatalf("unexpected model route primary: %#v", route)
	}
	if len(route.Fallbacks) != 1 || route.Fallbacks[0].Provider != token.ProviderPrem || route.Fallbacks[0].Model != "deepseek-v4-pro[1m]" {
		t.Fatalf("unexpected model route fallbacks: %#v", route.Fallbacks)
	}
}

func TestNormalizeModelRoutesAllowsDifferentUpstreamModel(t *testing.T) {
	cfg := Normalize(Config{
		ModelRoutes: ModelRoutes{
			"custom-client-model": GatewayRouteConfig{
				Provider: token.ProviderPrem,
				Model:    "prem-upstream-model",
			},
		},
	})

	route := cfg.ModelRoutes["custom-client-model"]
	if route.Provider != token.ProviderPrem || route.Model != "prem-upstream-model" {
		t.Fatalf("expected model route to preserve upstream model override, got %#v", route)
	}
}
