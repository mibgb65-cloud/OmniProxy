package workbuddyai

import (
	"encoding/json"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	// maxCredentialBytes 与受保护存储的上限一致：Marshal 结果不得超过 1 MiB。
	maxCredentialBytes = 1 << 20
	// maxTokenBytes 限制 access/refresh token；上游令牌都是短 ASCII。
	maxTokenBytes = 8192
	// 以下上限约束可展示身份字段，避免异常上游把超长文本写进账号表。
	maxUIDBytes         = 128
	maxDomainBytes      = 255
	maxTokenTypeBytes   = 64
	maxEnterpriseBytes  = 128
	maxAccountNameBytes = 200
	maxPlanBytes        = 80
)

// Credentials 是受保护存储中的 WorkBuddy AI 登录态；只在受控回调内短暂解密。
type Credentials struct {
	AccessToken      string    `json:"accessToken"`
	RefreshToken     string    `json:"refreshToken"`
	ExpiresAt        time.Time `json:"expiresAt,omitempty"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt,omitempty"`
	Domain           string    `json:"domain,omitempty"`
	TokenType        string    `json:"tokenType,omitempty"`
	UID              string    `json:"uid"`
	EnterpriseID     string    `json:"enterpriseId,omitempty"`
	AccountName      string    `json:"accountName,omitempty"`
	Plan             string    `json:"plan,omitempty"`
}

// Valid 校验登录态的结构与边界：access 与 uid 必填，所有字段长度有界且字符可打印。
func (c Credentials) Valid() bool {
	return safeASCII(c.AccessToken, maxTokenBytes, false) &&
		safeASCII(c.RefreshToken, maxTokenBytes, true) &&
		safeASCII(c.UID, maxUIDBytes, false) &&
		safeASCII(c.Domain, maxDomainBytes, true) &&
		safeASCII(c.TokenType, maxTokenTypeBytes, true) &&
		safeASCII(c.EnterpriseID, maxEnterpriseBytes, true) &&
		safeText(c.AccountName, maxAccountNameBytes, true) &&
		safeText(c.Plan, maxPlanBytes, true)
}

// SignedIn 只报告是否有访问令牌；令牌是否仍被上游接受由 401 判定。
func (c Credentials) SignedIn() bool { return c.AccessToken != "" }

// Marshal 序列化为受保护存储的正文；不包含任何调试信息。
func (c Credentials) Marshal() ([]byte, error) {
	if !c.Valid() {
		return nil, Failure(CodeCredentialUnavailable)
	}
	data, err := json.Marshal(c)
	if err != nil || len(data) > maxCredentialBytes {
		return nil, Failure(CodeCredentialUnavailable)
	}
	return data, nil
}

// ParseCredentials 解析受保护存储中的正文，拒绝超限、结构不符或含控制字符的数据。
func ParseCredentials(data []byte) (Credentials, error) {
	if len(data) == 0 || len(data) > maxCredentialBytes {
		return Credentials{}, Failure(CodeCredentialUnavailable)
	}
	var credentials Credentials
	if json.Unmarshal(data, &credentials) != nil || !credentials.Valid() {
		return Credentials{}, Failure(CodeCredentialUnavailable)
	}
	return credentials, nil
}

// Clone 返回独立副本；结构里只有值类型，复制后不共享可变状态。
func (c Credentials) Clone() Credentials { return c }

// safeASCII 校验可选或必填的短 ASCII 字段：长度有界且全部为可打印字符。
func safeASCII(value string, max int, optional bool) bool {
	if value == "" {
		return optional
	}
	if len(value) > max {
		return false
	}
	for _, char := range value {
		if char < 33 || char > 126 {
			return false
		}
	}
	return true
}

// safeText 校验可选或必填的 UTF-8 文本字段：长度有界且不含控制字符。
func safeText(value string, max int, optional bool) bool {
	if value == "" {
		return optional
	}
	if len(value) > max || !utf8.ValidString(value) {
		return false
	}
	return !containsControl(value)
}

func containsControl(value string) bool {
	for _, char := range value {
		if unicode.IsControl(char) {
			return true
		}
	}
	return false
}
