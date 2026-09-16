package config

import (
	"strconv"
	"strings"
)

func normalizeOutboundProxyModels(models []string) []string {
	if len(models) == 0 {
		return []string{}
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(models))
	for _, model := range models {
		model = strings.TrimSpace(model)
		if model == "" {
			continue
		}
		key := strings.ToLower(model)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, model)
	}
	return out
}

func normalizeOutboundProxyProviders(providers []string) []string {
	if len(providers) == 0 {
		return []string{}
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(providers))
	for _, item := range providers {
		value := normalizeOutboundProxyProvider(item)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func normalizeOutboundProxyProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "openai", "codex":
		return "openai"
	case "anthropic", "claude":
		return "anthropic"
	case "deepseek":
		return "deepseek"
	case "kimi":
		return "kimi"
	case "xiaomi", "mimo":
		return "xiaomi"
	case "zhipu", "glm":
		return "zhipu"
	case "minimax":
		return "minimax"
	case "gemini", "google":
		return "gemini"
	case "openrouter":
		return "openrouter"
	case "tokenrouter":
		return "tokenrouter"
	case "sub2api":
		return "sub2api"
	case "newapi", "new-api", "new api":
		return "newapi"
	case "anyrouter", "any-router", "any router":
		return "anyrouter"
	case "zo", "zocomputer", "zo-computer":
		return "zo"
	case "prem", "premai", "prem-ai", "prem ai":
		return "prem"
	case "atria", "atria-asi", "atria asi":
		return "atria"
	case "custom":
		return "custom"
	default:
		return ""
	}
}

func providersFromOutboundProxyModels(models []string) []string {
	normalizedModels := normalizeOutboundProxyModels(models)
	if sameStringSet(normalizedModels, defaultOutboundProxyModels) {
		return append([]string(nil), defaultOutboundProxyProviders...)
	}
	providers := make([]string, 0)
	for _, raw := range normalizedModels {
		model := strings.ToLower(strings.TrimSpace(raw))
		switch {
		case model == "gpt-*" || strings.HasPrefix(model, "gpt-"):
			providers = append(providers, "openai")
		case model == "claude-*" || strings.HasPrefix(model, "claude-"):
			providers = append(providers, "anthropic")
		case model == "gemini-*" || strings.HasPrefix(model, "gemini-"):
			providers = append(providers, "gemini")
		case strings.HasPrefix(model, "prem:") || strings.HasPrefix(model, "prem/") || strings.HasPrefix(model, "premai/"):
			providers = append(providers, "prem")
		case model == "*/*" || strings.Contains(model, "/"):
			providers = append(providers, "openrouter")
		case strings.HasPrefix(model, "deepseek-"):
			providers = append(providers, "deepseek")
		case strings.HasPrefix(model, "kimi-"):
			providers = append(providers, "kimi")
		case strings.HasPrefix(model, "glm-") || strings.HasPrefix(model, "zhipu-"):
			providers = append(providers, "zhipu")
		case strings.HasPrefix(model, "minimax-"):
			providers = append(providers, "minimax")
		case strings.HasPrefix(model, "mimo-"):
			providers = append(providers, "xiaomi")
		case strings.HasPrefix(model, "auto:") || strings.HasPrefix(model, "tokenrouter:") || strings.HasPrefix(model, "tokenrouter/"):
			providers = append(providers, "tokenrouter")
		case strings.HasPrefix(model, "custom-"):
			providers = append(providers, "custom")
		case strings.HasPrefix(model, "atria-"):
			providers = append(providers, "atria")
		}
	}
	return normalizeOutboundProxyProviders(providers)
}

func sameStringSet(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	seen := map[string]bool{}
	for _, item := range left {
		seen[strings.ToLower(strings.TrimSpace(item))] = true
	}
	for _, item := range right {
		if !seen[strings.ToLower(strings.TrimSpace(item))] {
			return false
		}
	}
	return true
}

func isPort(value string) bool {
	if value == "" {
		return false
	}
	port, err := strconv.Atoi(value)
	return err == nil && port > 0 && port <= 65535
}
