package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Create saves an API Key in protected storage and only its reference in SQLite.
func (s *Service) Create(ctx context.Context, input CreateInput) (Item, error) {
	if err := validateID(input.ProviderID); err != nil {
		return Item{}, err
	}
	name, err := validateName(input.Name)
	if err != nil {
		return Item{}, err
	}
	key, err := validateKey(input.APIKey)
	if err != nil {
		return Item{}, err
	}
	defer clear(key)
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM providers WHERE id = ?`, input.ProviderID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Item{}, ErrNotFound
		}
		return Item{}, fmt.Errorf("account: check provider: %w", err)
	}
	id, err := newID()
	if err != nil {
		return Item{}, fmt.Errorf("account: generate ID: %w", err)
	}
	ref, err := s.store.Put(key)
	if err != nil {
		return Item{}, fmt.Errorf("account: protect API Key: %w", err)
	}
	timestamp := now()
	item := Item{ID: id, ProviderID: input.ProviderID, Name: name, Enabled: true, CreatedAt: timestamp, UpdatedAt: timestamp}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO accounts (id, provider_id, name, credential_ref, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, item.ID, item.ProviderID, item.Name, ref, item.Enabled, item.CreatedAt, item.UpdatedAt)
	if err != nil {
		// 文件存储与 SQLite 没有共同事务，插入失败必须清理刚写入的密文。
		cleanupErr := s.store.Delete(ref)
		return Item{}, errors.Join(fmt.Errorf("account: create: %w", err), cleanupErr)
	}
	return item, nil
}

// SetEnabled changes the account's enabled state without exposing its credential.
func (s *Service) SetEnabled(ctx context.Context, id string, enabled bool) (Item, error) {
	if err := validateID(id); err != nil {
		return Item{}, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE accounts SET enabled = ?, updated_at = ? WHERE id = ?`, enabled, now(), id)
	if err != nil {
		return Item{}, fmt.Errorf("account: set enabled: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return Item{}, fmt.Errorf("account: set enabled result: %w", err)
	}
	if changed == 0 {
		return Item{}, ErrNotFound
	}
	item, _, err := s.get(ctx, id)
	return item, err
}

// Delete removes account metadata, then its protected credential.
func (s *Service) Delete(ctx context.Context, id string) error {
	if err := validateID(id); err != nil {
		return err
	}
	_, ref, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("account: delete: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("account: delete result: %w", err)
	}
	if changed == 0 {
		return ErrNotFound
	}
	if ref == "" {
		return nil // 旧账号可能只有元数据，没有密钥引用。
	}
	if err := s.store.Delete(ref); err != nil {
		return fmt.Errorf("account: remove protected API Key: %w", err)
	}
	return nil
}
