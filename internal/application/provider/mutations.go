package provider

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Update changes the name and endpoint of a custom provider.
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
	current, err := s.get(ctx, id)
	if err != nil {
		return Configuration{}, err
	}
	if current.Source == SourceBuiltIn {
		return Configuration{}, ErrBuiltIn
	}
	if baseURL == "" && current.Kind == KindOpenAICompatible {
		return Configuration{}, fmt.Errorf("%w: custom base URL is required", ErrInvalidInput)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx, `
		UPDATE providers SET name = ?, base_url = ?, updated_at = ?
		WHERE id = ? AND source = ?`, name, baseURL, now, id, SourceCustom)
	if err != nil {
		return Configuration{}, fmt.Errorf("provider: update: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return Configuration{}, fmt.Errorf("provider: update result: %w", err)
	}
	if changed == 0 {
		if err := s.changeError(ctx, id); err != nil {
			return Configuration{}, err
		}
	}
	return s.get(ctx, id)
}

// SetEnabled changes only the enabled state of a built-in or custom provider.
func (s *Service) SetEnabled(ctx context.Context, id string, enabled bool) (Configuration, error) {
	if err := validateID(id); err != nil {
		return Configuration{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	result, err := s.db.ExecContext(ctx, `UPDATE providers SET enabled = ?, updated_at = ? WHERE id = ?`, enabled, now, id)
	if err != nil {
		return Configuration{}, fmt.Errorf("provider: set enabled: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return Configuration{}, fmt.Errorf("provider: set enabled result: %w", err)
	}
	if changed == 0 {
		return Configuration{}, ErrNotFound
	}
	return s.get(ctx, id)
}

// Delete removes only a custom provider without attached accounts.
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := validateID(id); err != nil {
		return err
	}
	// 单条 SQL 同时检查来源和账号关联，防止删除内置项或级联删除账号后留下密文。
	result, err := s.db.ExecContext(ctx, `
		DELETE FROM providers WHERE id = ? AND source = ?
		AND NOT EXISTS (SELECT 1 FROM accounts WHERE provider_id = ?)`, id, SourceCustom, id)
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
	if err := s.changeError(ctx, id); err != nil {
		return err
	}
	return ErrHasAccounts
}

func (s *Service) changeError(ctx context.Context, id string) error {
	var source string
	err := s.db.QueryRowContext(ctx, `SELECT source FROM providers WHERE id = ?`, id).Scan(&source)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("provider: check configuration: %w", err)
	}
	if source == SourceBuiltIn {
		return ErrBuiltIn
	}
	return nil
}
