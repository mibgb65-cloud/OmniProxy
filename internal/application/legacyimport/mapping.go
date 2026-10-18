package legacyimport

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/legacy"
	"omniproxy/internal/provider/kimi"
	"omniproxy/internal/provider/mimo"
)

func displayName(name string, index int) string {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 256 || utf8.RuneCountInString(name) > 80 || !utf8.ValidString(name) || strings.ContainsFunc(name, unicode.IsControl) {
		return fmt.Sprintf("OmniProxy v1 #%d", index)
	}
	return name
}

func displayProvider(provider string) string {
	switch provider {
	case "", "openai", "codex", "anthropic", "deepseek", "kimi", "xiaomi", "zhipu", "minimax", "gemini", "openrouter", "featherless", "tokenrouter", "sub2api", "newapi", "anyrouter", "zo", "prem", "atria", "custom":
		if provider == "" {
			return "openai"
		}
		return provider
	}
	return "unknown"
}

func mapToken(token legacy.Token, endpoints map[string]string) (account.CreateInput, string) {
	var input account.CreateInput
	kind := token.Provider
	if kind == "" {
		kind = "openai"
	}
	if token.CredentialType == "codex_auth_json" || token.CredentialType == "claude_oauth_json" || kind == "codex" {
		return input, "login_required"
	}
	if token.CredentialType != "" && token.CredentialType != "api_key" && token.CredentialType != "mimo_token_plan" {
		return input, "unsupported"
	}
	field, fallback := "", ""
	var accepted []string
	switch kind {
	case "openai":
		field, fallback = "openaiBaseUrl", "https://api.openai.com"
		accepted = []string{fallback, fallback + "/v1"}
	case "anthropic":
		field, fallback = "anthropicBaseUrl", "https://api.anthropic.com"
		accepted = []string{fallback, fallback + "/v1"}
	case "deepseek":
		field, fallback = "deepseekBaseUrl", "https://api.deepseek.com"
		accepted = []string{fallback, fallback + "/v1"}
	case "atria":
		field, fallback = "atriaBaseUrl", "https://api.atria-asi.ai"
		accepted = []string{fallback, fallback + "/v1"}
	case "gemini":
		kind = "google"
		field, fallback = "geminiBaseUrl", "https://generativelanguage.googleapis.com"
		accepted = []string{fallback}
	case "zhipu":
		field, fallback = "zhipuBaseUrl", "https://open.bigmodel.cn/api/paas/v4"
		accepted = []string{fallback}
	case "kimi":
		field, fallback = "kimiBaseUrl", "https://api.kimi.com/coding"
		accepted = []string{fallback, "https://api.moonshot.cn/v1", "https://api.moonshot.ai/v1"}
		base := endpoint(endpoints, field, fallback)
		input.Kimi = &kimi.Options{Region: "cn", BillingMode: "coding"}
		if base == accepted[1] {
			input.Kimi.BillingMode = "payg"
		}
		if base == accepted[2] {
			input.Kimi.Region = "global"
			input.Kimi.BillingMode = "payg"
		}
	case "xiaomi":
		kind = "mimo"
		field, fallback = "xiaomiApiBaseUrl", "https://api.xiaomimimo.com/v1"
		accepted = []string{fallback}
		input.MiMo = &mimo.Options{BillingMode: "payg"}
		if token.CredentialType == "mimo_token_plan" {
			region := token.Region
			if region == "" {
				region = "cn"
			}
			if region != "cn" && region != "sgp" && region != "ams" {
				return input, "invalid"
			}
			input.MiMo = &mimo.Options{BillingMode: "token_plan", Region: region}
			field = "xiaomiTokenPlanBaseUrl"
			if region == "sgp" {
				field = "xiaomiTokenPlanSgpBaseUrl"
			}
			if region == "ams" {
				field = "xiaomiTokenPlanAmsBaseUrl"
			}
			fallback = "https://token-plan-" + region + ".xiaomimimo.com/v1"
			accepted = []string{fallback}
		}
	case "minimax":
		return input, "billing_confirmation"
	default:
		return input, "unsupported"
	}
	if token.CredentialType == "mimo_token_plan" && kind != "mimo" {
		return input, "unsupported"
	}
	if token.BaseURL != "" {
		return input, "endpoint_confirmation"
	}
	base := endpoint(endpoints, field, fallback)
	matched := false
	for _, candidate := range accepted {
		if base == candidate {
			matched = true
		}
	}
	if !matched {
		return input, "endpoint_confirmation"
	}
	input.ProviderID = "builtin_" + kind
	return input, "ready"
}

func endpoint(endpoints map[string]string, field, fallback string) string {
	base := strings.TrimRight(strings.TrimSpace(endpoints[field]), "/")
	if base == "" {
		return fallback
	}
	return base
}
