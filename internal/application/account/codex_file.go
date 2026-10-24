package account

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"omniproxy/internal/infrastructure/codex"
)

// AuthCodexFile 表示凭据是网关自己保管的 Codex auth.json（v1 的池账号形态）：
// 访问令牌与续期令牌都在本应用的受保护存储里，网关自己续期、自己回写。
const AuthCodexFile = "codex_auth_json"

// ImportCodexAuth 把一份 auth.json 收进网关自己的保护存储。
// 同一官方账号重复导入只更新凭据与元数据，不生成重复账号。
func (s *Service) ImportCodexAuth(ctx context.Context, providerID, name string, raw []byte) (Item, error) {
	if err := s.CheckSubscriptionProvider(ctx, providerID); err != nil {
		return Item{}, err
	}
	auth, err := codex.ParseAuthFile(raw)
	if err != nil {
		return Item{}, ErrInvalidInput
	}
	body, err := codex.ClientAuthFileFor(raw, auth.AccountID)
	if err != nil {
		return Item{}, ErrInvalidInput
	}
	defer clear(body)
	meta := SubscriptionMetadata{AccountID: auth.AccountID, Email: auth.Email, Plan: auth.Plan, State: "authenticated"}
	if validateSubscription(meta) != nil {
		return Item{}, ErrInvalidInput
	}
	existingID := ""
	err = s.db.QueryRowContext(ctx, `SELECT id FROM accounts WHERE provider_id=? AND auth_method=? AND auth_account_id=?`,
		providerID, AuthCodexFile, auth.AccountID).Scan(&existingID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return Item{}, fmt.Errorf("account: find codex auth")
	}
	if existingID != "" {
		return s.replaceImportedCodexAuth(ctx, existingID, body, meta)
	}
	return s.createCodexAuth(ctx, providerID, name, body, meta)
}

// UseCodexAuth 只在受控回调内交出 auth.json 副本，不带版本约束。
func (s *Service) UseCodexAuth(ctx context.Context, id string, consume func([]byte) error) error {
	return s.UseCodexAuthAt(ctx, id, "", consume)
}

// UseCodexAuthAt 只在受控回调内交出 auth.json 副本，回调结束即清理。
func (s *Service) UseCodexAuthAt(ctx context.Context, id, expectedAt string, consume func([]byte) error) error {
	if err := validateID(id); err != nil {
		return err
	}
	if consume == nil {
		return fmt.Errorf("%w: auth consumer", ErrInvalidInput)
	}
	item, ref, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	if expectedAt != "" && item.UpdatedAt != expectedAt {
		return ErrConflict
	}
	if item.AuthMethod != AuthCodexFile || item.AuthState != "authenticated" || !item.Enabled {
		return ErrNotFound
	}
	if ref == "" {
		return ErrNotFound
	}
	data, err := s.store.Get(ref)
	if err != nil {
		return fmt.Errorf("account: read codex auth: %w", err)
	}
	defer clear(data)
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := consume(data); err != nil {
		return err
	}
	current, _, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	if !current.Enabled || current.AuthState != "authenticated" || current.UpdatedAt != item.UpdatedAt {
		return ErrConflict
	}
	return ctx.Err()
}

// ReplaceCodexAuthTokens 只换令牌不动身份与版本：续期不是更换登录身份，
// 因此不能让进行中的请求因为版本变化被判为路由配置已变。
func (s *Service) ReplaceCodexAuthTokens(ctx context.Context, id string, raw []byte) error {
	item, ref, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	previous, err := s.store.Get(ref)
	if err != nil {
		return err
	}
	defer clear(previous)
	return s.ReplaceCodexAuthTokensAt(ctx, id, item.UpdatedAt, previous, raw)
}

// ReplaceCodexAuthTokensAt 用版本、成员身份及原凭据引用共同保护续期回写。
// 网络续期期间重新导入／注销或另一次刷新成功时，旧结果绝不能覆盖新授权。
func (s *Service) ReplaceCodexAuthTokensAt(ctx context.Context, id, expectedAt string, previous, raw []byte) error {
	if err := validateID(id); err != nil {
		return err
	}
	auth, err := codex.ParseAuthFile(raw)
	if err != nil {
		return ErrInvalidInput
	}
	item, oldRef, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	if item.AuthMethod != AuthCodexFile || !item.Enabled || item.AuthState != "authenticated" ||
		item.UpdatedAt != expectedAt || auth.AccountID != item.AuthAccountID ||
		!strings.EqualFold(strings.TrimSpace(auth.Email), strings.TrimSpace(item.AuthEmail)) {
		return ErrConflict
	}
	stored, err := s.store.Get(oldRef)
	if err != nil {
		return err
	}
	same := sha256.Sum256(stored) == sha256.Sum256(previous)
	clear(stored)
	if !same {
		return ErrConflict
	}
	ref, err := s.store.Put(raw)
	if err != nil {
		return fmt.Errorf("account: protect refreshed auth")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE accounts SET credential_ref=? WHERE id=? AND credential_ref=? AND updated_at=? AND enabled=1 AND auth_state='authenticated' AND auth_method=?`,
		ref, id, oldRef, expectedAt, AuthCodexFile)
	if err != nil {
		return errors.Join(fmt.Errorf("account: save refreshed auth"), s.store.Delete(ref))
	}
	if count, err := result.RowsAffected(); err != nil || count != 1 {
		return errors.Join(ErrConflict, s.store.Delete(ref))
	}
	if oldRef != "" && oldRef != ref {
		_ = s.store.Delete(oldRef)
	}
	return nil
}

func (s *Service) createCodexAuth(ctx context.Context, providerID, name string, body []byte, meta SubscriptionMetadata) (Item, error) {
	name, err := subscriptionName(name, meta.Email)
	if err != nil {
		return Item{}, err
	}
	id, err := newID()
	if err != nil {
		return Item{}, err
	}
	ref, err := s.store.Put(body)
	if err != nil {
		return Item{}, fmt.Errorf("account: protect codex auth")
	}
	timestamp := now()
	item := Item{
		ID: id, ProviderID: providerID, Name: name, Enabled: true, CreatedAt: timestamp, UpdatedAt: timestamp,
		AuthMethod: AuthCodexFile, AuthEmail: meta.Email, AuthPlan: meta.Plan, AuthState: meta.State, AuthAccountID: meta.AccountID,
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_email,auth_plan,auth_state,auth_account_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`,
		id, providerID, name, ref, true, timestamp, timestamp, AuthCodexFile, meta.Email, meta.Plan, meta.State, meta.AccountID)
	if err != nil {
		return Item{}, errors.Join(fmt.Errorf("account: create codex auth"), s.store.Delete(ref))
	}
	return item, nil
}

// replaceImportedCodexAuth 重新导入同一官方账号时更新凭据与元数据。
func (s *Service) replaceImportedCodexAuth(ctx context.Context, id string, body []byte, meta SubscriptionMetadata) (Item, error) {
	item, oldRef, err := s.get(ctx, id)
	if err != nil {
		return Item{}, err
	}
	ref, err := s.store.Put(body)
	if err != nil {
		return Item{}, fmt.Errorf("account: protect codex auth")
	}
	timestamp := now()
	_, err = s.db.ExecContext(ctx, `UPDATE accounts SET credential_ref=?,auth_email=?,auth_plan=?,auth_state=?,updated_at=? WHERE id=?`,
		ref, meta.Email, meta.Plan, meta.State, timestamp, id)
	if err != nil {
		return Item{}, errors.Join(fmt.Errorf("account: update codex auth"), s.store.Delete(ref))
	}
	if oldRef != "" && oldRef != ref {
		_ = s.store.Delete(oldRef)
	}
	item.AuthEmail, item.AuthPlan, item.AuthState, item.UpdatedAt = meta.Email, meta.Plan, meta.State, timestamp
	return item, nil
}
