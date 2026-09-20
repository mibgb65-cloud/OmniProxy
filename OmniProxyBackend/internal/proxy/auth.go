package proxy

import (
	"errors"
	"net/http"
	"strings"

	"omniproxy/internal/token"
)

func applyAuth(header http.Header, selected token.Token) error {
	return applyAuthWithProtocol(header, selected, "")
}

func applyRouteAuth(header http.Header, selected token.Token, route routeInfo) error {
	return applyAuthWithProtocol(header, selected, route.Protocol)
}

func applyAuthWithProtocol(header http.Header, selected token.Token, protocol string) error {
	header.Del("Authorization")
	header.Del("X-Api-Key")
	header.Del("Api-Key")
	header.Del("X-Goog-Api-Key")
	header.Del("ChatGPT-Account-Id")

	secret, err := credentialSecret(selected)
	if err != nil {
		return err
	}

	if selected.CredentialType == token.CredentialTypeCodexAuthJSON {
		accountID, ok := codexAccountID(selected.TokenValue)
		if ok {
			header.Set("ChatGPT-Account-Id", accountID)
		}
	}

	switch token.NormalizeProvider(selected.Provider) {
	case token.ProviderAnthropic:
		if selected.CredentialType == token.CredentialTypeClaudeOAuth {
			header.Set("Authorization", "Bearer "+secret)
			appendHeaderValue(header, "anthropic-beta", "oauth-2025-04-20")
			appendHeaderValue(header, "anthropic-beta", "claude-code-20250219")
		} else {
			header.Set("x-api-key", secret)
		}
		if header.Get("anthropic-version") == "" {
			header.Set("anthropic-version", "2023-06-01")
		}
	case token.ProviderDeepSeek, token.ProviderZhipu, token.ProviderMiniMax, token.ProviderAtria, token.ProviderCustom:
		if protocol == "anthropic" {
			header.Set("x-api-key", secret)
			if header.Get("anthropic-version") == "" {
				header.Set("anthropic-version", "2023-06-01")
			}
		} else {
			header.Set("Authorization", "Bearer "+secret)
		}
	case token.ProviderKimi:
		if protocol == "anthropic" {
			header.Set("x-api-key", secret)
			if header.Get("anthropic-version") == "" {
				header.Set("anthropic-version", "2023-06-01")
			}
		} else {
			header.Set("Authorization", "Bearer "+secret)
		}
	case token.ProviderXiaomi:
		header.Set("api-key", secret)
	case token.ProviderGemini:
		header.Set("x-goog-api-key", secret)
	case token.ProviderSub2API, token.ProviderNewAPI:
		switch protocol {
		case "anthropic":
			header.Set("x-api-key", secret)
			if header.Get("anthropic-version") == "" {
				header.Set("anthropic-version", "2023-06-01")
			}
		case "gemini":
			header.Set("x-goog-api-key", secret)
		default:
			header.Set("Authorization", "Bearer "+secret)
		}
	case token.ProviderAnyRouter:
		if protocol == "anthropic" {
			header.Set("x-api-key", secret)
			if header.Get("anthropic-version") == "" {
				header.Set("anthropic-version", "2023-06-01")
			}
		} else {
			header.Set("Authorization", "Bearer "+secret)
		}
	case token.ProviderFeatherless:
		header.Set("Authorization", "Bearer "+secret)
		if strings.TrimSpace(header.Get("HTTP-Referer")) == "" {
			header.Set("HTTP-Referer", "https://github.com/mibgb65-cloud/OmniProxy")
		}
		if strings.TrimSpace(header.Get("X-Title")) == "" {
			header.Set("X-Title", "OmniProxy")
		}
	default:
		header.Set("Authorization", "Bearer "+secret)
	}

	return nil
}

func credentialSecret(selected token.Token) (string, error) {
	credentialType := selected.CredentialType
	if credentialType == "" {
		credentialType = token.CredentialTypeAPIKey
	}

	if credentialType == token.CredentialTypeClaudeOAuth {
		fields, ok := token.ExtractClaudeOAuthFields(selected.TokenValue)
		if ok && strings.TrimSpace(fields.AccessToken) != "" {
			return strings.TrimSpace(fields.AccessToken), nil
		}
		return "", errors.New("claude OAuth JSON does not contain access_token")
	}

	if credentialType != token.CredentialTypeCodexAuthJSON {
		value := strings.TrimSpace(selected.TokenValue)
		if value == "" {
			return "", errors.New("empty token value")
		}
		return value, nil
	}

	if fields, ok := token.ExtractCodexAuthFields(selected.TokenValue); ok {
		if secret := fields.Secret(); secret != "" {
			return secret, nil
		}
	}

	return "", errors.New("codex auth.json does not contain a supported token field")
}

func appendHeaderValue(header http.Header, key string, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	existing := strings.TrimSpace(header.Get(key))
	if existing == "" {
		header.Set(key, value)
		return
	}
	for _, part := range strings.Split(existing, ",") {
		if strings.EqualFold(strings.TrimSpace(part), value) {
			return
		}
	}
	header.Set(key, existing+", "+value)
}

func codexAccountID(raw string) (string, bool) {
	fields, ok := token.ExtractCodexAuthFields(raw)
	accountID := strings.TrimSpace(fields.AccountID)
	return accountID, ok && accountID != ""
}

func findStringField(value any, wanted string) (string, bool) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if strings.EqualFold(key, wanted) {
				if text, ok := child.(string); ok {
					return text, true
				}
			}
			if text, ok := findStringField(child, wanted); ok {
				return text, true
			}
		}
	case []any:
		for _, child := range typed {
			if text, ok := findStringField(child, wanted); ok {
				return text, true
			}
		}
	}
	return "", false
}
