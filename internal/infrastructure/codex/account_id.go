package codex

import (
	"context"
	"errors"
)

// AccountID 从官方用量快照取得当前工作区身份，不需要向应用暴露任何令牌。
func (c *rpcClient) AccountID(ctx context.Context) (string, error) {
	var result struct {
		AccountID string `json:"accountId"`
	}
	if err := c.call(ctx, "account/rateLimits/read", map[string]any{}, &result); err != nil {
		return "", err
	}
	if result.AccountID == "" {
		return "", errors.New("subscription: account_id_pending")
	}
	if len(result.AccountID) > 128 {
		return "", ErrProtocol
	}
	for _, char := range result.AccountID {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-' || char == '_') {
			return "", ErrProtocol
		}
	}
	return result.AccountID, nil
}
