package subscription

import (
	"context"
	"errors"

	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/codex"
)

type tokenContextError struct{ cause error }

func (e *tokenContextError) Error() string { return "subscription: operation_cancelled" }
func (e *tokenContextError) Unwrap() error { return e.cause }

// UseToken 按需读取／续期受保护授权，将短期凭据副本用于后端回调，结束即清零。
func (s *Service) UseToken(ctx context.Context, id string, consume func([]byte, string) error) error {
	return s.UseTokenAt(ctx, id, "", consume)
}

// UseTokenAt 在等待凭据锁后复核路由版本，防止重新登录的身份沿用旧目标快照。
func (s *Service) UseTokenAt(ctx context.Context, id, expectedAt string, consume func([]byte, string) error) error {
	if consume == nil {
		return failure("invalid_input")
	}
	// 普通 HTTP 的模型流没有握手期限，但授权准备仍须有独立上限。
	prepareCtx, finish := context.WithTimeout(ctx, operationTimeout)
	defer finish()
	key, officialID, err := s.prepareToken(prepareCtx, id, expectedAt)
	finish()
	if err != nil {
		return err
	}
	defer clear(key)
	if ctx.Err() != nil {
		return &tokenContextError{ctx.Err()}
	}
	// 模型响应可能持续数分钟；凭据已复制并核验，不能让整段网络流占用全局准备锁。
	return consume(key, officialID)
}

func (s *Service) prepareToken(ctx context.Context, id, expectedAt string) ([]byte, string, error) {
	stage := proxy.BeginSocketStage(ctx, "credential_queue")
	err := s.operations.LockAccount(ctx, id)
	stage(err)
	if err != nil {
		return nil, "", &tokenContextError{err}
	}
	defer s.operations.UnlockAccount(id)
	item, profile, err := s.accounts.SubscriptionProfile(ctx, id)
	if ctx.Err() != nil {
		return nil, "", &tokenContextError{ctx.Err()}
	}
	if err != nil || !item.Enabled || item.AuthState != "authenticated" || s.accountBusy(id) {
		return nil, "", failure("invalid_account")
	}
	if expectedAt != "" && item.UpdatedAt != expectedAt {
		return nil, "", failure("identity_changed")
	}
	if item.AuthEmail == "" || item.AuthAccountID == "" {
		return nil, "", failure("identity_changed")
	}
	path, err := s.profilePath(profile)
	if err != nil {
		return nil, "", err
	}
	var key []byte
	var officialID string
	// 读取器核对同一份受保护材料的邮箱及工作区；不再每次启动官方组件强制联网刷新。
	borrow := func(value []byte, accountID string) error {
		key, officialID = append([]byte(nil), value...), accountID
		return nil
	}
	err = s.readToken(ctx, path, item.AuthAccountID, item.AuthEmail, borrow)
	if errors.Is(err, codex.ErrRefreshRequired) {
		clear(key)
		key = nil
		client, openErr := s.open(ctx, profile)
		if openErr != nil {
			if ctx.Err() != nil {
				return nil, "", &tokenContextError{ctx.Err()}
			}
			return nil, "", openErr
		}
		identity, refreshErr := client.Account(ctx, true)
		client.Close()
		if ctx.Err() != nil {
			return nil, "", &tokenContextError{ctx.Err()}
		}
		if refreshErr != nil || identity == nil {
			return nil, "", failure("reauth_required")
		}
		if !sameEmail(item.AuthEmail, identity.Email) {
			return nil, "", failure("identity_changed")
		}
		// 续期结果由官方组件保存在原隔离存储；重新核对同一份实际材料。
		err = s.readToken(ctx, path, item.AuthAccountID, item.AuthEmail, borrow)
	}
	if err != nil {
		clear(key)
		if ctx.Err() != nil {
			return nil, "", &tokenContextError{ctx.Err()}
		}
		return nil, "", err
	}
	// 外部状态刷新或重新授权可能在准备期间修改版本；发布前必须再次核对。
	current, currentProfile, err := s.accounts.SubscriptionProfile(ctx, id)
	if err != nil || ctx.Err() != nil || currentProfile != profile || current.UpdatedAt != item.UpdatedAt ||
		!current.Enabled || current.AuthState != "authenticated" || s.accountBusy(id) {
		clear(key)
		if ctx.Err() != nil {
			return nil, "", &tokenContextError{ctx.Err()}
		}
		return nil, "", failure("identity_changed")
	}
	return key, officialID, nil
}
