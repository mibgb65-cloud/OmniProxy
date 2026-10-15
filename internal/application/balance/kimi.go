package balance

import (
	"context"
	"math/big"
	"time"

	"omniproxy/internal/application/account"
)

func (s *Service) readKimi(ctx context.Context, item account.Item) (Info, error) {
	if item.Kimi == nil || !item.Kimi.Valid() {
		return Info{}, failure("configuration_unavailable")
	}
	access, ok := s.accounts.(interface {
		UseAPIKeyAt(context.Context, string, string, func([]byte) error) error
	})
	if !ok {
		return Info{}, failure("configuration_unavailable")
	}
	result := Info{}
	called := false
	err := access.UseAPIKeyAt(ctx, item.ID, item.UpdatedAt, func(key []byte) error {
		called = true
		value, err := s.kimi.ReadBalance(ctx, *item.Kimi, key)
		if err != nil {
			return err
		}
		amount, ok := new(big.Rat).SetString(value.Available)
		if !ok {
			return failure("invalid_response")
		}
		result.Kimi, result.Available = &value, amount.Sign() > 0
		return nil
	})
	if err != nil {
		if !called {
			return Info{}, failure("credential_unavailable")
		}
		switch err.Error() {
		case "balance: unauthorized", "balance: forbidden", "balance: rate_limited", "balance: service_unavailable", "balance: network_failed", "balance: invalid_response", "balance: credential_unavailable", "balance: configuration_unavailable":
			return Info{}, err
		}
		return Info{}, failure("query_failed")
	}
	// 地区与密钥都属于查询身份，在途变更不能把旧平台余额显示到新配置。
	current, err := s.accounts.Metadata(ctx, item.ID)
	if err != nil || current.UpdatedAt != item.UpdatedAt || current.Kimi == nil || *current.Kimi != *item.Kimi {
		return Info{}, failure("configuration_unavailable")
	}
	result.FetchedAt = time.Now().UTC().Format(time.RFC3339)
	return result, nil
}
