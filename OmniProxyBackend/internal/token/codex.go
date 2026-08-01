package token

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

type CodexAuthFields struct {
	Type         string
	Email        string
	PlanType     string
	IDToken      string
	AccessToken  string
	RefreshToken string
	AccountID    string
	OpenAIAPIKey string
}

func (f CodexAuthFields) Secret() string {
	switch {
	case strings.TrimSpace(f.AccessToken) != "":
		return strings.TrimSpace(f.AccessToken)
	case strings.TrimSpace(f.OpenAIAPIKey) != "":
		return strings.TrimSpace(f.OpenAIAPIKey)
	case strings.TrimSpace(f.IDToken) != "":
		return strings.TrimSpace(f.IDToken)
	default:
		return ""
	}
}

func (f CodexAuthFields) HasSupportedToken() bool {
	return f.Secret() != ""
}

func ExtractCodexEmail(authJSON string) (string, bool) {
	fields, ok := ExtractCodexAuthFields(authJSON)
	if !ok || strings.TrimSpace(fields.Email) == "" {
		return "", false
	}
	return strings.TrimSpace(fields.Email), true
}

func CodexAccountID(item Token) string {
	if NormalizeProvider(item.Provider) != ProviderOpenAI || item.CredentialType != CredentialTypeCodexAuthJSON {
		return ""
	}
	fields, ok := ExtractCodexAuthFields(item.TokenValue)
	if !ok {
		return ""
	}
	return strings.TrimSpace(fields.AccountID)
}

func DisplayName(item Token) string {
	name := strings.TrimSpace(item.Name)
	if accountID := CodexAccountID(item); accountID != "" {
		if name == "" {
			return "account_id: " + ShortAccountID(accountID)
		}
		return name + " (account_id: " + ShortAccountID(accountID) + ")"
	}
	return name
}

func ShortAccountID(accountID string) string {
	accountID = strings.TrimSpace(accountID)
	if len(accountID) <= 18 {
		return accountID
	}
	return accountID[:8] + "..." + accountID[len(accountID)-6:]
}

func codexAuthSameIdentity(left CodexAuthFields, right CodexAuthFields) bool {
	leftEmail := strings.TrimSpace(left.Email)
	rightEmail := strings.TrimSpace(right.Email)
	if leftEmail != "" && rightEmail != "" && !strings.EqualFold(leftEmail, rightEmail) {
		return false
	}

	leftAccountID := strings.TrimSpace(left.AccountID)
	rightAccountID := strings.TrimSpace(right.AccountID)
	if leftAccountID != "" && rightAccountID != "" {
		return leftAccountID == rightAccountID
	}
	return leftEmail != "" && rightEmail != "" && strings.EqualFold(leftEmail, rightEmail)
}

func ExtractCodexAuthFields(authJSON string) (CodexAuthFields, bool) {
	var data map[string]any
	if err := json.Unmarshal([]byte(authJSON), &data); err != nil {
		return CodexAuthFields{}, false
	}
	fields := codexAuthFieldsFromObject(data)

	if fields.Email == "" {
		if email, ok := emailFromCodexIDToken(fields.IDToken); ok {
			fields.Email = email
		}
	}
	if fields.AccountID == "" {
		if accountID, ok := accountIDFromCodexIDToken(fields.IDToken); ok {
			fields.AccountID = accountID
		}
	}
	if planType, ok := planTypeFromCodexIDToken(fields.IDToken); ok {
		fields.PlanType = planType
	}
	return fields, true
}

func codexAuthFieldsFromObject(data map[string]any) CodexAuthFields {
	if data == nil {
		return CodexAuthFields{}
	}
	tokens, _ := data["tokens"].(map[string]any)
	return CodexAuthFields{
		Type:         stringField(data, "type"),
		Email:        stringField(data, "email"),
		IDToken:      firstNonEmpty(stringField(tokens, "id_token"), stringField(data, "id_token")),
		AccessToken:  firstNonEmpty(stringField(tokens, "access_token"), stringField(data, "access_token")),
		RefreshToken: firstNonEmpty(stringField(tokens, "refresh_token"), stringField(data, "refresh_token")),
		AccountID:    firstNonEmpty(stringField(tokens, "account_id"), stringField(data, "account_id")),
		OpenAIAPIKey: stringField(data, "OPENAI_API_KEY"),
	}
}

func stringField(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	value, ok := data[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func emailFromCodexIDToken(idToken string) (string, bool) {
	payload, ok := decodeJWTPayload(idToken)
	if !ok {
		return "", false
	}

	if profile, ok := payload["https://api.openai.com/profile"].(map[string]any); ok {
		if email := stringField(profile, "email"); email != "" {
			return email, true
		}
	}
	if email := stringField(payload, "email"); email != "" {
		return email, true
	}
	return "", false
}

func accountIDFromCodexIDToken(idToken string) (string, bool) {
	payload, ok := decodeJWTPayload(idToken)
	if !ok {
		return "", false
	}

	if auth, ok := payload["https://api.openai.com/auth"].(map[string]any); ok {
		if accountID := stringField(auth, "chatgpt_account_id"); accountID != "" {
			return accountID, true
		}
	}
	if accountID := stringField(payload, "account_id"); accountID != "" {
		return accountID, true
	}
	return "", false
}

func planTypeFromCodexIDToken(idToken string) (string, bool) {
	payload, ok := decodeJWTPayload(idToken)
	if !ok {
		return "", false
	}

	if auth, ok := payload["https://api.openai.com/auth"].(map[string]any); ok {
		if planType := stringField(auth, "chatgpt_plan_type"); planType != "" {
			return planType, true
		}
	}
	return "", false
}

func decodeJWTPayload(jwt string) (map[string]any, bool) {
	parts := strings.Split(jwt, ".")
	if len(parts) != 3 {
		return nil, false
	}

	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		raw, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil, false
		}
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, false
	}
	return payload, true
}
