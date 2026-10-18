package googlesubscription

import (
	"context"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/provider/antigravity"
)

func (s *Service) credentials(ctx context.Context, id string, force bool) (antigravity.Credentials, error) {
	if s.busy(id) {
		return antigravity.Credentials{}, antigravity.Failure("busy")
	}
	var current antigravity.Credentials
	err := s.accounts.UseGoogleCredentials(ctx, id, func(item account.Item, cred antigravity.Credentials) error {
		if force || time.Until(cred.ExpiresAt) < 5*time.Minute {
			operation, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			if err := s.client.Refresh(operation, &cred); err != nil {
				return err
			}
			if force {
				if err := s.client.Identity(operation, &cred); err != nil {
					return err
				}
				if err := s.client.Project(operation, &cred); err != nil {
					return err
				}
			}
			if _, err := s.accounts.RotateGoogleCredentials(ctx, item, cred); err != nil {
				return err
			}
		}
		current = cred
		return nil
	})
	return current, err
}

// Use 在受控回调获取短暂 token 快照，推理不占用全局刷新锁。
func (s *Service) Use(ctx context.Context, id string, consume func(antigravity.Credentials) error) error {
	return s.UseAt(ctx, id, "", consume)
}

// UseAt 在刷新互斥门内复核配置，普通 token 轮换不改变账号池身份。
func (s *Service) UseAt(ctx context.Context, id, expectedAt string, consume func(antigravity.Credentials) error) error {
	s.ops.Lock()
	if expectedAt != "" {
		item, err := s.accounts.Metadata(ctx, id)
		if err != nil || item.UpdatedAt != expectedAt {
			s.ops.Unlock()
			return antigravity.Failure("configuration_changed")
		}
	}
	cred, err := s.credentials(ctx, id, false)
	s.ops.Unlock()
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return antigravity.Failure("cancelled")
	}
	return consume(cred)
}

// Models 返回与当前凭据版本绑定的真实账号目录。
func (s *Service) Models(ctx context.Context, id string) ([]antigravity.Model, string, error) {
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
	models, err := s.client.Models(ctx, cred)
	return models, version, err
}

// Refresh 主动刷新并校验身份，错误不被误报为成功。
func (s *Service) Refresh(ctx context.Context, id string) (account.Item, error) {
	s.ops.Lock()
	defer s.ops.Unlock()
	if _, err := s.credentials(ctx, id, true); err != nil {
		return account.Item{}, err
	}
	s.notify()
	return s.accounts.Metadata(ctx, id)
}

// Logout 清除当前 OmniProxy 授权，不修改用户在官方应用的登录。
func (s *Service) Logout(ctx context.Context, id string) (account.Item, error) {
	s.ops.Lock()
	defer s.ops.Unlock()
	if s.busy(id) {
		return account.Item{}, antigravity.Failure("busy")
	}
	item, err := s.accounts.LogoutGoogleSubscription(ctx, id)
	s.notify()
	return item, err
}

// Delete 通过同一互斥门移除 Google 订阅账号与受保护授权。
func (s *Service) Delete(ctx context.Context, id string) error {
	s.ops.Lock()
	defer s.ops.Unlock()
	if s.busy(id) {
		return antigravity.Failure("busy")
	}
	item, err := s.accounts.Metadata(ctx, id)
	if err != nil || s.accounts.CheckGoogleProvider(ctx, item.ProviderID) != nil || item.AuthMethod != account.AuthSubscription {
		return antigravity.Failure("invalid_input")
	}
	if _, err = s.accounts.LogoutGoogleSubscription(ctx, id); err != nil {
		s.notify()
		return err
	}
	err = s.accounts.DeleteManaged(ctx, id)
	s.notify()
	return err
}
