// Package account manages account metadata and protected API Key credentials.
package account

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"omniproxy/internal/provider/stepfun"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	maxNameRunes     = 80
	maxNameBytes     = 256
	maxKeyBytes      = 8192
	maxIDBytes       = 128
	maxBatchKeys     = 100
	defaultNameRunes = 8
)

var (
	// ErrInvalidInput means an account field failed validation.
	ErrInvalidInput = errors.New("account: invalid input")
	// ErrNotFound means the account or provider does not exist.
	ErrNotFound = errors.New("account: not found")
)

// Item 返回账号元数据；自动名称含用户要求的 Key 前缀，不应写入日志。
type Item struct {
	StepFun       *stepfun.Options `json:"stepfun,omitempty"`
	ID            string           `json:"id"`
	ProviderID    string           `json:"providerId"`
	Name          string           `json:"name"`
	Enabled       bool             `json:"enabled"`
	CreatedAt     string           `json:"createdAt"`
	UpdatedAt     string           `json:"updatedAt"`
	AuthMethod    string           `json:"authMethod,omitempty"`
	AuthEmail     string           `json:"authEmail,omitempty"`
	AuthPlan      string           `json:"authPlan,omitempty"`
	AuthState     string           `json:"authState,omitempty"`
	AuthAccountID string           `json:"authAccountId,omitempty"`
}

// CreateInput contains fields for a new API Key account. APIKey is never persisted in SQLite.
type CreateInput struct {
	StepFun    *stepfun.Options `json:"stepfun,omitempty"`
	ProviderID string           `json:"providerId"`
	Name       string           `json:"name"`
	APIKey     string           `json:"apiKey"`
}

// BatchCreateInput 批量创建自动命名的账号，不接受自定义名称。
type BatchCreateInput struct {
	StepFun    *stepfun.Options `json:"stepfun,omitempty"`
	ProviderID string           `json:"providerId"`
	APIKeys    []string         `json:"apiKeys"`
}

// CredentialStore is the protected storage needed by account mutations.
type CredentialStore interface {
	Put([]byte) (string, error)
	Get(string) ([]byte, error)
	Delete(string) error
}

// Service manages account metadata and their protected credentials.
type Service struct {
	db    *sql.DB
	store CredentialStore
}

// NewService constructs an account application service.
func NewService(db *sql.DB, store CredentialStore) *Service {
	return &Service{db: db, store: store}
}

// List returns account metadata without credential references or secrets.
func (s *Service) List(ctx context.Context) ([]Item, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.provider_id, a.name, a.enabled, a.created_at, a.updated_at, a.auth_method, a.auth_email, a.auth_plan, a.auth_state, a.auth_account_id,
		COALESCE(s.region,''),COALESCE(s.billing_mode,''),COALESCE(s.config_version,0)
		FROM accounts a LEFT JOIN stepfun_account_options s ON s.account_id=a.id ORDER BY a.created_at DESC, a.id DESC`)
	if err != nil {
		return nil, fmt.Errorf("account: list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]Item, 0)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("account: iterate: %w", err)
	}
	return items, nil
}

func (s *Service) get(ctx context.Context, id string) (Item, string, error) {
	var item Item
	var ref string
	var options stepfun.Options
	err := s.db.QueryRowContext(ctx, `
		SELECT a.id, a.provider_id, a.name, a.enabled, a.created_at, a.updated_at, a.credential_ref, a.auth_method, a.auth_email, a.auth_plan, a.auth_state, a.auth_account_id,
		COALESCE(s.region,''),COALESCE(s.billing_mode,''),COALESCE(s.config_version,0)
		FROM accounts a LEFT JOIN stepfun_account_options s ON s.account_id=a.id WHERE a.id = ?`, id).Scan(
		&item.ID, &item.ProviderID, &item.Name, &item.Enabled, &item.CreatedAt, &item.UpdatedAt, &ref, &item.AuthMethod, &item.AuthEmail, &item.AuthPlan, &item.AuthState, &item.AuthAccountID, &options.Region, &options.BillingMode, &options.Version,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, "", ErrNotFound
	}
	if err != nil {
		return Item{}, "", fmt.Errorf("account: read: %w", err)
	}
	if options.Region != "" {
		item.StepFun = &options
	}
	return item, ref, nil
}

type scanner interface{ Scan(dest ...any) error }

func scanItem(row scanner) (Item, error) {
	var item Item
	var options stepfun.Options
	if err := row.Scan(&item.ID, &item.ProviderID, &item.Name, &item.Enabled, &item.CreatedAt, &item.UpdatedAt, &item.AuthMethod, &item.AuthEmail, &item.AuthPlan, &item.AuthState, &item.AuthAccountID, &options.Region, &options.BillingMode, &options.Version); err != nil {
		return Item{}, fmt.Errorf("account: scan: %w", err)
	}
	if options.Region != "" {
		item.StepFun = &options
	}
	return item, nil
}

func validateName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" || !utf8.ValidString(name) || len(name) > maxNameBytes || utf8.RuneCountInString(name) > maxNameRunes {
		return "", fmt.Errorf("%w: name length", ErrInvalidInput)
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", fmt.Errorf("%w: name contains control characters", ErrInvalidInput)
		}
	}
	return name, nil
}

func resolveName(value string, key []byte) (string, error) {
	if strings.TrimSpace(value) == "" {
		// 名称按字符截取，避免 Unicode Key 被截断成无效 UTF-8。
		runes := []rune(string(key))
		value = string(runes[:min(defaultNameRunes, len(runes))])
	}
	return validateName(value)
}

func validateKey(value string) ([]byte, error) {
	key := strings.TrimSpace(value)
	if key == "" || !utf8.ValidString(key) || len(key) > maxKeyBytes {
		return nil, fmt.Errorf("%w: API Key length", ErrInvalidInput)
	}
	for _, r := range key {
		if unicode.IsControl(r) {
			return nil, fmt.Errorf("%w: API Key contains control characters", ErrInvalidInput)
		}
	}
	return []byte(key), nil
}

func validateID(id string) error {
	if len(id) == 0 || len(id) > maxIDBytes || !utf8.ValidString(id) || strings.TrimSpace(id) != id {
		return fmt.Errorf("%w: ID", ErrInvalidInput)
	}
	for _, r := range id {
		if unicode.IsControl(r) {
			return fmt.Errorf("%w: ID", ErrInvalidInput)
		}
	}
	return nil
}

func newID() (string, error) {
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return "acct_" + hex.EncodeToString(random[:]), nil
}

func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }
