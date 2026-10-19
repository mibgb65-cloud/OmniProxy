package subscription

import (
	"context"
	"omniproxy/internal/infrastructure/codex"
)

// Models 只使用本应用管理的授权目录，保持用户官方登录不变。
func (s *Service) Models(ctx context.Context, id string) ([]codex.Model, error) {
	if err := s.operations.Lock(ctx); err != nil {
		return nil, failure("operation_cancelled")
	}
	defer s.operations.Unlock()
	item, profile, err := s.accounts.SubscriptionProfile(ctx, id)
	if err != nil || item.AuthState != "authenticated" || s.accountBusy(id) {
		return nil, failure("invalid_account")
	}
	client, err := s.open(ctx, profile)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	identity, err := client.AccountID(ctx)
	if err != nil || item.AuthAccountID != "" && item.AuthAccountID != identity {
		return nil, failure("identity_changed")
	}
	if err := verifyMember(ctx, client, item); err != nil {
		return nil, err
	}
	catalog, ok := client.(interface {
		Models(context.Context) ([]codex.Model, error)
	})
	if !ok {
		return nil, failure("models_unavailable")
	}
	return catalog.Models(ctx)
}
