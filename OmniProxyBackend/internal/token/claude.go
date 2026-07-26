package token

import (
	"encoding/json"
	"strings"
)

type ClaudeOAuthFields struct {
	AccessToken  string
	RefreshToken string
	Email        string
	AccountID    string
}

func ExtractClaudeOAuthFields(raw string) (ClaudeOAuthFields, bool) {
	var payload map[string]any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return ClaudeOAuthFields{}, false
	}

	fields := ClaudeOAuthFields{
		AccessToken:  firstStringField(payload, "access_token", "accessToken"),
		RefreshToken: firstStringField(payload, "refresh_token", "refreshToken"),
		Email:        firstStringField(payload, "email"),
		AccountID:    firstStringField(payload, "account_uuid", "accountUuid"),
	}
	if nested, ok := mapField(payload, "claudeAiOauth"); ok {
		if fields.AccessToken == "" {
			fields.AccessToken = firstStringField(nested, "access_token", "accessToken")
		}
		if fields.RefreshToken == "" {
			fields.RefreshToken = firstStringField(nested, "refresh_token", "refreshToken")
		}
		if fields.Email == "" {
			fields.Email = firstStringField(nested, "email")
		}
		if fields.AccountID == "" {
			fields.AccountID = firstStringField(nested, "account_uuid", "accountUuid")
		}
	}
	if nested, ok := mapField(payload, "claude"); ok {
		if fields.AccessToken == "" {
			fields.AccessToken = firstStringField(nested, "access_token", "accessToken")
		}
		if fields.RefreshToken == "" {
			fields.RefreshToken = firstStringField(nested, "refresh_token", "refreshToken")
		}
		if fields.Email == "" {
			fields.Email = firstStringField(nested, "email")
		}
		if fields.AccountID == "" {
			fields.AccountID = firstStringField(nested, "account_uuid", "accountUuid")
		}
	}

	return fields, fields.AccessToken != "" || fields.RefreshToken != ""
}

func firstStringField(payload map[string]any, names ...string) string {
	for _, name := range names {
		if value, ok := payload[name]; ok {
			if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
				return strings.TrimSpace(text)
			}
		}
	}
	return ""
}

func mapField(payload map[string]any, name string) (map[string]any, bool) {
	value, ok := payload[name]
	if !ok {
		return nil, false
	}
	nested, ok := value.(map[string]any)
	return nested, ok
}
