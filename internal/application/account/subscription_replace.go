package account

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// SubscriptionProfiles 保留尚待清理的旧系统凭证位置，防止重新登录后丢失清理引用。
func (s *Service) SubscriptionProfiles(ctx context.Context, id string) (Item, []string, error) {
	item, current, err := s.SubscriptionProfile(ctx, id)
	if err != nil {
		return item, nil, err
	}
	_, ref, err := s.get(ctx, id)
	if err != nil {
		return item, nil, err
	}
	data, err := s.store.Get(ref)
	if err != nil {
		return item, nil, ErrNotFound
	}
	defer clear(data)
	var locator managedLocator
	if json.Unmarshal(data, &locator) != nil || len(locator.Retired) > 16 {
		return item, nil, ErrInvalidInput
	}
	profiles := []string{current}
	seen := map[string]bool{current: true}
	for _, profile := range locator.Retired {
		if !validProfile(profile) || seen[profile] {
			return item, nil, ErrInvalidInput
		}
		seen[profile] = true
		profiles = append(profiles, profile)
	}
	return item, profiles, nil
}

// ReplaceSubscription 只有授权成功后才切换隔离配置，取消不会破坏原会话。
func (s *Service) ReplaceSubscription(ctx context.Context, id, profile string, meta SubscriptionMetadata) (Item, error) {
	item, profiles, err := s.SubscriptionProfiles(ctx, id)
	if err != nil {
		return Item{}, err
	}
	if !validProfile(profile) || validateSubscription(meta) != nil || len(profiles) > 16 {
		return Item{}, ErrInvalidInput
	}
	return s.writeLocator(ctx, item, managedLocator{Kind: "codex-keyring", Profile: profile, Retired: profiles}, meta)
}

// ClearRetired 仅在旧系统凭证全部清理成功后移除清理引用。
func (s *Service) ClearRetired(ctx context.Context, id string) (Item, error) {
	item, profile, err := s.SubscriptionProfile(ctx, id)
	if err != nil {
		return item, err
	}
	return s.writeLocator(ctx, item, managedLocator{Kind: "codex-keyring", Profile: profile}, SubscriptionMetadata{Email: item.AuthEmail, Plan: item.AuthPlan, State: item.AuthState, AccountID: item.AuthAccountID})
}

func (s *Service) writeLocator(ctx context.Context, item Item, locator managedLocator, meta SubscriptionMetadata) (Item, error) {
	_, oldRef, err := s.get(ctx, item.ID)
	if err != nil {
		return Item{}, err
	}
	data, err := json.Marshal(locator)
	if err != nil {
		return Item{}, ErrInvalidInput
	}
	defer clear(data)
	ref, err := s.store.Put(data)
	if err != nil {
		return Item{}, fmt.Errorf("account: protect subscription reference")
	}
	timestamp := now()
	result, err := s.db.ExecContext(ctx, `UPDATE accounts SET credential_ref=?,auth_email=?,auth_plan=?,auth_state=?,auth_account_id=?,updated_at=? WHERE id=? AND credential_ref=? AND updated_at=?`, ref, meta.Email, meta.Plan, meta.State, meta.AccountID, timestamp, item.ID, oldRef, item.UpdatedAt)
	if err == nil {
		changed, rowErr := result.RowsAffected()
		if rowErr != nil || changed != 1 {
			err = ErrConflict
		}
	}
	if err != nil {
		return Item{}, errors.Join(err, s.store.Delete(ref))
	}
	item.AuthEmail, item.AuthPlan, item.AuthState, item.UpdatedAt = meta.Email, meta.Plan, meta.State, timestamp
	item.AuthAccountID = meta.AccountID
	// 已切换成功；旧定位密文删除失败不影响新会话，系统令牌仍由 Retired 引用管理。
	if err := s.store.Delete(oldRef); err != nil {
		return item, fmt.Errorf("account: locator_cleanup_warning")
	}
	return item, nil
}
