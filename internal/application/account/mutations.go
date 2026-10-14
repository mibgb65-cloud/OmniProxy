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
	key, err := validateKey(input.APIKey)
	if err != nil {
		return Item{}, err
	}
	defer clear(key)
	name, err := resolveName(input.Name, key)
	if err != nil {
		return Item{}, err
	}
	if err := s.checkProvider(ctx, input.ProviderID); err != nil {
		return Item{}, err
	}
	options, err := s.prepareStep(ctx, input.ProviderID, input.StepFun, nil)
	if err != nil {
		return Item{}, err
	}
	if err := validateStepKey(options, key); err != nil {
		return Item{}, err
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
	item := Item{ID: id, ProviderID: input.ProviderID, Name: name, Enabled: true, CreatedAt: timestamp, UpdatedAt: timestamp, AuthMethod: AuthAPIKey}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Item{}, errors.Join(err, s.store.Delete(ref))
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO accounts (id, provider_id, name, credential_ref, enabled, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, item.ID, item.ProviderID, item.Name, ref, item.Enabled, item.CreatedAt, item.UpdatedAt)
	if err == nil {
		err = saveStep(ctx, tx, item.ID, options)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		// 文件存储与 SQLite 没有共同事务，插入失败必须清理刚写入的密文。
		cleanupErr := s.store.Delete(ref)
		return Item{}, errors.Join(fmt.Errorf("account: create: %w", err), cleanupErr)
	}
	item.StepFun = options
	return item, nil
}

func (s *Service) checkProvider(ctx context.Context, id string) error {
	var exists int
	if err := s.db.QueryRowContext(ctx, `SELECT 1 FROM providers WHERE id = ?`, id).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("account: check provider: %w", err)
	}
	return nil
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
	return s.deleteAccount(ctx, id, false)
}

// DeleteManaged 由订阅生命周期服务完成系统凭证注销后调用。
func (s *Service) DeleteManaged(ctx context.Context, id string) error {
	return s.deleteAccount(ctx, id, true)
}

func (s *Service) deleteAccount(ctx context.Context, id string, managed bool) error {
	if err := validateID(id); err != nil {
		return err
	}
	item, ref, err := s.get(ctx, id)
	if err != nil {
		return err
	}
	if item.AuthMethod == AuthSubscription && !managed {
		return fmt.Errorf("%w: managed deletion required", ErrInvalidInput)
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
