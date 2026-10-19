package subscription

import "context"

// UseToken 先让官方组件刷新，再将短期凭据副本用于后端回调，结束即清零。
func (s *Service) UseToken(ctx context.Context, id string, consume func([]byte, string) error) error {
	return s.UseTokenAt(ctx, id, "", consume)
}

// UseTokenAt 在等待凭据锁后复核路由版本，防止重新登录的身份沿用旧目标快照。
func (s *Service) UseTokenAt(ctx context.Context, id, expectedAt string, consume func([]byte, string) error) error {
	if consume == nil {
		return failure("invalid_input")
	}
	key, officialID, err := s.prepareToken(ctx, id, expectedAt)
	if err != nil {
		return err
	}
	defer clear(key)
	if ctx.Err() != nil {
		return failure("operation_cancelled")
	}
	// 模型响应可能持续数分钟；凭据已复制并核验，不能让整段网络流占用全局准备锁。
	return consume(key, officialID)
}

func (s *Service) prepareToken(ctx context.Context, id, expectedAt string) ([]byte, string, error) {
	if err := s.operations.Lock(ctx); err != nil {
		return nil, "", failure("operation_cancelled")
	}
	defer s.operations.Unlock()
	item, profile, err := s.accounts.SubscriptionProfile(ctx, id)
	if err != nil || !item.Enabled || item.AuthState != "authenticated" || s.accountBusy(id) {
		return nil, "", failure("invalid_account")
	}
	if expectedAt != "" && item.UpdatedAt != expectedAt {
		return nil, "", failure("identity_changed")
	}
	client, err := s.open(ctx, profile)
	if err != nil {
		return nil, "", err
	}
	identity, err := client.Account(ctx, true)
	client.Close()
	if err != nil || identity == nil {
		return nil, "", failure("reauth_required")
	}
	if !sameEmail(item.AuthEmail, identity.Email) || item.AuthAccountID == "" {
		return nil, "", failure("identity_changed")
	}
	path, err := s.profilePath(profile)
	if err != nil {
		return nil, "", err
	}
	var key []byte
	var officialID string
	err = s.readToken(path, item.AuthAccountID, func(value []byte, accountID string) error {
		key, officialID = append([]byte(nil), value...), accountID
		return nil
	})
	if err != nil {
		clear(key)
		return nil, "", err
	}
	return key, officialID, nil
}
