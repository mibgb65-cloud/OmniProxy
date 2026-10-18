package account

import (
	"context"
	"fmt"
	"strings"
)

// UpdateSubscription 更新公开身份和会话状态，保持已有名称与凭证引用。
func (s *Service) UpdateSubscription(ctx context.Context, id string, meta SubscriptionMetadata) (Item, error) {
	if err := validateID(id); err != nil {
		return Item{}, err
	}
	if err := validateSubscription(meta); err != nil {
		return Item{}, err
	}
	item, _, err := s.get(ctx, id)
	if err != nil {
		return Item{}, err
	}
	if item.AuthMethod != AuthSubscription {
		return Item{}, ErrNotFound
	}
	timestamp, err := nowAfter(item.UpdatedAt)
	if err != nil {
		return Item{}, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE accounts SET auth_email=?,auth_plan=?,auth_state=?,auth_account_id=?,updated_at=? WHERE id=? AND auth_method=? AND updated_at=?`, meta.Email, meta.Plan, meta.State, meta.AccountID, timestamp, id, AuthSubscription, item.UpdatedAt)
	if err != nil {
		return Item{}, fmt.Errorf("account: update subscription")
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return Item{}, ErrConflict
	}
	item, _, err = s.get(ctx, id)
	return item, err
}

func (s *Service) updateSubscriptionName(ctx context.Context, item Item, input UpdateInput) (UpdateResult, error) {
	if strings.TrimSpace(input.APIKey) != "" {
		return UpdateResult{}, ErrInvalidInput
	}
	name, err := subscriptionName(input.Name, item.AuthEmail)
	if err != nil {
		return UpdateResult{}, err
	}
	timestamp, err := nowAfter(item.UpdatedAt)
	if err != nil {
		return UpdateResult{}, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE accounts SET name=?,updated_at=? WHERE id=? AND updated_at=? AND auth_method=?`, name, timestamp, item.ID, item.UpdatedAt, AuthSubscription)
	if err != nil {
		return UpdateResult{}, fmt.Errorf("account: subscription name update")
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return UpdateResult{}, ErrConflict
	}
	item.Name, item.UpdatedAt = name, timestamp
	return UpdateResult{Account: item}, nil
}
