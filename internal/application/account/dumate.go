package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"omniproxy/internal/provider/dumate"
)

// AuthDuMateCookie 是百度搭子个人版的登录态账号类型；Cookie 不能当成 API Key。
const AuthDuMateCookie = dumate.AuthMethod

// CheckDuMateProvider 限定 DuMate 账号只能挂在 dumate 提供商下。
func (s *Service) CheckDuMateProvider(ctx context.Context, providerID string) error {
	if validateID(providerID) != nil {
		return ErrInvalidInput
	}
	var kind string
	if s.db.QueryRowContext(ctx, "SELECT kind FROM providers WHERE id=?", providerID).Scan(&kind) != nil || kind != dumate.Kind {
		return ErrInvalidInput
	}
	return nil
}

// FindDuMateAccount 按远端身份查找同一账号；桌面端切换账号后不能静默覆盖原账号。
func (s *Service) FindDuMateAccount(ctx context.Context, providerID, subject string) (Item, bool, error) {
	if validateID(providerID) != nil || strings.TrimSpace(subject) == "" || len(subject) > 128 {
		return Item{}, false, ErrInvalidInput
	}
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM accounts WHERE provider_id=? AND auth_method=? AND auth_account_id=? ORDER BY created_at LIMIT 1`, providerID, AuthDuMateCookie, subject).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, false, nil
	}
	if err != nil {
		return Item{}, false, fmt.Errorf("account: find dumate account: %w", err)
	}
	item, err := s.Metadata(ctx, id)
	return item, err == nil, err
}

// SaveDuMateAccount 保存已核对过远端身份的 Cookie 登录态。
// id 为空时按身份复用或新建；id 非空时只允许同一身份的重新导入。
func (s *Service) SaveDuMateAccount(ctx context.Context, providerID, id, name string, credentials dumate.Credentials) (Item, error) {
	if s.CheckDuMateProvider(ctx, providerID) != nil || !credentials.Valid() || strings.TrimSpace(credentials.AccountID) == "" {
		return Item{}, ErrInvalidInput
	}
	if id != "" {
		item, err := s.Metadata(ctx, id)
		if err != nil {
			return Item{}, err
		}
		if item.ProviderID != providerID || item.AuthMethod != AuthDuMateCookie || item.AuthAccountID != credentials.AccountID {
			return Item{}, dumate.Failure("identity_changed")
		}
		return s.writeDuMateCredentials(ctx, item, credentials)
	}
	existing, found, err := s.FindDuMateAccount(ctx, providerID, credentials.AccountID)
	if err != nil {
		return Item{}, err
	}
	if found {
		return s.writeDuMateCredentials(ctx, existing, credentials)
	}
	return s.createDuMateAccount(ctx, providerID, name, credentials)
}

func (s *Service) createDuMateAccount(ctx context.Context, providerID, name string, credentials dumate.Credentials) (Item, error) {
	name, err := dumateAccountName(name, credentials.Name)
	if err != nil {
		return Item{}, err
	}
	id, err := newID()
	if err != nil {
		return Item{}, fmt.Errorf("account: generate ID: %w", err)
	}
	data, err := credentials.Marshal()
	if err != nil {
		return Item{}, err
	}
	defer clear(data)
	ref, err := s.store.Put(data)
	if err != nil {
		return Item{}, dumate.Failure(dumate.CodeCredentialUnavailable)
	}
	stamp := now()
	_, err = s.db.ExecContext(ctx, `INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_state,auth_account_id) VALUES(?,?,?,?,1,?,?,?,?,?)`,
		id, providerID, name, ref, stamp, stamp, AuthDuMateCookie, "authenticated", credentials.AccountID)
	if err != nil {
		return Item{}, errors.Join(ErrInvalidInput, s.store.Delete(ref))
	}
	return s.Metadata(ctx, id)
}

// writeDuMateCredentials 重新导入同一身份时替换密文；账号名称与目录保留。
func (s *Service) writeDuMateCredentials(ctx context.Context, item Item, credentials dumate.Credentials) (Item, error) {
	current, oldRef, err := s.get(ctx, item.ID)
	if err != nil || current.UpdatedAt != item.UpdatedAt || current.AuthAccountID != credentials.AccountID {
		return Item{}, ErrConflict
	}
	stamp, err := nowAfter(item.UpdatedAt)
	if err != nil {
		return Item{}, err
	}
	data, err := credentials.Marshal()
	if err != nil {
		return Item{}, err
	}
	defer clear(data)
	ref, err := s.store.Put(data)
	if err != nil {
		return Item{}, dumate.Failure(dumate.CodeCredentialUnavailable)
	}
	result, err := s.db.ExecContext(ctx, `UPDATE accounts SET credential_ref=?,auth_state='authenticated',updated_at=? WHERE id=? AND credential_ref=? AND updated_at=? AND auth_method=?`,
		ref, stamp, item.ID, oldRef, item.UpdatedAt, AuthDuMateCookie)
	if err == nil {
		var changed int64
		if changed, err = result.RowsAffected(); err == nil && changed != 1 {
			err = ErrConflict
		}
	}
	if err != nil {
		return Item{}, errors.Join(ErrConflict, s.store.Delete(ref))
	}
	item.AuthState, item.UpdatedAt = "authenticated", stamp
	if oldRef != "" {
		if err = s.store.Delete(oldRef); err != nil {
			return item, dumate.Failure("cleanup_failed")
		}
	}
	return item, nil
}

// UseDuMateCredentials 只在校验账号归属与身份后，于回调内短暂解密 Cookie。
func (s *Service) UseDuMateCredentials(ctx context.Context, id string, consume func(Item, dumate.Credentials) error) error {
	if validateID(id) != nil || consume == nil {
		return dumate.Failure("invalid_input")
	}
	item, ref, err := s.get(ctx, id)
	if err != nil || item.AuthMethod != AuthDuMateCookie || item.AuthState != "authenticated" || s.CheckDuMateProvider(ctx, item.ProviderID) != nil {
		return dumate.Failure(dumate.CodeCredentialUnavailable)
	}
	if ref == "" {
		return dumate.Failure(dumate.CodeCredentialUnavailable)
	}
	data, err := s.store.Get(ref)
	if err != nil {
		return dumate.Failure(dumate.CodeCredentialUnavailable)
	}
	defer clear(data)
	credentials, err := dumate.ParseCredentials(data)
	if err != nil || credentials.AccountID != item.AuthAccountID {
		return dumate.Failure(dumate.CodeCredentialUnavailable)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return consume(item, credentials)
}

// dumateAccountName 只接受用户提供的名称或远端展示名，拒绝控制字符。
func dumateAccountName(name, fallback string) (string, error) {
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSpace(fallback)
	}
	if name == "" {
		name = "百度搭子 DuMate"
	}
	return validateName(name)
}

// updateDuMateName 只改名称，不触碰密文与目录。
func (s *Service) updateDuMateName(ctx context.Context, item Item, name string) (Item, error) {
	if strings.TrimSpace(name) == "" {
		name = item.Name
	}
	return s.writeDuMateName(ctx, item, name)
}

func (s *Service) writeDuMateName(ctx context.Context, item Item, name string) (Item, error) {
	name, err := validateName(name)
	if err != nil {
		return Item{}, err
	}
	stamp, err := nowAfter(item.UpdatedAt)
	if err != nil {
		return Item{}, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE accounts SET name=?,updated_at=? WHERE id=? AND updated_at=? AND auth_method=?`, name, stamp, item.ID, item.UpdatedAt, AuthDuMateCookie)
	if err != nil {
		return Item{}, fmt.Errorf("account: rename dumate: %w", err)
	}
	if changed, err := result.RowsAffected(); err != nil || changed != 1 {
		return Item{}, ErrConflict
	}
	item.Name, item.UpdatedAt = name, stamp
	return item, nil
}
