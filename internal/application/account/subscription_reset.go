package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ResetAttempt 只保存公开标识与稳定结果，支持关闭窗口或重启后的幂等核对。
type ResetAttempt struct {
	OperationID   string
	AccountID     string
	AuthAccountID string
	CreditID      string
	State         string
	Outcome       string
}

func (s *Service) resetAttempt(ctx context.Context, query string, args ...any) (ResetAttempt, error) {
	var attempt ResetAttempt
	err := s.db.QueryRowContext(ctx, query, args...).Scan(&attempt.OperationID, &attempt.AccountID, &attempt.AuthAccountID, &attempt.CreditID, &attempt.State, &attempt.Outcome)
	if errors.Is(err, sql.ErrNoRows) {
		return attempt, nil
	}
	if err != nil {
		return ResetAttempt{}, fmt.Errorf("account: reset record unavailable")
	}
	return attempt, nil
}

func (s *Service) PendingSubscriptionReset(ctx context.Context, id string) (ResetAttempt, error) {
	if _, err := s.Metadata(ctx, id); err != nil {
		return ResetAttempt{}, err
	}
	return s.resetAttempt(ctx, `SELECT operation_id,account_id,auth_account_id,credit_id,state,outcome FROM subscription_reset_attempts WHERE account_id=? AND state='pending'`, id)
}

func (s *Service) SubscriptionResetAttempt(ctx context.Context, id, operationID string) (ResetAttempt, error) {
	return s.resetAttempt(ctx, `SELECT operation_id,account_id,auth_account_id,credit_id,state,outcome FROM subscription_reset_attempts WHERE account_id=? AND operation_id=?`, id, operationID)
}

// LatestSubscriptionReset 保留已完成的回执，避免 IPC 丢失后误以为没有执行。
func (s *Service) LatestSubscriptionReset(ctx context.Context, id string) (ResetAttempt, error) {
	return s.resetAttempt(ctx, `SELECT operation_id,account_id,auth_account_id,credit_id,state,outcome FROM subscription_reset_attempts WHERE account_id=? ORDER BY created_at DESC, operation_id DESC LIMIT 1`, id)
}

func (s *Service) SaveSubscriptionReset(ctx context.Context, attempt ResetAttempt) error {
	item, err := s.Metadata(ctx, attempt.AccountID)
	if err != nil || item.AuthMethod != AuthSubscription {
		return ErrInvalidInput
	}
	if err := validateSubscription(SubscriptionMetadata{AccountID: attempt.AuthAccountID, State: "authenticated"}); err != nil || attempt.AuthAccountID == "" {
		return ErrInvalidInput
	}
	timestamp := now()
	_, err = s.db.ExecContext(ctx, `INSERT INTO subscription_reset_attempts(operation_id,account_id,auth_account_id,credit_id,state,created_at,updated_at) VALUES(?,?,?,?,'pending',?,?)`, attempt.OperationID, attempt.AccountID, attempt.AuthAccountID, attempt.CreditID, timestamp, timestamp)
	if err != nil {
		return fmt.Errorf("account: reset record save failed")
	}
	return nil
}

func (s *Service) CompleteSubscriptionReset(ctx context.Context, id, key, outcome string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE subscription_reset_attempts SET state='completed',outcome=?,updated_at=? WHERE account_id=? AND operation_id=?`, outcome, now(), id, key)
	if err != nil {
		return fmt.Errorf("account: reset record update failed")
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return ErrNotFound
	}
	return nil
}
