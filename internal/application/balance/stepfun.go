package balance

import (
	"context"
	"omniproxy/internal/application/account"
	"strings"
	"time"
)

func (s *Service) readStep(ctx context.Context, item account.Item) (Info, error) {
	if item.StepFun == nil || !item.StepFun.Valid() || item.StepFun.BillingMode != "payg" {
		return Info{}, failure("unsupported")
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
		balance, err := s.stepfun.ReadBalance(ctx, *item.StepFun, key)
		if err != nil {
			return err
		}
		result.StepFun = &balance
		return nil
	})
	if err != nil {
		if !called {
			return Info{}, failure("credential_unavailable")
		}
		code := strings.TrimPrefix(err.Error(), "stepfun: ")
		switch code {
		case "unauthorized", "forbidden", "rate_limited", "service_unavailable", "network_failed", "invalid_response", "credential_unavailable":
			return Info{}, failure(code)
		}
		return Info{}, failure("query_failed")
	}
	// 查询完成时再检查版本，不能把切换地区／Key 之前的余额展示为新账号资源。
	current, err := s.accounts.Metadata(ctx, item.ID)
	if err != nil || current.UpdatedAt != item.UpdatedAt {
		return Info{}, failure("configuration_unavailable")
	}
	result.FetchedAt = time.Now().UTC().Format(time.RFC3339)
	return result, nil
}
