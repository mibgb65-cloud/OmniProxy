package provider

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"omniproxy/internal/core/netproxy"
)

// Configuration is one saved endpoint configuration of a provider kind.
type Configuration struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Source  string `json:"source"`
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
	Enabled bool   `json:"enabled"`
	// ProxyMode 决定该提供商是否经代理出站：inherit | direct | proxy。
	ProxyMode string `json:"proxyMode"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

// CreateInput contains user-supplied fields for a new configuration.
type CreateInput struct {
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
}

// UpdateInput contains editable fields of an existing configuration.
type UpdateInput struct {
	Name    string `json:"name"`
	BaseURL string `json:"baseUrl"`
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
		SELECT id, kind, source, name, base_url, enabled, proxy_mode, created_at, updated_at
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

// Create validates and saves a custom OpenAI-compatible configuration.
func (s *Service) Create(ctx context.Context, input CreateInput) (Configuration, error) {
	name, err := validateName(input.Name)
	if err != nil {
		return Configuration{}, err
	}
	baseURL, err := validateBaseURL(input.BaseURL)
	if err != nil {
		return Configuration{}, err
	}
	if baseURL == "" {
		return Configuration{}, fmt.Errorf("%w: custom base URL is required", ErrInvalidInput)
	}
	id, err := newID()
	if err != nil {
		return Configuration{}, fmt.Errorf("provider: generate ID: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	item := Configuration{ID: id, Kind: KindOpenAICompatible, Source: SourceCustom, Name: name, BaseURL: baseURL, Enabled: true, ProxyMode: netproxy.ModeInherit, CreatedAt: now, UpdatedAt: now}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO providers (id, kind, source, name, base_url, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, item.ID, item.Kind, item.Source, item.Name, item.BaseURL, item.Enabled, item.CreatedAt, item.UpdatedAt); err != nil {
		return Configuration{}, fmt.Errorf("provider: create: %w", err)
	}
	return item, nil
}

func (s *Service) get(ctx context.Context, id string) (Configuration, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, kind, source, name, base_url, enabled, proxy_mode, created_at, updated_at
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
	if err := row.Scan(&item.ID, &item.Kind, &item.Source, &item.Name, &item.BaseURL, &item.Enabled, &item.ProxyMode, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return Configuration{}, fmt.Errorf("provider: scan: %w", err)
	}
	return item, nil
}
