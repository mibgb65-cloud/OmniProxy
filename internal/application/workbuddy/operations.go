package workbuddy

import (
	"context"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/provider/workbuddyai"
)

// credentials 读取账号令牌，临近到期时续期；force 时同时核对远端身份。
func (s *Service) credentials(ctx context.Context, id string, force bool) (workbuddyai.Credentials, error) {
	if s.busy(id) {
		return workbuddyai.Credentials{}, workbuddyai.Failure("busy")
	}
	build, item, err := s.buildOf(ctx, id)
	if err != nil {
		return workbuddyai.Credentials{}, err
	}
	client := s.clientFor(build)
	scoped := item.OutboundContext(ctx)
	var current workbuddyai.Credentials
	err = s.accounts.UseWorkBuddyCredentials(ctx, id, func(item account.Item, cred workbuddyai.Credentials) error {
		if force || cred.ExpiresAt.IsZero() || time.Until(cred.ExpiresAt) < refreshMargin {
			operation, cancel := context.WithTimeout(scoped, operationTimeout)
			defer cancel()
			token, err := client.Refresh(operation, cred)
			if err != nil {
				return err
			}
			next := credentialsFrom(token, workbuddyai.Identity{UID: cred.UID, EnterpriseID: cred.EnterpriseID, Name: cred.AccountName, Plan: cred.Plan})
			if force {
				identity, err := client.ReadIdentity(operation, token)
				if err != nil {
					return err
				}
				if identity.UID != cred.UID {
					return workbuddyai.Failure("identity_changed")
				}
				next = credentialsFrom(token, identity)
			}
			if _, err := s.accounts.RotateWorkBuddyCredentials(ctx, item, next); err != nil {
				return err
			}
			current = next
			return nil
		}
		current = cred
		return nil
	})
	return current, err
}

// Use 在受控回调内取得短暂令牌快照，推理不占用全局刷新锁。
func (s *Service) Use(ctx context.Context, id string, consume func(workbuddyai.Credentials) error) error {
	return s.UseAt(ctx, id, "", consume)
}

// UseAt 在刷新互斥门内复核账号版本，普通令牌轮换不改变账号池身份。
func (s *Service) UseAt(ctx context.Context, id, expectedAt string, consume func(workbuddyai.Credentials) error) error {
	s.ops.Lock()
	if expectedAt != "" {
		item, err := s.accounts.Metadata(ctx, id)
		if err != nil || item.UpdatedAt != expectedAt {
			s.ops.Unlock()
			return workbuddyai.Failure("configuration_changed")
		}
	}
	cred, err := s.credentials(ctx, id, false)
	s.ops.Unlock()
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return workbuddyai.Failure("cancelled")
	}
	return consume(cred)
}

// Models 返回与当前凭据版本绑定的账号目录。
func (s *Service) Models(ctx context.Context, id string) ([]workbuddyai.Model, string, error) {
	build, item, err := s.buildOf(ctx, id)
	if err != nil {
		return nil, "", err
	}
	client := s.clientFor(build)
	s.ops.Lock()
	cred, err := s.credentials(ctx, id, false)
	version := ""
	if err == nil {
		version, err = s.accounts.CatalogVersion(ctx, id)
	}
	s.ops.Unlock()
	if err != nil {
		return nil, "", err
	}
	operation, cancel := context.WithTimeout(item.OutboundContext(ctx), operationTimeout)
	defer cancel()
	models, err := client.DiscoverModels(operation, cred)
	return models, version, err
}

// Refresh 主动续期并核对远端身份，失败不会被误报为成功。
func (s *Service) Refresh(ctx context.Context, id string) (account.Item, error) {
	s.ops.Lock()
	defer s.ops.Unlock()
	if _, err := s.credentials(ctx, id, true); err != nil {
		return account.Item{}, err
	}
	s.notify()
	return s.accounts.Metadata(ctx, id)
}

// Logout 清除本机授权，不修改用户在 WorkBuddy 应用里的登录。
func (s *Service) Logout(ctx context.Context, id string) (account.Item, error) {
	s.ops.Lock()
	defer s.ops.Unlock()
	if s.busy(id) {
		return account.Item{}, workbuddyai.Failure("busy")
	}
	item, err := s.accounts.LogoutWorkBuddySubscription(ctx, id)
	s.forgetQuota(id)
	s.notify()
	return item, err
}

// Delete 通过同一互斥门移除账号与受保护令牌。
func (s *Service) Delete(ctx context.Context, id string) error {
	s.ops.Lock()
	defer s.ops.Unlock()
	if s.busy(id) {
		return workbuddyai.Failure("busy")
	}
	item, err := s.accounts.Metadata(ctx, id)
	if err != nil || s.accounts.CheckWorkBuddyProvider(ctx, item.ProviderID) != nil || item.AuthMethod != account.AuthWorkBuddyLogin {
		return workbuddyai.Failure("invalid_input")
	}
	if _, err = s.accounts.LogoutWorkBuddySubscription(ctx, id); err != nil {
		s.notify()
		return err
	}
	err = s.accounts.DeleteManaged(ctx, id)
	s.forgetQuota(id)
	s.notify()
	return err
}
