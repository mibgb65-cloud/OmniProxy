package codex

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

// accessTokenFromAuth 取出访问令牌、续期令牌与账号标识，供令牌缓存使用。
func accessTokenFromAuth(raw string) (string, string, string, bool) {
	_, tokens, _, err := parseAuth(raw)
	if err != nil {
		return "", "", "", false
	}
	access := mapValue(tokens, "access_token")
	account := mapValue(tokens, "account_id")
	if !validTokenValue(access) || account == "" || len(account) > 256 {
		return "", "", "", false
	}
	return access, mapValue(tokens, "refresh_token"), account, true
}

func parseAuth(raw string) (map[string]any, map[string]any, bool, error) {
	if len(raw) == 0 || len(raw) > maxAuthJSONBytes {
		return nil, nil, false, ErrProtocol
	}
	var auth map[string]any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&auth) != nil {
		return nil, nil, false, ErrProtocol
	}
	value, ok := auth["tokens"]
	if !ok {
		if mapValue(auth, "access_token") != "" || mapValue(auth, "refresh_token") != "" {
			return auth, auth, true, nil
		}
		return nil, nil, false, ErrProtocol
	}
	tokens, ok := value.(map[string]any)
	if !ok {
		return nil, nil, false, ErrProtocol
	}
	return auth, tokens, false, nil
}

func authClientID(auth, tokens map[string]any) string {
	for _, source := range []map[string]any{tokens, auth} {
		if clientID := mapValue(source, "client_id"); validTokenValue(clientID) {
			return clientID
		}
	}
	return oauthClientID
}

// validTokenValue 与读取路径保持同一条边界：可见 ASCII 且长度有界。
func validTokenValue(value string) bool {
	if len(value) == 0 || len(value) > 32768 {
		return false
	}
	for _, ch := range value {
		if ch < 33 || ch > 126 {
			return false
		}
	}
	return true
}

func mapValue(values map[string]any, key string) string {
	value, ok := values[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

// storedExpirySoon 读取已保存的过期字段，与 JWT 判断互相补充。
func storedExpirySoon(values map[string]any, now time.Time) bool {
	for _, key := range []string{"expired", "expires_at", "expiresAt"} {
		value, ok := values[key]
		if !ok {
			continue
		}
		if expiresAt, ok := parseExpiry(value); ok {
			return !expiresAt.After(now.Add(accessRefreshMargin))
		}
	}
	return false
}

func parseExpiry(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case string:
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if parsed, err := time.Parse(layout, strings.TrimSpace(typed)); err == nil {
				return parsed, true
			}
		}
	case json.Number:
		if seconds, err := typed.Int64(); err == nil && seconds > 0 {
			return time.Unix(seconds, 0), true
		}
	}
	return time.Time{}, false
}

// expiresSoon 解析访问令牌 JWT 的 exp；不是 JWT 时按"未过期"处理。
func expiresSoon(token string, now time.Time) bool {
	expiresAt, ok := jwtExpiry(token)
	if !ok {
		return false
	}
	return !expiresAt.After(now.Add(accessRefreshMargin))
}

func jwtExpiry(token string) (time.Time, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return time.Time{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return time.Time{}, false
	}
	var data struct {
		Exp json.Number `json:"exp"`
	}
	if json.NewDecoder(bytes.NewReader(payload)).Decode(&data) != nil || data.Exp == "" {
		return time.Time{}, false
	}
	seconds, err := data.Exp.Int64()
	if err != nil || seconds <= 0 {
		return time.Time{}, false
	}
	return time.Unix(seconds, 0), true
}
