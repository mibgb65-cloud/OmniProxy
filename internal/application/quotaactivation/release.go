package quotaactivation

import (
	"context"
	"omniproxy/internal/application/account"
	"time"
)

func (s *Service) release(ctx context.Context, item account.Item, claims []windowClaim) error {
	// 仅清理已确定没有发送的本次登记；允许取消后的本地清理，但不继续网络请求。
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()
	tx, err := s.db.BeginTx(cleanup, nil)
	if err != nil {
		return failure("record_failed")
	}
	defer tx.Rollback()
	for _, claim := range claims {
		if _, err := tx.ExecContext(cleanup, `UPDATE codex_quota_activation_windows SET attempted_cycle='',attempted_at='' WHERE account_id=? AND window_kind=? AND identity_version=? AND attempted_cycle=?`, item.ID, claim.kind, identityVersion(item), claim.cycle); err != nil {
			return failure("record_failed")
		}
	}
	if err := tx.Commit(); err != nil {
		return failure("record_failed")
	}
	return nil
}
