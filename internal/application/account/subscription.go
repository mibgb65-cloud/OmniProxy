package account

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const AuthAPIKey = "api_key"
const AuthSubscription = "subscription_login"

// SubscriptionMetadata 不包含授权令牌。
type SubscriptionMetadata struct {
	AccountID string
	Email     string
	Plan      string
	State     string
}
type managedLocator struct {
	Kind    string   `json:"kind"`
	Profile string   `json:"profile"`
	Retired []string `json:"retired,omitempty"`
}

// CheckSubscriptionProvider 限定到 OpenAI 配置，客户端不能为其他提供商保存订阅账号。
func (s *Service) CheckSubscriptionProvider(ctx context.Context, id string) error {
	if err := validateID(id); err != nil {
		return err
	}
	var kind string
	if err := s.db.QueryRowContext(ctx, "SELECT kind FROM providers WHERE id = ?", id).Scan(&kind); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("account: subscription provider unavailable")
	}
	if kind != "openai" {
		return ErrInvalidInput
	}
	return nil
}

// Metadata 获取安全账号信息。
func (s *Service) Metadata(ctx context.Context, id string) (Item, error) {
	if err := validateID(id); err != nil {
		return Item{}, err
	}
	item, _, err := s.get(ctx, id)
	return item, err
}

func validProfile(id string) bool {
	if len(id) != 32 {
		return false
	}
	for _, r := range id {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}
func subscriptionName(name, email string) (string, error) {
	if strings.TrimSpace(name) == "" {
		name = email
		if name == "" {
			name = "ChatGPT Codex"
		}
		runes := []rune(name)
		name = string(runes[:min(len(runes), maxNameRunes)])
	}
	return validateName(name)
}

// CreateSubscription 保存系统凭证库的隔离定位信息，令牌不进入 SQLite 或返回值。
func (s *Service) CreateSubscription(ctx context.Context, providerID, name, profile string, meta SubscriptionMetadata) (Item, error) {
	if err := s.CheckSubscriptionProvider(ctx, providerID); err != nil {
		return Item{}, err
	}
	if !validProfile(profile) || validateSubscription(meta) != nil {
		return Item{}, ErrInvalidInput
	}
	name, err := subscriptionName(name, meta.Email)
	if err != nil {
		return Item{}, err
	}
	id, err := newID()
	if err != nil {
		return Item{}, err
	}
	data, err := json.Marshal(managedLocator{Kind: "codex-keyring", Profile: profile})
	if err != nil {
		return Item{}, err
	}
	defer clear(data)
	ref, err := s.store.Put(data)
	if err != nil {
		return Item{}, fmt.Errorf("account: protect subscription reference")
	}
	timestamp := now()
	item := Item{ID: id, ProviderID: providerID, Name: name, Enabled: true, CreatedAt: timestamp, UpdatedAt: timestamp, AuthMethod: AuthSubscription, AuthEmail: meta.Email, AuthPlan: meta.Plan, AuthState: meta.State, AuthAccountID: meta.AccountID}
	_, err = s.db.ExecContext(ctx, `INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method,auth_email,auth_plan,auth_state,auth_account_id) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, id, providerID, name, ref, true, timestamp, timestamp, AuthSubscription, meta.Email, meta.Plan, meta.State, meta.AccountID)
	if err != nil {
		return Item{}, errors.Join(fmt.Errorf("account: create subscription"), s.store.Delete(ref))
	}
	return item, nil
}

// SubscriptionProfile 只返回隔离配置 ID，完整系统凭证由官方登录组件持有。
func (s *Service) SubscriptionProfile(ctx context.Context, id string) (Item, string, error) {
	if err := validateID(id); err != nil {
		return Item{}, "", err
	}
	item, ref, err := s.get(ctx, id)
	if err != nil {
		return item, "", err
	}
	if item.AuthMethod != AuthSubscription {
		return Item{}, "", ErrInvalidInput
	}
	data, err := s.store.Get(ref)
	if err != nil {
		return Item{}, "", fmt.Errorf("account: subscription reference unavailable")
	}
	defer clear(data)
	if len(data) > 1024 {
		return Item{}, "", ErrInvalidInput
	}
	var locator managedLocator
	if json.Unmarshal(data, &locator) != nil || locator.Kind != "codex-keyring" || !validProfile(locator.Profile) {
		return Item{}, "", ErrInvalidInput
	}
	return item, locator.Profile, nil
}

// UpdateSubscription 更新公开身份和会话状态，保持已有名称与凭证引用。
func (s *Service) UpdateSubscription(ctx context.Context, id string, meta SubscriptionMetadata) (Item, error) {
	if err := validateID(id); err != nil {
		return Item{}, err
	}
	if err := validateSubscription(meta); err != nil {
		return Item{}, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE accounts SET auth_email=?,auth_plan=?,auth_state=?,auth_account_id=?,updated_at=? WHERE id=? AND auth_method=?`, meta.Email, meta.Plan, meta.State, meta.AccountID, now(), id, AuthSubscription)
	if err != nil {
		return Item{}, fmt.Errorf("account: update subscription")
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return Item{}, ErrNotFound
	}
	item, _, err := s.get(ctx, id)
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
	timestamp := now()
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
