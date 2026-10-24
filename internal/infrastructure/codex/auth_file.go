package codex

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
)

// AuthFile 是网关自己保管的登录材料，对应 v1 池里的 auth.json：
// 访问令牌、续期令牌、官方账号标识与展示用的邮箱／套餐。
type AuthFile struct {
	AccessToken  string
	RefreshToken string
	AccountID    string
	Email        string
	Plan         string
}

// ParseAuthFile 只读取网关需要维护的字段；不可用的授权必须明确报错。
func ParseAuthFile(data []byte) (AuthFile, error) {
	access, refresh, accountID, ok := accessTokenFromAuth(string(data))
	if !ok {
		return AuthFile{}, ErrProtocol
	}
	auth, tokens, _, err := parseAuth(string(data))
	if err != nil {
		return AuthFile{}, err
	}
	idToken := mapValue(tokens, "id_token")
	if idToken == "" {
		idToken = mapValue(auth, "id_token")
	}
	claims := idTokenClaims(idToken)
	result := AuthFile{AccessToken: access, RefreshToken: refresh, AccountID: accountID}
	result.Email = claimString(claims, "email")
	if nested, ok := claims["https://api.openai.com/auth"].(map[string]any); ok {
		if result.Email == "" {
			result.Email = claimString(nested, "email")
		}
		result.Plan = claimString(nested, "chatgpt_plan_type")
	}
	if result.Email == "" {
		result.Email = mapValue(auth, "email")
	}
	return result, nil
}

// ClientAuthFileFor 生成客户端可直接加载的 auth.json，并抹掉混合登录遗留的 API Key。
func ClientAuthFileFor(data []byte, accountID string) ([]byte, error) {
	return ClientAuthFile(data, accountID)
}

// idTokenClaims 只解析载荷，不校验签名：签名由官方服务端负责，这里只取展示信息。
func idTokenClaims(token string) map[string]any {
	if token == "" {
		return nil
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil
	}
	var claims map[string]any
	if json.NewDecoder(bytes.NewReader(payload)).Decode(&claims) != nil {
		return nil
	}
	return claims
}

func claimString(claims map[string]any, key string) string {
	value, ok := claims[key].(string)
	if !ok || len(value) > 320 {
		return ""
	}
	return strings.TrimSpace(value)
}
