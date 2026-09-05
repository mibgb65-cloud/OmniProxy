package main

import (
	"fmt"
	"omniproxy/internal/clientconfig"
	"omniproxy/internal/logs"
	"os"
	"path/filepath"
	"strings"
)

func (a *appServer) configureOpenCode() (clientConfigureResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return clientConfigureResult{}, err
	}

	a.mu.Lock()
	port := a.cfg.ProxyPort
	a.mu.Unlock()

	opencodeDir := filepath.Join(home, ".config", "opencode")
	if err := os.MkdirAll(opencodeDir, 0o755); err != nil {
		return clientConfigureResult{}, err
	}
	configPath := filepath.Join(opencodeDir, "opencode.json")
	routerBaseURL := fmt.Sprintf("http://127.0.0.1:%d/opencode-router/v1", port)
	geminiBaseURL := fmt.Sprintf("http://127.0.0.1:%d/gemini", port)
	openRouterBaseURL := fmt.Sprintf("http://127.0.0.1:%d/openrouter/v1", port)
	tokenRouterBaseURL := fmt.Sprintf("http://127.0.0.1:%d/tokenrouter/v1", port)
	zoBaseURL := fmt.Sprintf("http://127.0.0.1:%d/zo/v1", port)
	customBaseURL := fmt.Sprintf("http://127.0.0.1:%d/custom/v1", port)
	openRouterModels := a.openCodeOpenRouterModels()

	if err := writeOpenCodeConfig(configPath, routerBaseURL, geminiBaseURL, openRouterBaseURL, tokenRouterBaseURL, zoBaseURL, customBaseURL, openRouterModels); err != nil {
		return clientConfigureResult{}, err
	}

	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "opencode configured for omniproxy"})
	return clientConfigureResult{
		ConfigPath: configPath,
		BackupPath: configPath + ".omniproxy.bak",
		BaseURL:    routerBaseURL,
		ProviderID: opencodeOmniProviderID,
		Message:    "OpenCode 已添加 OmniProxy 网关 provider，后端平台请在网关路由中选择",
	}, nil
}

func (a *appServer) restoreOpenCodeConfig() (clientConfigureResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return clientConfigureResult{}, err
	}

	configPath := filepath.Join(home, ".config", "opencode", "opencode.json")
	if err := clientconfig.RestoreBackup(configPath, configPath+".omniproxy.bak"); err != nil {
		return clientConfigureResult{}, err
	}

	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "opencode config restored"})
	return clientConfigureResult{
		ConfigPath: configPath,
		BackupPath: configPath + ".omniproxy.bak",
		Message:    "OpenCode 配置已恢复到一键配置前的原始配置",
	}, nil
}

func (a *appServer) configurePi() (clientConfigureResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return clientConfigureResult{}, err
	}

	a.mu.Lock()
	port := a.cfg.ProxyPort
	a.mu.Unlock()

	piDir := filepath.Join(home, ".pi", "agent")
	if err := os.MkdirAll(piDir, 0o755); err != nil {
		return clientConfigureResult{}, err
	}
	configPath := filepath.Join(piDir, "models.json")
	routerBaseURL := fmt.Sprintf("http://127.0.0.1:%d/pi-router/v1", port)
	zoBaseURL := fmt.Sprintf("http://127.0.0.1:%d/zo/v1", port)
	openRouterModels := a.piOpenRouterModels()

	if err := writePiModelsConfig(configPath, routerBaseURL, zoBaseURL, openRouterModels); err != nil {
		return clientConfigureResult{}, err
	}

	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "pi configured for omniproxy"})
	return clientConfigureResult{
		ConfigPath: configPath,
		BackupPath: configPath + ".omniproxy.bak",
		BaseURL:    routerBaseURL,
		ProviderID: piOmniProviderID,
		Message:    "Pi Coding Agent 已添加 OmniProxy 网关 provider，后端平台请在网关路由中选择",
	}, nil
}

func (a *appServer) restorePiConfig() (clientConfigureResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return clientConfigureResult{}, err
	}

	configPath := filepath.Join(home, ".pi", "agent", "models.json")
	if err := clientconfig.RestoreBackup(configPath, configPath+".omniproxy.bak"); err != nil {
		return clientConfigureResult{}, err
	}

	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "pi config restored"})
	return clientConfigureResult{
		ConfigPath: configPath,
		BackupPath: configPath + ".omniproxy.bak",
		Message:    "Pi Coding Agent 配置已恢复到一键配置前的原始配置",
	}, nil
}

func writeOpenCodeConfig(path string, routerBaseURL string, geminiBaseURL string, openRouterBaseURL string, tokenRouterBaseURL string, zoBaseURL string, customBaseURL string, openRouterModels map[string]any) error {
	data, err := clientconfig.ReadJSONObject(path)
	if err != nil {
		return err
	}
	if err := clientconfig.BackupFile(path, path+".omniproxy.bak", []byte("{\n  \"$schema\": \"https://opencode.ai/config.json\"\n}\n")); err != nil {
		return err
	}
	if _, ok := data["$schema"].(string); !ok {
		data["$schema"] = "https://opencode.ai/config.json"
	}

	providers, _ := data["provider"].(map[string]any)
	if providers == nil {
		providers = map[string]any{}
	}
	for _, id := range []string{opencodeGeminiProviderID, opencodeOpenRouterProviderID, opencodeTokenRouterProviderID, opencodeZoProviderID, opencodeCustomProviderID} {
		delete(providers, id)
	}
	providers[opencodeOmniProviderID] = openCodeRouterProvider(routerBaseURL, openRouterModels)
	data["provider"] = providers

	return clientconfig.WriteJSONObject(path, data)
}

func writePiModelsConfig(path string, routerBaseURL string, zoBaseURL string, openRouterModels []map[string]any) error {
	data, err := clientconfig.ReadJSONObject(path)
	if err != nil {
		return err
	}
	if err := clientconfig.BackupFile(path, path+".omniproxy.bak", []byte("{\n  \"providers\": {}\n}\n")); err != nil {
		return err
	}

	providers, _ := data["providers"].(map[string]any)
	if providers == nil {
		providers = map[string]any{}
	}
	for _, id := range []string{piAnthropicProviderID, piGeminiProviderID, piOpenRouterProviderID, piZoProviderID, piCustomProviderID} {
		delete(providers, id)
	}
	providers[piOmniProviderID] = piOpenAIProvider("OmniProxy", routerBaseURL, piRouterModels(openRouterModels), true)
	data["providers"] = providers

	return clientconfig.WriteJSONObject(path, data)
}

func piRouterModels(openRouterModels []map[string]any) []map[string]any {
	models := []map[string]any{
		{"id": "gpt-5.6-sol", "name": "GPT-5.6 Sol"},
		{"id": "gpt-5.6-terra", "name": "GPT-5.6 Terra"},
		{"id": "gpt-5.6-luna", "name": "GPT-5.6 Luna"},
		piReasoningModel("gpt-6-astra", "GPT-6 Astra"),
		{"id": "gpt-5.4", "name": "GPT-5.4"},
		{"id": "deepseek-v4-pro", "name": "DeepSeek V4 Pro"},
		{"id": "kimi-for-coding", "name": "Kimi for Coding"},
		{"id": "glm-5.1", "name": "GLM-5.1"},
		{"id": "MiniMax-M2.7", "name": "MiniMax M2.7"},
		{"id": "auto:balance", "name": "TokenRouter Auto Balance"},
		{"id": "auto:quality", "name": "TokenRouter Auto Quality"},
		{"id": "auto:speed", "name": "TokenRouter Auto Speed"},
		{"id": "auto:cost", "name": "TokenRouter Auto Cost"},
		piReasoningModel(mimoModel, "MiMo V2.5 Pro"),
		{"id": "custom-model", "name": "Custom Gateway Model"},
	}
	seen := map[string]bool{}
	for _, model := range models {
		if id, ok := model["id"].(string); ok {
			seen[strings.ToLower(strings.TrimSpace(id))] = true
		}
	}
	for _, model := range openRouterModels {
		id, _ := model["id"].(string)
		key := strings.ToLower(strings.TrimSpace(id))
		if key == "" || seen[key] {
			continue
		}
		models = append(models, model)
		seen[key] = true
	}
	if !seen["openrouter/auto"] {
		models = append(models, defaultPiOpenRouterModels()...)
	}
	return models
}

func (a *appServer) openCodeOpenRouterModels() map[string]any {
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()
	if cached, ok := a.cachedOpenRouterModels(cfg.OpenRouterBaseURL, "", false); ok {
		return openCodeModelsFromOpenRouter(cached.Models)
	}
	return defaultOpenCodeOpenRouterModels()
}

func (a *appServer) piOpenRouterModels() []map[string]any {
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()
	if cached, ok := a.cachedOpenRouterModels(cfg.OpenRouterBaseURL, "", false); ok {
		return piModelsFromOpenRouter(cached.Models)
	}
	return defaultPiOpenRouterModels()
}

func piOpenAIProvider(name string, baseURL string, models []map[string]any, supportsReasoningEffort bool) map[string]any {
	if len(models) == 0 {
		models = defaultPiOpenRouterModels()
	}
	return map[string]any{
		"name":       name,
		"api":        "openai-completions",
		"baseUrl":    baseURL,
		"apiKey":     localClientAPIKey,
		"authHeader": true,
		"compat": map[string]any{
			"supportsDeveloperRole":   false,
			"supportsReasoningEffort": supportsReasoningEffort,
		},
		"models": models,
	}
}

func piReasoningModel(id string, name string) map[string]any {
	return map[string]any{
		"id":        id,
		"name":      name,
		"reasoning": true,
		"compat": map[string]any{
			"supportsReasoningEffort": true,
		},
	}
}

func piAnthropicProvider(baseURL string) map[string]any {
	return map[string]any{
		"name":    "OmniProxy Anthropic Router",
		"api":     "anthropic-messages",
		"baseUrl": baseURL,
		"apiKey":  localClientAPIKey,
		"models": []map[string]any{
			{"id": "claude-sonnet-4-5", "name": "Claude Sonnet 4.5"},
			{"id": "deepseek-v4-pro", "name": "DeepSeek V4 Pro"},
			{"id": "kimi-for-coding", "name": "Kimi for Coding"},
			{"id": "glm-5.1", "name": "GLM-5.1"},
			{"id": "mimo-v2.5-pro[1m]", "name": "MiMo V2.5 Pro 1M"},
		},
	}
}

func piGeminiProvider(baseURL string) map[string]any {
	return map[string]any{
		"name":    "OmniProxy Gemini",
		"api":     "google-generative-ai",
		"baseUrl": baseURL,
		"apiKey":  localClientAPIKey,
		"models": []map[string]any{
			{"id": "gemini-3-pro-preview", "name": "Gemini 3 Pro Preview"},
			{"id": "gemini-3-flash-preview", "name": "Gemini 3 Flash Preview"},
			{"id": "gemini-2.5-flash-lite", "name": "Gemini 2.5 Flash Lite"},
		},
	}
}

func piModelsFromOpenRouter(models []openRouterModelResponse) []map[string]any {
	out := make([]map[string]any, 0, len(models))
	for _, model := range models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(model.Name)
		if name == "" {
			name = id
		}
		out = append(out, map[string]any{"id": id, "name": name})
	}
	if len(out) == 0 {
		return defaultPiOpenRouterModels()
	}
	return out
}

func defaultPiOpenRouterModels() []map[string]any {
	return []map[string]any{
		{"id": "openrouter/auto", "name": "OpenRouter Auto"},
	}
}

func openCodeRouterProvider(baseURL string, extraModels map[string]any) map[string]any {
	models := map[string]any{
		"gpt-6-astra":     map[string]any{"name": "GPT-6 Astra"},
		"gpt-5.6-sol":     map[string]any{"name": "GPT-5.6 Sol"},
		"gpt-5.6-terra":   map[string]any{"name": "GPT-5.6 Terra"},
		"gpt-5.6-luna":    map[string]any{"name": "GPT-5.6 Luna"},
		"gpt-5.4":         map[string]any{"name": "GPT-5.4"},
		"deepseek-v4-pro": map[string]any{"name": "DeepSeek V4 Pro"},
		"glm-5.1":         map[string]any{"name": "GLM-5.1"},
		"MiniMax-M2.7":    map[string]any{"name": "MiniMax M2.7"},
		"mimo-v2.5-pro":   map[string]any{"name": "MiMo V2.5 Pro"},
		"auto:balance":    map[string]any{"name": "TokenRouter Auto Balance"},
		"custom-model":    map[string]any{"name": "Custom Gateway Model"},
	}
	for id, model := range extraModels {
		if strings.TrimSpace(id) == "" {
			continue
		}
		if _, exists := models[id]; exists {
			continue
		}
		models[id] = model
	}
	return map[string]any{
		"npm":  "@ai-sdk/openai-compatible",
		"name": "OmniProxy",
		"options": map[string]any{
			"baseURL":     baseURL,
			"apiKey":      localClientAPIKey,
			"setCacheKey": true,
		},
		"models": models,
	}
}

func openCodeGeminiProvider(baseURL string) map[string]any {
	return map[string]any{
		"npm":  "@ai-sdk/google",
		"name": "OmniProxy Gemini",
		"options": map[string]any{
			"baseURL": baseURL,
			"apiKey":  localClientAPIKey,
		},
		"models": map[string]any{
			"gemini-3-pro-preview":   map[string]any{"name": "Gemini 3 Pro Preview"},
			"gemini-3-flash-preview": map[string]any{"name": "Gemini 3 Flash Preview"},
			"gemini-2.5-flash-lite":  map[string]any{"name": "Gemini 2.5 Flash Lite"},
		},
	}
}

func openCodeOpenRouterProvider(baseURL string, models map[string]any) map[string]any {
	if len(models) == 0 {
		models = defaultOpenCodeOpenRouterModels()
	}
	return map[string]any{
		"npm":  "@ai-sdk/openai-compatible",
		"name": "OmniProxy OpenRouter",
		"options": map[string]any{
			"baseURL":     baseURL,
			"apiKey":      localClientAPIKey,
			"setCacheKey": true,
		},
		"models": models,
	}
}

func openCodeTokenRouterProvider(baseURL string) map[string]any {
	return map[string]any{
		"npm":  "@ai-sdk/openai-compatible",
		"name": "OmniProxy TokenRouter",
		"options": map[string]any{
			"baseURL":     baseURL,
			"apiKey":      localClientAPIKey,
			"setCacheKey": true,
		},
		"models": map[string]any{
			"auto:balance": map[string]any{"name": "Auto Balance"},
			"auto:quality": map[string]any{"name": "Auto Quality"},
			"auto:speed":   map[string]any{"name": "Auto Speed"},
			"auto:cost":    map[string]any{"name": "Auto Cost"},
		},
	}
}

func openCodeZoProvider(baseURL string) map[string]any {
	return map[string]any{
		"npm":  "@ai-sdk/openai-compatible",
		"name": "OmniProxy Zo Computer",
		"options": map[string]any{
			"baseURL":     baseURL,
			"apiKey":      localClientAPIKey,
			"setCacheKey": true,
		},
		"models": map[string]any{
			"claude-opus-4-7":   map[string]any{"name": "Zo Claude Opus 4.7"},
			"claude-sonnet-4-6": map[string]any{"name": "Zo Claude Sonnet 4.6"},
			"gemini-3.1-pro":    map[string]any{"name": "Zo Gemini 3.1 Pro"},
			"glm-5":             map[string]any{"name": "Zo GLM 5"},
			"minimax-2.7":       map[string]any{"name": "Zo MiniMax 2.7"},
			"gpt-5.4":           map[string]any{"name": "Zo GPT-5.4"},
			"gpt-5.4-mini":      map[string]any{"name": "Zo GPT-5.4 mini"},
			"gpt-5.5":           map[string]any{"name": "Zo GPT-5.5"},
			"deepseek-v4-pro":   map[string]any{"name": "Zo DeepSeek V4 Pro"},
		},
	}
}

func piZoModels() []map[string]any {
	return []map[string]any{
		{"id": "claude-opus-4-7", "name": "Zo Claude Opus 4.7"},
		{"id": "claude-sonnet-4-6", "name": "Zo Claude Sonnet 4.6"},
		{"id": "gemini-3.1-pro", "name": "Zo Gemini 3.1 Pro"},
		{"id": "glm-5", "name": "Zo GLM 5"},
		{"id": "minimax-2.7", "name": "Zo MiniMax 2.7"},
		{"id": "gpt-5.4", "name": "Zo GPT-5.4"},
		{"id": "gpt-5.4-mini", "name": "Zo GPT-5.4 mini"},
		{"id": "gpt-5.5", "name": "Zo GPT-5.5"},
		{"id": "deepseek-v4-pro", "name": "Zo DeepSeek V4 Pro"},
	}
}

func openCodeModelsFromOpenRouter(models []openRouterModelResponse) map[string]any {
	out := map[string]any{}
	for _, model := range models {
		id := strings.TrimSpace(model.ID)
		if id == "" {
			continue
		}
		name := strings.TrimSpace(model.Name)
		if name == "" {
			name = id
		}
		out[id] = map[string]any{"name": name}
	}
	if len(out) == 0 {
		return defaultOpenCodeOpenRouterModels()
	}
	return out
}

func defaultOpenCodeOpenRouterModels() map[string]any {
	return map[string]any{
		"openrouter/auto": map[string]any{"name": "OpenRouter Auto"},
	}
}

func openCodeCustomProvider(baseURL string) map[string]any {
	return map[string]any{
		"npm":  "@ai-sdk/openai-compatible",
		"name": "OmniProxy Custom Gateway",
		"options": map[string]any{
			"baseURL":     baseURL,
			"apiKey":      localClientAPIKey,
			"setCacheKey": true,
		},
		"models": map[string]any{
			"custom-model": map[string]any{"name": "Custom Gateway Model"},
		},
	}
}
