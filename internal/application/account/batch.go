package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// CreateBatch 校验并统一保存一批账号，失败时回滚元数据并清理本批凭证。
func (s *Service) CreateBatch(ctx context.Context, input BatchCreateInput) (items []Item, err error) {
	if err := validateID(input.ProviderID); err != nil {
		return nil, err
	}
	if len(input.APIKeys) == 0 || len(input.APIKeys) > maxBatchKeys {
		return nil, fmt.Errorf("%w: batch count", ErrInvalidInput)
	}
	options, err := s.prepareStep(ctx, input.ProviderID, input.StepFun, nil)
	if err != nil {
		return nil, err
	}
	kimiOptions, err := s.prepareKimi(ctx, input.ProviderID, input.Kimi, nil)
	if err != nil {
		return nil, err
	}
	keys := make([][]byte, len(input.APIKeys))
	defer func() {
		for _, key := range keys {
			clear(key)
		}
	}()
	items = make([]Item, len(keys))
	timestamp := now()
	for i, value := range input.APIKeys {
		if len(value) > maxKeyBytes {
			return nil, fmt.Errorf("%w: batch key %d length", ErrInvalidInput, i+1)
		}
		key, keyErr := validateKey(value)
		if keyErr != nil {
			return nil, fmt.Errorf("account: batch key %d: %w", i+1, keyErr)
		}
		keys[i] = key
		if err := validateKimiKey(kimiOptions, key); err != nil {
			return nil, err
		}
		if err := validateStepKey(options, key); err != nil {
			return nil, err
		}
		name, nameErr := resolveName("", key)
		if nameErr != nil {
			return nil, fmt.Errorf("account: batch name %d: %w", i+1, nameErr)
		}
		id, idErr := newID()
		if idErr != nil {
			return nil, fmt.Errorf("account: generate batch ID: %w", idErr)
		}
		items[i] = Item{ID: id, ProviderID: input.ProviderID, Name: name, Enabled: true, CreatedAt: timestamp, UpdatedAt: timestamp, AuthMethod: AuthAPIKey}
	}
	if err := s.checkProvider(ctx, input.ProviderID); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("account: begin batch: %w", err)
	}
	refs := make([]string, 0, len(keys))
	defer func() {
		if err == nil {
			return
		}
		// 凭证文件不参与数据库事务；先回滚元数据，再清理本批每一个密文文件。
		if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
			err = errors.Join(err, fmt.Errorf("account: rollback batch: %w", rollbackErr))
		}
		for _, ref := range refs {
			if cleanupErr := s.store.Delete(ref); cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("account: clean batch credential: %w", cleanupErr))
			}
		}
	}()
	for i, key := range keys {
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("account: batch cancelled: %w", err)
		}
		ref, putErr := s.store.Put(key)
		if putErr != nil {
			return nil, fmt.Errorf("account: protect batch key %d: %w", i+1, putErr)
		}
		refs = append(refs, ref)
		item := items[i]
		if _, insertErr := tx.ExecContext(ctx, `
			INSERT INTO accounts (id, provider_id, name, credential_ref, enabled, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?)`, item.ID, item.ProviderID, item.Name, ref, item.Enabled, item.CreatedAt, item.UpdatedAt); insertErr != nil {
			return nil, fmt.Errorf("account: insert batch account %d: %w", i+1, insertErr)
		}
		if options != nil {
			copy := *options
			if optionErr := saveStep(ctx, tx, item.ID, &copy); optionErr != nil {
				return nil, optionErr
			}
			items[i].StepFun = &copy
		}
		if kimiOptions != nil {
			copy := *kimiOptions
			if err := saveKimi(ctx, tx, item.ID, &copy); err != nil {
				return nil, err
			}
			items[i].Kimi = &copy
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("account: commit batch: %w", err)
	}
	return items, nil
}
