package account

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"omniproxy/internal/infrastructure/credential"
	"omniproxy/internal/provider/workbuddyai"
)

// AuthWorkBuddyLogin 是 WorkBuddy AI 订阅账号的认证类型；登录令牌不能伪装成 API Key。
const AuthWorkBuddyLogin = workbuddyai.AuthMethod

// ErrWorkBuddySave 表示远端登录已成功，但本地保存失败（数据库或受保护存储）。
// 与「登录失败」区分：用户不需要重新登录，重试即可，因此单独给稳定码。
var ErrWorkBuddySave = errors.New("account: save workbuddy login")

// protectedWorkBuddyCredential 是受保护存储正文：登录令牌与待清理的旧引用。
type protectedWorkBuddyCredential struct {
	workbuddyai.Credentials
	Retired []string `json:"retired,omitempty"`
}

// CheckWorkBuddyProvider 限定 WorkBuddy AI 账号只能挂在 workbuddy_ai 提供商下。
func (s *Service) CheckWorkBuddyProvider(ctx context.Context, providerID string) error {
	if validateID(providerID) != nil {
		return ErrInvalidInput
	}
	var kind string
	if s.db.QueryRowContext(ctx, "SELECT kind FROM providers WHERE id=?", providerID).Scan(&kind) != nil || kind != workbuddyai.Kind {
		return ErrInvalidInput
	}
	return nil
}

// FindWorkBuddyAccount 按远端 uid 查找同一账号；重新登录同一身份只更新凭据。
func (s *Service) FindWorkBuddyAccount(ctx context.Context, providerID, subject string) (Item, bool, error) {
	if validateID(providerID) != nil || strings.TrimSpace(subject) == "" || len(subject) > 128 {
		return Item{}, false, ErrInvalidInput
	}
	var id string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM accounts WHERE provider_id=? AND auth_method=? AND auth_account_id=? ORDER BY created_at LIMIT 1`, providerID, AuthWorkBuddyLogin, subject).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, false, nil
	}
	if err != nil {
		return Item{}, false, errors.New("account: find workbuddy account")
	}
	item, err := s.Metadata(ctx, id)
	return item, err == nil, err
}

// SaveWorkBuddyAccount 保存已核对过远端身份的登录令牌。
// id 为空时按身份复用或新建；id 非空时只允许同一身份的重新登录。
func (s *Service) SaveWorkBuddyAccount(ctx context.Context, providerID, id, name string, cred workbuddyai.Credentials) (Item, error) {
	if s.CheckWorkBuddyProvider(ctx, providerID) != nil || !cred.Valid() || strings.TrimSpace(cred.UID) == "" {
		return Item{}, ErrInvalidInput
	}
	if id != "" {
		item, err := s.Metadata(ctx, id)
		if err != nil || item.ProviderID != providerID || item.AuthMethod != AuthWorkBuddyLogin || item.AuthAccountID != cred.UID {
			return Item{}, workbuddyai.Failure("identity_changed")
		}
		return s.writeWorkBuddyCredentials(ctx, item, cred, false)
	}
	existing, found, err := s.FindWorkBuddyAccount(ctx, providerID, cred.UID)
	if err != nil {
		return Item{}, err
	}
	if found {
		return s.writeWorkBuddyCredentials(ctx, existing, cred, false)
	}
	return s.createWorkBuddyAccount(ctx, providerID, name, cred)
}

func (s *Service) createWorkBuddyAccount(ctx context.Context, providerID, name string, cred workbuddyai.Credentials) (Item, error) {
	name, err := workbuddyAccountName(name, cred.AccountName)
	if err != nil {
		return Item{}, err
	}
	id, err := newID()
	if err != nil {
		return Item{}, errors.New("account: generate ID")
	}
	bundle := protectedWorkBuddyCredential{Credentials: cred}
	data, err := json.Marshal(bundle)
	if err != nil || len(data) > 1<<20 {
		return Item{}, workbuddyai.Failure(workbuddyai.CodeCredentialUnavailable)
	}
	defer clear(data)
	ref, err := s.store.Put(data)
	if err != nil {
		return Item{}, workbuddyai.Failure(workbuddyai.CodeCredentialUnavailable)
	}
	stamp := now()
	_, err = s.db.ExecContext(ctx, `INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_plan,auth_state,auth_account_id) VALUES(?,?,?,?,1,?,?,?,?,'authenticated',?)`,
		id, providerID, name, ref, stamp, stamp, AuthWorkBuddyLogin, cred.Plan, cred.UID)
	if err != nil {
		return Item{}, errors.Join(ErrWorkBuddySave, s.store.Delete(ref))
	}
	return s.Metadata(ctx, id)
}

// RotateWorkBuddyCredentials 令牌轮换不改变身份与模型权限，原目录在同一事务内重新绑定。
func (s *Service) RotateWorkBuddyCredentials(ctx context.Context, expected Item, cred workbuddyai.Credentials) (Item, error) {
	if !cred.Valid() || cred.UID != expected.AuthAccountID || s.CheckWorkBuddyProvider(ctx, expected.ProviderID) != nil {
		return Item{}, ErrInvalidInput
	}
	return s.writeWorkBuddyCredentials(ctx, expected, cred, true)
}

// writeWorkBuddyCredentials 写入新令牌；preserve 为真（普通轮换）时把目录与额度快照
// 重新绑定到新引用，否则（重新登录）留给后续同步重新建立目录。
func (s *Service) writeWorkBuddyCredentials(ctx context.Context, item Item, cred workbuddyai.Credentials, preserve bool) (Item, error) {
	current, oldRef, err := s.get(ctx, item.ID)
	if err != nil || current.UpdatedAt != item.UpdatedAt || current.AuthAccountID != cred.UID {
		return Item{}, ErrConflict
	}
	stamp, err := nowAfter(item.UpdatedAt)
	if err != nil {
		return Item{}, err
	}
	retired, err := s.workbuddyRetired(oldRef)
	if err != nil {
		return Item{}, err
	}
	if err = s.cleanupWorkBuddyReferences(retired); err != nil {
		return Item{}, err
	}
	bundle := protectedWorkBuddyCredential{Credentials: cred}
	if oldRef != "" {
		bundle.Retired = []string{oldRef}
	}
	data, err := json.Marshal(bundle)
	if err != nil || len(data) > 1<<20 {
		return Item{}, workbuddyai.Failure(workbuddyai.CodeCredentialUnavailable)
	}
	defer clear(data)
	ref, err := s.store.Put(data)
	if err != nil {
		return Item{}, workbuddyai.Failure(workbuddyai.CodeCredentialUnavailable)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Item{}, errors.Join(ErrWorkBuddySave, s.store.Delete(ref))
	}
	defer tx.Rollback()
	var oldVersion, newVersion string
	err = tx.QueryRowContext(ctx, "SELECT version FROM account_catalog_versions WHERE id=?", item.ID).Scan(&oldVersion)
	if err == nil {
		result, e := tx.ExecContext(ctx, `UPDATE accounts SET credential_ref=?,auth_plan=?,auth_state='authenticated',updated_at=? WHERE id=? AND credential_ref=? AND updated_at=? AND auth_method=?`,
			ref, cred.Plan, stamp, item.ID, oldRef, item.UpdatedAt, AuthWorkBuddyLogin)
		err = e
		if e == nil {
			count, e := result.RowsAffected()
			if e != nil || count != 1 {
				err = ErrConflict
			}
		}
	}
	if err == nil && preserve {
		// 续期只换令牌：目录必须跟着新引用走，否则模型会静默消失。
		err = tx.QueryRowContext(ctx, "SELECT version FROM account_catalog_versions WHERE id=?", item.ID).Scan(&newVersion)
		if err == nil {
			_, err = tx.ExecContext(ctx, "UPDATE account_models SET account_version=? WHERE account_id=? AND account_version=?", newVersion, item.ID, oldVersion)
		}
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		// 版本冲突（同一账号被并发改写）保留 ErrConflict 语义，交给界面提示刷新；
		// 其余数据库失败统一按保存失败处理。
		if errors.Is(err, ErrConflict) {
			return Item{}, errors.Join(ErrConflict, s.store.Delete(ref))
		}
		return Item{}, errors.Join(ErrWorkBuddySave, s.store.Delete(ref))
	}
	item.AuthPlan, item.AuthState, item.UpdatedAt = cred.Plan, "authenticated", stamp
	if oldRef != "" {
		if err = s.store.Delete(oldRef); err != nil {
			return item, workbuddyai.Failure("cleanup_failed")
		}
	}
	return item, nil
}

// UseWorkBuddyCredentials 只在校验账号归属与身份后，于回调内短暂解密登录令牌。
func (s *Service) UseWorkBuddyCredentials(ctx context.Context, id string, consume func(Item, workbuddyai.Credentials) error) error {
	if validateID(id) != nil || consume == nil {
		return workbuddyai.Failure("invalid_input")
	}
	item, ref, err := s.get(ctx, id)
	if err != nil || item.AuthMethod != AuthWorkBuddyLogin || item.AuthState != "authenticated" || s.CheckWorkBuddyProvider(ctx, item.ProviderID) != nil {
		return workbuddyai.Failure(workbuddyai.CodeCredentialUnavailable)
	}
	if ref == "" {
		return workbuddyai.Failure(workbuddyai.CodeCredentialUnavailable)
	}
	data, err := s.store.Get(ref)
	if err != nil {
		return workbuddyai.Failure(workbuddyai.CodeCredentialUnavailable)
	}
	defer clear(data)
	var bundle protectedWorkBuddyCredential
	if len(data) > 1<<20 || json.Unmarshal(data, &bundle) != nil || !bundle.Credentials.Valid() || bundle.Credentials.UID != item.AuthAccountID {
		return workbuddyai.Failure(workbuddyai.CodeCredentialUnavailable)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return consume(item, bundle.Credentials)
}

// LogoutWorkBuddySubscription 先停用授权，再清理密文；清理失败保留引用以便重试。
func (s *Service) LogoutWorkBuddySubscription(ctx context.Context, id string) (Item, error) {
	if validateID(id) != nil {
		return Item{}, ErrInvalidInput
	}
	item, ref, err := s.get(ctx, id)
	if err != nil || item.AuthMethod != AuthWorkBuddyLogin || s.CheckWorkBuddyProvider(ctx, item.ProviderID) != nil {
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
		return Item{}, workbuddyai.Failure("operation_failed")
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 {
		return Item{}, ErrConflict
	}
	if ref != "" {
		retired, err := s.workbuddyRetired(ref)
		if err != nil {
			return Item{}, err
		}
		if err = s.cleanupWorkBuddyReferences(append(retired, ref)); err != nil {
			return Item{}, err
		}
		result, err = s.db.ExecContext(ctx, "UPDATE accounts SET credential_ref='',updated_at=? WHERE id=? AND credential_ref=? AND updated_at=? AND auth_state='logged_out'", clearedAt, id, ref, stamp)
		if err != nil {
			return Item{}, workbuddyai.Failure("cleanup_failed")
		}
		count, err = result.RowsAffected()
		if err != nil || count != 1 {
			return Item{}, workbuddyai.Failure("cleanup_failed")
		}
	}
	return s.Metadata(ctx, id)
}

func (s *Service) workbuddyRetired(ref string) ([]string, error) {
	if ref == "" {
		return nil, nil
	}
	data, err := s.store.Get(ref)
	if errors.Is(err, credential.ErrNotFound) || errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, workbuddyai.Failure("cleanup_failed")
	}
	defer clear(data)
	var bundle protectedWorkBuddyCredential
	if len(data) > 1<<20 || json.Unmarshal(data, &bundle) != nil || len(bundle.Retired) > 16 {
		return nil, workbuddyai.Failure("cleanup_failed")
	}
	seen := map[string]bool{ref: true}
	for _, retired := range bundle.Retired {
		if validateID(retired) != nil || seen[retired] {
			return nil, workbuddyai.Failure("cleanup_failed")
		}
		seen[retired] = true
	}
	return bundle.Retired, nil
}

func (s *Service) cleanupWorkBuddyReferences(refs []string) error {
	for _, ref := range refs {
		if err := s.store.Delete(ref); err != nil && !errors.Is(err, credential.ErrNotFound) && !errors.Is(err, ErrNotFound) {
			return workbuddyai.Failure("cleanup_failed")
		}
	}
	return nil
}

// workbuddyAccountName 只接受用户提供的名称或远端展示名，拒绝控制字符。
func workbuddyAccountName(name, fallback string) (string, error) {
	if strings.TrimSpace(name) == "" {
		name = strings.TrimSpace(fallback)
	}
	if name == "" {
		name = "WorkBuddy AI"
	}
	return validateName(name)
}

// updateWorkBuddyName 只改名称，不触碰密文与目录。
func (s *Service) updateWorkBuddyName(ctx context.Context, item Item, name string) (Item, error) {
	if strings.TrimSpace(name) == "" {
		name = item.Name
	}
	name, err := validateName(name)
	if err != nil {
		return Item{}, err
	}
	stamp, err := nowAfter(item.UpdatedAt)
	if err != nil {
		return Item{}, err
	}
	result, err := s.db.ExecContext(ctx, "UPDATE accounts SET name=?,updated_at=? WHERE id=? AND updated_at=? AND auth_method=?", name, stamp, item.ID, item.UpdatedAt, AuthWorkBuddyLogin)
	if err != nil {
		return Item{}, errors.New("account: rename workbuddy")
	}
	if changed, err := result.RowsAffected(); err != nil || changed != 1 {
		return Item{}, ErrConflict
	}
	item.Name, item.UpdatedAt = name, stamp
	return item, nil
}
