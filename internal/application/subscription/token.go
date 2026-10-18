package subscription

import (
	"context"
	"omniproxy/internal/infrastructure/codex"
)

// UseToken 先让官方组件刷新，再在后端回调内短暂读取隔离系统凭据。
func (s *Service) UseToken(ctx context.Context, id string, consume func([]byte, string) error) error {
	return s.UseTokenAt(ctx, id, "", consume)
}

// UseTokenAt 在等待凭据锁后复核路由版本，防止重新登录的身份沿用旧目标快照。
func (s *Service) UseTokenAt(ctx context.Context, id, expectedAt string, consume func([]byte, string) error) error {
	s.operations.Lock()
	defer s.operations.Unlock()
	item, profile, err := s.accounts.SubscriptionProfile(ctx, id)
	if err != nil || !item.Enabled || item.AuthState != "authenticated" || s.accountBusy(id) {
		return failure("invalid_account")
	}
	if expectedAt != "" && item.UpdatedAt != expectedAt {
		return failure("identity_changed")
	}
	client, err := s.open(ctx, profile)
	if err != nil {
		return err
	}
	identity, err := client.Account(ctx, true)
	client.Close()
	if err != nil || identity == nil {
		return failure("reauth_required")
	}
	if !sameEmail(item.AuthEmail, identity.Email) || item.AuthAccountID == "" {
		return failure("identity_changed")
	}
	path, err := s.profilePath(profile)
	if err != nil {
		return err
	}
	return codex.UseToken(path, item.AuthAccountID, consume)
}
