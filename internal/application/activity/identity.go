package activity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"omniproxy/internal/core/proxy"
	"strings"
)

// 同 Team 工作区不同邮箱是不同成员；账号删除重建同一身份可继续合并历史。
func subject(target proxy.TargetSnapshot) (string, error) {
	parts := []string{target.ProviderKind, "account", target.AccountID}
	if target.ProviderKind == "kimi" {
		parts = append(parts, target.ProviderRegion)
	}
	if target.ProviderKind == "stepfun" {
		parts = append(parts, target.ProviderRegion, target.BillingMode)
	}
	if target.AuthMethod == "subscription_login" {
		email := strings.ToLower(strings.TrimSpace(target.Email))
		if email == "" || target.WorkspaceID == "" {
			return "", ErrInvalid
		}
		parts = []string{target.ProviderKind, "subscription", email, target.WorkspaceID}
	}
	if target.AccountID == "" {
		return "", nil
	}
	raw, _ := json.Marshal(parts)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}
