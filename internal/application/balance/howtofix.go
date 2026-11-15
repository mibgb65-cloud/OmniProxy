package balance

import (
	"context"
	"math/big"
	"strings"
	"time"

	"omniproxy/internal/application/account"
)

// readHowToFix 用账号 Key 读取 HowToFix 的预付余额；上游未返回已用额度时只显示余额。
func (s *Service) readHowToFix(ctx context.Context, item account.Item, baseURL string) (Info, error) {
	result := Info{}
	called := false
	err := s.accounts.UseAPIKey(ctx, item.ID, func(key []byte) error {
		called = true
		value, err := s.howtofix.ReadBalance(ctx, baseURL, key)
		if err != nil {
			return err
		}
		remaining, ok := new(big.Rat).SetString(value.Remaining)
		if !ok {
			return failure("invalid_response")
		}
		result.HowToFix, result.Available = &value, remaining.Sign() > 0
		return nil
	})
	if err != nil {
		if !called {
			return Info{}, failure("credential_unavailable")
		}
		code := strings.TrimPrefix(err.Error(), "balance: ")
		switch code {
		case "unauthorized", "forbidden", "rate_limited", "service_unavailable", "network_failed", "invalid_response", "credential_unavailable", "configuration_unavailable":
			return Info{}, failure(code)
		}
		return Info{}, failure("query_failed")
	}
	result.FetchedAt = time.Now().UTC().Format(time.RFC3339)
	return result, nil
}
