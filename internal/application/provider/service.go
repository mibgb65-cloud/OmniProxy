package provider

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Configuration is one saved endpoint configuration of a provider kind.
type Configuration struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	BaseURL   string `json:"baseUrl"`
	Enabled   bool   `json:"enabled"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// CreateInput contains user-supplied fields for a new configuration.
type CreateInput struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
	Enabled bool   `json:"enabled"`
}

// UpdateInput contains editable fields of an existing configuration.
type UpdateInput struct {
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
	Enabled bool   `json:"enabled"`
}

// Service persists provider configurations using the existing SQLite schema.
type Service struct {
	db *sql.DB
}

// NewService constructs the provider application service.
func NewService(db *sql.DB) *Service { return &Service{db: db} }

// List returns all saved provider configurations, newest first.
func (s *Service) List(ctx context.Context) ([]Configuration, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, name, base_url, enabled, created_at, updated_at
		FROM providers ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("provider: list: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]Configuration, 0)
	for rows.Next() {
		item, err := scanConfiguration(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("provider: iterate: %w", err)
	}
	return items, nil
}

// Create validates and saves a provider configuration.
func (s *Service) Create(ctx context.Context, input CreateInput) (Configuration, error) {
	if err := validateKind(input.Kind); err != nil {
		return Configuration{}, err
	}
	name, err := validateName(input.Name)
	if err != nil {
		return Configuration{}, err
	}
	baseURL, err := validateBaseURL(input.BaseURL)
	if err != nil {
		return Configuration{}, err
	}
	id, err := newID()
	if err != nil {
		return Configuration{}, fmt.Errorf("provider: generate ID: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	item := Configuration{ID: id, Kind: input.Kind, Name: name, BaseURL: baseURL, Enabled: input.Enabled, CreatedAt: now, UpdatedAt: now}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO providers (id, kind, name, base_url, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, item.ID, item.Kind, item.Name, item.BaseURL, item.Enabled, item.CreatedAt, item.UpdatedAt); err != nil {
		return Configuration{}, fmt.Errorf("provider: create: %w", err)
	}
	return item, nil
}

// Update changes the editable fields of a saved provider configuration.
func (s *Service) Update(ctx context.Context, id string, input UpdateInput) (Configuration, error) {
	if err := validateID(id); err != nil {
		return Configuration{}, err
	}
	name, err := validateName(input.Name)
	if err != nil {
		return Configuration{}, err
	}
	baseURL, err := validateBaseURL(input.BaseURL)
	if err != nil {
		return Configuration{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx, `
		UPDATE providers SET name = ?, base_url = ?, enabled = ?, updated_at = ?
		WHERE id = ?`, name, baseURL, input.Enabled, now, id)
	if err != nil {
		return Configuration{}, fmt.Errorf("provider: update: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return Configuration{}, fmt.Errorf("provider: update result: %w", err)
	}
	if changed == 0 {
		return Configuration{}, ErrNotFound
	}
	return s.get(ctx, id)
}

// Delete removes an unreferenced provider configuration.
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := validateID(id); err != nil {
		return err
	}
	// 单条 SQL 同时检查账号关联和删除，避免级联删除账号后留下孤儿密文。
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM providers WHERE id = ?
		AND NOT EXISTS (SELECT 1 FROM accounts WHERE provider_id = ?)`, id, id)
	if err != nil {
		return fmt.Errorf("provider: delete: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("provider: delete result: %w", err)
	}
	if changed > 0 {
		return nil
	}
	var exists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM providers WHERE id = ?)`, id).Scan(&exists); err != nil {
		return fmt.Errorf("provider: check delete result: %w", err)
	}
	if !exists {
		return ErrNotFound
	}
	return ErrHasAccounts
}

func (s *Service) get(ctx context.Context, id string) (Configuration, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, kind, name, base_url, enabled, created_at, updated_at
		FROM providers WHERE id = ?`, id)
	item, err := scanConfiguration(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Configuration{}, ErrNotFound
	}
	return item, err
}

type scanner interface{ Scan(dest ...any) error }

func scanConfiguration(row scanner) (Configuration, error) {
	var item Configuration
	if err := row.Scan(&item.ID, &item.Kind, &item.Name, &item.BaseURL, &item.Enabled, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return Configuration{}, fmt.Errorf("provider: scan: %w", err)
	}
	return item, nil
}
