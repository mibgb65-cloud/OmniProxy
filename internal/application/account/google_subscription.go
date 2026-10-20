package account

import (
	"context"
	"encoding/json"
	"errors"

	"omniproxy/internal/provider/antigravity"
)

// CatalogVersion 返回不含凭证的授权版本，防止旧目录覆盖重新登录后的账号。
func (s *Service) CatalogVersion(ctx context.Context, id string) (string, error) {
	if validateID(id) != nil {
		return "", ErrInvalidInput
	}
	var version string
	if s.db.QueryRowContext(ctx, "SELECT version FROM account_catalog_versions WHERE id=?", id).Scan(&version) != nil {
		return "", ErrNotFound
	}
	return version, nil
}

// CheckGoogleProvider 限定订阅账号归属，避免与 Codex 或普通 API 凭据混用。
func (s *Service) CheckGoogleProvider(ctx context.Context, providerID string) error {
	if validateID(providerID) != nil {
		return ErrInvalidInput
	}
	var kind string
	if s.db.QueryRowContext(ctx, "SELECT kind FROM providers WHERE id=?", providerID).Scan(&kind) != nil || kind != "google" {
		return ErrInvalidInput
	}
	return nil
}

// UseGoogleCredentials 仅在后端回调内短暂解密当前账号凭证。
func (s *Service) UseGoogleCredentials(ctx context.Context, id string, consume func(Item, antigravity.Credentials) error) error {
	if validateID(id) != nil || consume == nil {
		return antigravity.Failure("invalid_input")
	}
	item, ref, err := s.get(ctx, id)
	if err != nil || item.AuthMethod != AuthSubscription || item.AuthState != "authenticated" || s.CheckGoogleProvider(ctx, item.ProviderID) != nil {
		return antigravity.Failure("reauth_required")
	}
	data, err := s.store.Get(ref)
	if err != nil {
		return antigravity.Failure("credential_unavailable")
	}
	defer clear(data)
	var cred antigravity.Credentials
	if len(data) > 65536 || json.Unmarshal(data, &cred) != nil || !cred.Valid() || cred.Subject != item.AuthAccountID {
		return antigravity.Failure("credential_unavailable")
	}
	return consume(item, cred)
}

// SaveGoogleSubscription 在新授权确认成功后写入独立受保护凭证，重新登录必须保持身份。
func (s *Service) SaveGoogleSubscription(ctx context.Context, providerID, id, name string, cred antigravity.Credentials) (Item, error) {
	if s.CheckGoogleProvider(ctx, providerID) != nil || !cred.Valid() {
		return Item{}, ErrInvalidInput
	}
	if id != "" {
		item, err := s.Metadata(ctx, id)
		if err != nil || item.ProviderID != providerID || item.AuthMethod != AuthSubscription || item.AuthAccountID != cred.Subject {
			return Item{}, antigravity.Failure("identity_changed")
		}
		return s.writeGoogleCredentials(ctx, item, cred, false)
	}
	name, err := subscriptionName(name, cred.Email)
	if err != nil {
		return Item{}, err
	}
	id, err = newID()
	if err != nil {
		return Item{}, err
	}
	data, err := json.Marshal(cred)
	if err != nil {
		return Item{}, ErrInvalidInput
	}
	defer clear(data)
	ref, err := s.store.Put(data)
	if err != nil {
		return Item{}, antigravity.Failure("credential_unavailable")
	}
	stamp := now()
	_, err = s.db.ExecContext(ctx, `INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_email,auth_plan,auth_state,auth_account_id) VALUES(?,?,?,?,1,?,?,?,?,?,'authenticated',?)`, id, providerID, name, ref, stamp, stamp, AuthSubscription, cred.Email, cred.Plan, cred.Subject)
	if err != nil {
		return Item{}, errors.Join(ErrInvalidInput, s.store.Delete(ref))
	}
	return s.Metadata(ctx, id)
}

// RotateGoogleCredentials 令牌轮换不改变身份与模型权限，原目录在同一事务内重新绑定。
func (s *Service) RotateGoogleCredentials(ctx context.Context, expected Item, cred antigravity.Credentials) (Item, error) {
	if !cred.Valid() || cred.Subject != expected.AuthAccountID || s.CheckGoogleProvider(ctx, expected.ProviderID) != nil {
		return Item{}, ErrInvalidInput
	}
	return s.writeGoogleCredentials(ctx, expected, cred, true)
}

func (s *Service) writeGoogleCredentials(ctx context.Context, item Item, cred antigravity.Credentials, preserve bool) (Item, error) {
	current, oldRef, err := s.get(ctx, item.ID)
	if err != nil || current.UpdatedAt != item.UpdatedAt || current.AuthAccountID != cred.Subject {
		return Item{}, ErrConflict
	}
	stamp, err := nowAfter(item.UpdatedAt)
	if err != nil {
		return Item{}, err
	}
	retired, err := s.googleRetired(oldRef)
	if err != nil {
		return Item{}, err
	}
	if err = s.cleanupGoogleReferences(retired); err != nil {
		return Item{}, err
	}
	bundle := protectedGoogleCredential{Credentials: cred}
	if oldRef != "" {
		bundle.Retired = []string{oldRef}
	}
	// 先把待清理位置写进新密文，提交后删除失败仍可在刷新／退出时恢复清理。
	data, err := json.Marshal(bundle)
	if err != nil {
		return Item{}, ErrInvalidInput
	}
	defer clear(data)
	ref, err := s.store.Put(data)
	if err != nil {
		return Item{}, antigravity.Failure("credential_unavailable")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Item{}, errors.Join(err, s.store.Delete(ref))
	}
	defer tx.Rollback()
	var oldVersion, newVersion string
	err = tx.QueryRowContext(ctx, "SELECT version FROM account_catalog_versions WHERE id=?", item.ID).Scan(&oldVersion)
	if err == nil {
		result, e := tx.ExecContext(ctx, `UPDATE accounts SET credential_ref=?,auth_email=?,auth_plan=?,auth_state='authenticated',updated_at=? WHERE id=? AND credential_ref=? AND updated_at=? AND auth_method=?`, ref, cred.Email, cred.Plan, stamp, item.ID, oldRef, item.UpdatedAt, AuthSubscription)
		err = e
		if e == nil {
			count, e := result.RowsAffected()
			if e != nil || count != 1 {
				err = ErrConflict
			}
		}
	}
	if err == nil && preserve {
		err = tx.QueryRowContext(ctx, "SELECT version FROM account_catalog_versions WHERE id=?", item.ID).Scan(&newVersion)
		if err == nil {
			_, err = tx.ExecContext(ctx, "UPDATE account_models SET account_version=? WHERE account_id=? AND account_version=?", newVersion, item.ID, oldVersion)
		}
		if err == nil {
			// 普通令牌轮换保留同身份显示快照；重新登录不重绑定旧授权快照。
			_, err = tx.ExecContext(ctx, "UPDATE google_subscription_quota_snapshots SET account_version=? WHERE account_id=? AND account_version=?", newVersion, item.ID, oldVersion)
		}
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		return Item{}, errors.Join(ErrConflict, s.store.Delete(ref))
	}
	item.AuthEmail, item.AuthPlan, item.AuthState, item.UpdatedAt = cred.Email, cred.Plan, "authenticated", stamp
	if oldRef != "" {
		if err = s.store.Delete(oldRef); err != nil {
			return item, antigravity.Failure("cleanup_failed")
		}
	}
	return item, nil
}
