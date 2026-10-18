package account

import (
	"context"
	"omniproxy/internal/provider/antigravity"
)

// LogoutGoogleSubscription 先停用授权，再清理密文；清理失败保留引用以便重试。
func (s *Service) LogoutGoogleSubscription(ctx context.Context, id string) (Item, error) {
	if validateID(id) != nil {
		return Item{}, ErrInvalidInput
	}
	item, ref, err := s.get(ctx, id)
	if err != nil || item.AuthMethod != AuthSubscription || s.CheckGoogleProvider(ctx, item.ProviderID) != nil {
		return Item{}, ErrInvalidInput
	}
	stamp, err := nowAfter(item.UpdatedAt)
	if err != nil {
		return Item{}, err
	}
	clearedAt, err := nowAfter(stamp)
	if err != nil {
		return Item{}, err
	}
	result, err := s.db.ExecContext(ctx, "UPDATE accounts SET auth_state='logged_out',updated_at=? WHERE id=? AND credential_ref=? AND updated_at=?", stamp, id, ref, item.UpdatedAt)
	if err != nil {
		return Item{}, antigravity.Failure("operation_failed")
	}
	count, err := result.RowsAffected()
	if err != nil {
		return Item{}, antigravity.Failure("operation_failed")
	}
	if count != 1 {
		return Item{}, ErrConflict
	}
	if ref != "" {
		retired, err := s.googleRetired(ref)
		if err != nil {
			return Item{}, err
		}
		if err = s.cleanupGoogleReferences(append(retired, ref)); err != nil {
			return Item{}, err
		}
		result, err = s.db.ExecContext(ctx, "UPDATE accounts SET credential_ref='',updated_at=? WHERE id=? AND credential_ref=? AND updated_at=? AND auth_state='logged_out'", clearedAt, id, ref, stamp)
		if err != nil {
			return Item{}, antigravity.Failure("cleanup_failed")
		}
		count, err = result.RowsAffected()
		if err != nil {
			return Item{}, antigravity.Failure("cleanup_failed")
		}
		if count != 1 {
			return Item{}, ErrConflict
		}
	}
	return s.Metadata(ctx, id)
}
