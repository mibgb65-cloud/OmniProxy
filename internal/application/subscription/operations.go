package subscription

import (
	"context"
	"omniproxy/internal/application/account"
	"os"
)

// Refresh 由官方组件刷新系统凭证，并更新身份与需要重新授权的状态。
func (s *Service) Refresh(ctx context.Context, id string) (account.Item, error) {
	if err := s.operations.Lock(ctx); err != nil {
		return account.Item{}, failure("operation_cancelled")
	}
	defer s.operations.Unlock()
	if s.accountBusy(id) {
		return account.Item{}, failure("busy")
	}
	item, profile, err := s.accounts.SubscriptionProfile(ctx, id)
	if err != nil {
		return account.Item{}, failure("invalid_account")
	}
	client, err := s.open(ctx, profile)
	if err != nil {
		return item, err
	}
	defer client.Close()
	identity, err := client.Account(ctx, true)
	state := "authenticated"
	email, plan := item.AuthEmail, item.AuthPlan
	authAccountID := item.AuthAccountID
	var accountIDErr error
	if err != nil {
		state = "unknown"
	} else if identity == nil {
		state = "reauth_required"
	} else {
		if item.AuthEmail != "" && !sameEmail(item.AuthEmail, identity.Email) {
			return item, failure("identity_changed")
		}
		email, plan = identity.Email, identity.Plan
		var currentID string
		currentID, accountIDErr = readAccountID(ctx, client)
		if accountIDErr == nil {
			authAccountID = currentID
		}
	}
	updated, updateErr := s.accounts.UpdateSubscription(ctx, id, account.SubscriptionMetadata{Email: email, Plan: plan, State: state, AccountID: authAccountID})
	if updateErr != nil {
		return account.Item{}, failure("account_update_failed")
	}
	if err != nil {
		return updated, err
	}
	if accountIDErr != nil {
		return updated, accountIDErr
	}
	return updated, nil
}

// Logout 清除该账号系统凭证，保留元数据供重新授权。
func (s *Service) Logout(ctx context.Context, id string) (account.Item, error) {
	if err := s.operations.Lock(ctx); err != nil {
		return account.Item{}, failure("operation_cancelled")
	}
	defer s.operations.Unlock()
	if s.accountBusy(id) {
		return account.Item{}, failure("busy")
	}
	item, profile, err := s.accounts.SubscriptionProfile(ctx, id)
	if err != nil {
		return account.Item{}, failure("invalid_account")
	}
	_ = profile
	_, profiles, err := s.accounts.SubscriptionProfiles(ctx, id)
	if err != nil {
		return item, failure("invalid_account")
	}
	for _, profile := range profiles {
		if err := s.logoutProfile(ctx, profile); err != nil {
			return item, err
		}
	}
	return s.accounts.UpdateSubscription(ctx, id, account.SubscriptionMetadata{Email: item.AuthEmail, Plan: item.AuthPlan, State: "logged_out", AccountID: item.AuthAccountID})
}

// Delete 仅清理本地凭据和账号，不启动官方组件或执行官方退出登录。
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.operations.Lock(ctx); err != nil {
		return failure("operation_cancelled")
	}
	defer s.operations.Unlock()
	if s.accountBusy(id) {
		return failure("busy")
	}
	_, profiles, err := s.accounts.SubscriptionProfiles(ctx, id)
	if err != nil {
		return failure("invalid_account")
	}
	for _, profile := range profiles {
		path, err := s.profilePath(profile)
		if err != nil {
			return err
		}
		// 本地清理失败时保留账号和定位信息，以便重试，不能遗失系统凭据引用。
		if err := s.removeCredentials(path); err != nil {
			return failure("cleanup_failed")
		}
	}
	if err := s.accounts.DeleteManaged(ctx, id); err != nil {
		return failure("account_delete_failed")
	}
	for _, profile := range profiles {
		path, err := s.profilePath(profile)
		if err != nil {
			return err
		}
		if err := os.RemoveAll(path); err != nil {
			return failure("profile_cleanup_failed")
		}
	}
	return nil
}

func (s *Service) accountBusy(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed || s.current != nil && s.current.input.AccountID == id
}
