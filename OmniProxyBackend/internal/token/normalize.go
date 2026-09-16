package token

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func normalizeRequest(req UpsertRequest) (string, string, string, string, string, string, error) {
	name := strings.TrimSpace(req.Name)
	provider := strings.TrimSpace(strings.ToLower(req.Provider))
	credentialType := strings.TrimSpace(strings.ToLower(req.CredentialType))
	value := strings.TrimSpace(req.TokenValue)

	provider, credentialType, err := NormalizeProviderAndCredential(provider, credentialType)
	if err != nil {
		return "", "", "", "", "", "", err
	}
	region, err := NormalizeRegion(provider, credentialType, req.Region)
	if err != nil {
		return "", "", "", "", "", "", err
	}
	baseURL, err := NormalizeBaseURL(provider, req.BaseURL, providerRequiresAccountBaseURL(provider))
	if err != nil {
		return "", "", "", "", "", "", err
	}

	if credentialType == CredentialTypeCodexAuthJSON {
		if !json.Valid([]byte(value)) {
			return "", "", "", "", "", "", errors.New("codex auth.json must be valid JSON")
		}
		fields, ok := ExtractCodexAuthFields(value)
		if !ok {
			return "", "", "", "", "", "", errors.New("codex auth.json must be a JSON object")
		}
		if fields.Type != "" && !strings.EqualFold(fields.Type, "codex") {
			return "", "", "", "", "", "", errors.New("codex auth.json type must be codex")
		}
		if strings.TrimSpace(fields.Email) == "" {
			return "", "", "", "", "", "", errors.New("codex auth.json does not contain email or an email in id_token")
		}
		if !fields.HasSupportedToken() {
			return "", "", "", "", "", "", errors.New("codex auth.json does not contain a supported token field")
		}
		name = fields.Email
	} else if credentialType == CredentialTypeClaudeOAuth {
		if !json.Valid([]byte(value)) {
			return "", "", "", "", "", "", errors.New("claude OAuth JSON must be valid JSON")
		}
		fields, ok := ExtractClaudeOAuthFields(value)
		if !ok {
			return "", "", "", "", "", "", errors.New("claude OAuth JSON must contain access_token or refresh_token")
		}
		if fields.Email != "" {
			name = fields.Email
		}
	} else if provider == ProviderXiaomi && credentialType == CredentialTypeAPIKey && !strings.HasPrefix(value, "sk-") {
		return "", "", "", "", "", "", errors.New("xiaomi pay-as-you-go API key must start with sk-")
	} else if provider == ProviderXiaomi && credentialType == CredentialTypeMimoTokenPlan && !strings.HasPrefix(value, "tp-") {
		return "", "", "", "", "", "", errors.New("xiaomi token plan API key must start with tp-")
	} else if provider == ProviderTokenRouter && credentialType == CredentialTypeAPIKey && !strings.HasPrefix(value, "tr_") {
		return "", "", "", "", "", "", errors.New("tokenrouter API key must start with tr_")
	} else if provider == ProviderAtria && credentialType == CredentialTypeAPIKey && !strings.HasPrefix(value, "atr_") {
		return "", "", "", "", "", "", errors.New("atria API key must start with atr_")
	} else if len(value) < 12 {
		return "", "", "", "", "", "", errors.New("token value is too short")
	}

	if name == "" {
		return "", "", "", "", "", "", errors.New("token name is required")
	}

	return name, provider, credentialType, region, baseURL, value, nil
}

func normalizeUpdateRequest(existing Token, req UpsertRequest) (string, string, string, string, string, string, error) {
	if strings.TrimSpace(req.TokenValue) != "" {
		return normalizeRequest(req)
	}

	provider, credentialType, err := NormalizeProviderAndCredential(req.Provider, req.CredentialType)
	if err != nil {
		return "", "", "", "", "", "", err
	}
	if provider != NormalizeProvider(existing.Provider) || credentialType != existing.CredentialType {
		return "", "", "", "", "", "", errors.New("token value is required when changing provider or credential type")
	}
	region, err := NormalizeRegion(provider, credentialType, req.Region)
	if err != nil {
		return "", "", "", "", "", "", err
	}
	baseURL, err := NormalizeBaseURL(provider, req.BaseURL, providerRequiresAccountBaseURL(provider))
	if err != nil {
		return "", "", "", "", "", "", err
	}

	name := strings.TrimSpace(req.Name)
	if credentialType == CredentialTypeCodexAuthJSON || credentialType == CredentialTypeClaudeOAuth {
		name = existing.Name
	}
	if name == "" {
		return "", "", "", "", "", "", errors.New("token name is required")
	}

	value := strings.TrimSpace(existing.TokenValue)
	if value == "" {
		return "", "", "", "", "", "", errors.New("existing token value is empty")
	}
	return name, provider, credentialType, region, baseURL, value, nil
}

func NormalizeProvider(provider string) string {
	normalized, _, err := NormalizeProviderAndCredential(provider, "")
	if err != nil {
		return ProviderOpenAI
	}
	return normalized
}

func NormalizeRegion(provider string, credentialType string, region string) (string, error) {
	provider, credentialType, err := NormalizeProviderAndCredential(provider, credentialType)
	if err != nil {
		return "", err
	}
	if provider != ProviderXiaomi || credentialType != CredentialTypeMimoTokenPlan {
		return "", nil
	}

	switch strings.ToLower(strings.TrimSpace(region)) {
	case "", MimoRegionCN, "china", "cn-mainland", "mainland":
		return MimoRegionCN, nil
	case MimoRegionSGP, "singapore":
		return MimoRegionSGP, nil
	case MimoRegionAMS, "eu", "europe", "european", "amsterdam":
		return MimoRegionAMS, nil
	case "global", "overseas", "foreign", "international", "intl":
		return MimoRegionSGP, nil
	default:
		return "", errors.New("unsupported xiaomi token plan region")
	}
}

func normalizeStoredRegion(provider string, credentialType string, region string) string {
	normalized, err := NormalizeRegion(provider, credentialType, region)
	if err != nil {
		if provider == ProviderXiaomi && credentialType == CredentialTypeMimoTokenPlan {
			return MimoRegionCN
		}
		return ""
	}
	return normalized
}

func NormalizeBaseURL(provider string, baseURL string, required bool) (string, error) {
	provider = NormalizeProvider(provider)
	baseURL = strings.TrimSpace(baseURL)
	if provider != ProviderSub2API && provider != ProviderNewAPI && provider != ProviderAnyRouter {
		return "", nil
	}
	label := provider
	if provider == ProviderNewAPI {
		label = "new-api"
	}
	if baseURL == "" {
		if required {
			return "", errors.New(label + " base url is required")
		}
		return "", nil
	}
	parsed, err := url.ParseRequestURI(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New(label + " base url must be a valid URL")
	}
	return strings.TrimRight(baseURL, "/"), nil
}

func providerRequiresAccountBaseURL(provider string) bool {
	provider = NormalizeProvider(provider)
	return provider == ProviderSub2API || provider == ProviderNewAPI || provider == ProviderAnyRouter
}

func normalizeStoredBaseURL(provider string, baseURL string) string {
	normalized, err := NormalizeBaseURL(provider, baseURL, false)
	if err != nil {
		return ""
	}
	return normalized
}

func NormalizeProviderAndCredential(provider string, credentialType string) (string, string, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))
	credentialType = strings.TrimSpace(strings.ToLower(credentialType))

	if provider == "" {
		provider = ProviderOpenAI
	}
	if provider == "codex" {
		provider = ProviderOpenAI
		if credentialType == "" {
			credentialType = CredentialTypeCodexAuthJSON
		}
	} else if provider == "premai" || provider == "prem-ai" || provider == "prem ai" {
		provider = ProviderPrem
	}

	switch provider {
	case ProviderOpenAI:
		if credentialType == "" {
			credentialType = CredentialTypeAPIKey
		}
		if credentialType != CredentialTypeAPIKey && credentialType != CredentialTypeCodexAuthJSON {
			return "", "", errors.New("unsupported OpenAI credential type")
		}
	case ProviderAnthropic:
		if credentialType == "" {
			credentialType = CredentialTypeAPIKey
		}
		if credentialType != CredentialTypeAPIKey && credentialType != CredentialTypeClaudeOAuth {
			return "", "", errors.New("anthropic supports API key or Claude OAuth JSON only")
		}
	case ProviderZhipu:
		if credentialType == "" {
			credentialType = CredentialTypeAPIKey
		}
		if credentialType != CredentialTypeAPIKey && credentialType != CredentialTypeCodingPlan {
			return "", "", errors.New("zhipu supports API key or Coding Plan key only")
		}
	case ProviderDeepSeek, ProviderKimi, ProviderMiniMax, ProviderGemini, ProviderOpenRouter, ProviderTokenRouter, ProviderSub2API, ProviderNewAPI, ProviderAnyRouter, ProviderZo, ProviderPrem, ProviderAtria, ProviderCustom:
		if credentialType == "" {
			credentialType = CredentialTypeAPIKey
		}
		if credentialType != CredentialTypeAPIKey {
			return "", "", errors.New("this provider currently supports API key only")
		}
	case ProviderXiaomi:
		if credentialType == "" {
			credentialType = CredentialTypeAPIKey
		}
		if credentialType != CredentialTypeAPIKey && credentialType != CredentialTypeMimoTokenPlan {
			return "", "", errors.New("xiaomi supports API key or Token Plan key only")
		}
	default:
		return "", "", errors.New("unsupported provider")
	}

	return provider, credentialType, nil
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
