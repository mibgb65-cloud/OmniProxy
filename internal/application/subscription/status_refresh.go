package subscription

import (
	"context"
	"errors"

	"omniproxy/internal/application/account"
)

// Refresh 读取官方组件托管的身份；普通状态刷新不强制进行 OAuth 续期。
func (s *Service) Refresh(ctx context.Context, id string) (account.Item, error) {
	readCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	if err := s.operations.LockAccount(readCtx, id); err != nil {
		return account.Item{}, refreshFailure(readCtx, true, err)
	}
	defer s.operations.UnlockAccount(id)
	if s.accountBusy(id) {
		return account.Item{}, failure("busy")
	}
	item, profile, err := s.accounts.SubscriptionProfile(readCtx, id)
	if err != nil {
		return account.Item{}, refreshFailure(readCtx, false, failure("invalid_account"))
	}
	client, err := s.open(readCtx, profile)
	if err != nil {
		return item, refreshFailure(readCtx, false, err)
	}
	defer client.Close()
	// refreshToken=true 会强制访问认证服务；网络失败不代表现有托管授权失效。
	identity, err := client.Account(readCtx, false)
	if err != nil {
		return item, refreshFailure(readCtx, false, err)
	}
	meta := account.SubscriptionMetadata{Email: item.AuthEmail, Plan: item.AuthPlan, State: "reauth_required", AccountID: item.AuthAccountID}
	var accountIDErr error
	if identity != nil {
		if item.AuthEmail != "" && !sameEmail(item.AuthEmail, identity.Email) {
			return item, failure("identity_changed")
		}
		meta.Email, meta.Plan, meta.State = identity.Email, identity.Plan, "authenticated"
		// 已知工作区不重复调用附加用量接口。实际额度／推理仍独立核验成员和工作区，
		// 普通状态刷新只在旧账号缺少标识时尝试补全，避免无意义的网络失败。
		if meta.AccountID == "" {
			meta.AccountID, accountIDErr = readAccountID(readCtx, client)
		}
	}
	if err := readCtx.Err(); err != nil {
		return item, refreshFailure(readCtx, false, err)
	}
	updated, err := s.accounts.UpdateSubscription(readCtx, id, meta)
	if err != nil {
		return item, refreshFailure(readCtx, false, failure("account_update_failed"))
	}
	return updated, accountIDErr
}

func refreshFailure(ctx context.Context, queued bool, cause error) error {
	switch {
	case errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(cause, context.DeadlineExceeded) || ErrorCode(cause) == "operation_timeout":
		if queued {
			return failure("refresh_busy")
		}
		return failure("refresh_timeout")
	case errors.Is(ctx.Err(), context.Canceled) || errors.Is(cause, context.Canceled):
		return failure("operation_cancelled")
	}
	switch code := ErrorCode(cause); code {
	case "codex_unavailable", "initialization_failed", "invalid_account", "account_update_failed", "identity_changed", "busy", "operation_cancelled":
		return failure(code)
	default:
		return failure("refresh_unavailable")
	}
}
