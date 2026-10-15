package account

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"omniproxy/internal/infrastructure/credential"
	"omniproxy/internal/provider/kimi"
	"omniproxy/internal/provider/mimo"
	"omniproxy/internal/provider/minimax"
	"omniproxy/internal/provider/stepfun"
)

// ErrConflict 表示账号在凭证准备期间被其他操作修改，更新未提交。
var ErrConflict = errors.New("account: concurrent update")

// UpdateInput 更新指定账号；APIKey 留空保留原凭证，Name 留空自动命名。
type UpdateInput struct {
	MiniMax *minimax.Options `json:"minimax,omitempty"`
	MiMo    *mimo.Options    `json:"mimo,omitempty"`
	Kimi    *kimi.Options    `json:"kimi,omitempty"`
	StepFun *stepfun.Options `json:"stepfun,omitempty"`
	ID      string           `json:"id"`
	Name    string           `json:"name"`
	APIKey  string           `json:"apiKey"`
}

// UpdateResult 区分账号更新成功与提交后的旧凭证清理警告。
type UpdateResult struct {
	Account        Item `json:"account"`
	CleanupWarning bool `json:"cleanupWarning"`
}

// Update 安全更新账号；只有数据库引用切换成功后才清理原凭证。
func (s *Service) Update(ctx context.Context, input UpdateInput) (result UpdateResult, err error) {
	if err := validateID(input.ID); err != nil {
		return result, err
	}
	if len(input.Name) > maxNameBytes || len(input.APIKey) > maxKeyBytes {
		return result, fmt.Errorf("%w: update field length", ErrInvalidInput)
	}
	item, oldRef, err := s.get(ctx, input.ID)
	if err != nil {
		return result, err
	}
	options, err := s.prepareStep(ctx, item.ProviderID, input.StepFun, &item)
	if err != nil {
		return result, err
	}
	kimiOptions, err := s.prepareKimi(ctx, item.ProviderID, input.Kimi, &item)
	if err != nil {
		return result, err
	}
	mimoOptions, err := s.prepareMiMo(ctx, item.ProviderID, input.MiMo, &item)
	if err != nil {
		return result, err
	}
	minimaxOptions, err := s.prepareMiniMax(ctx, item.ProviderID, input.MiniMax, &item)
	if err != nil {
		return result, err
	}
	if item.AuthMethod == AuthSubscription {
		return s.updateSubscriptionName(ctx, item, input)
	}
	keyChanged := strings.TrimSpace(input.APIKey) != ""
	var key []byte
	if keyChanged {
		key, err = validateKey(input.APIKey)
		if err != nil {
			return result, err
		}
		defer clear(key)
		if err := validateMiMoKey(mimoOptions, key); err != nil {
			return result, err
		}
		if err := validateMiniMaxKey(minimaxOptions, key); err != nil {
			return result, err
		}
		if err := validateKimiKey(kimiOptions, key); err != nil {
			return result, err
		}
		if err := validateStepKey(options, key); err != nil {
			return result, err
		}
	}
	if !keyChanged && input.MiMo != nil {
		if err := s.UseAPIKeyAt(ctx, item.ID, item.UpdatedAt, func(currentKey []byte) error { return validateMiMoKey(mimoOptions, currentKey) }); err != nil {
			return result, err
		}
	}
	var name string
	if !keyChanged && input.MiniMax != nil {
		if err := s.UseAPIKeyAt(ctx, item.ID, item.UpdatedAt, func(currentKey []byte) error {
			return validateMiniMaxKey(minimaxOptions, currentKey)
		}); err != nil {
			return result, err
		}
	}
	if strings.TrimSpace(input.Name) != "" {
		name, err = validateName(input.Name)
	} else if keyChanged {
		name, err = resolveName("", key)
	} else {
		err = s.UseAPIKey(ctx, input.ID, func(currentKey []byte) error {
			name, err = resolveName("", currentKey)
			return err
		})
	}
	if err != nil {
		return result, err
	}
	newRef := oldRef
	if keyChanged {
		newRef, err = s.store.Put(key)
		if err != nil {
			return result, fmt.Errorf("account: protect replacement key: %w", err)
		}
	}
	var tx *sql.Tx
	committed := false
	defer func() {
		if committed {
			return
		}
		if tx != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
				err = errors.Join(err, fmt.Errorf("account: rollback update: %w", rollbackErr))
			}
		}
		if newRef != oldRef {
			if cleanupErr := s.store.Delete(newRef); cleanupErr != nil {
				err = errors.Join(err, fmt.Errorf("account: clean replacement key: %w", cleanupErr))
			}
		}
	}()
	tx, err = s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("account: begin update: %w", err)
	}
	timestamp := now()
	// 引用和时间戳共同检查，避免并发更新时覆盖其他修改或删除正在使用的密钥。
	updated, err := tx.ExecContext(ctx, `UPDATE accounts SET name = ?, credential_ref = ?, updated_at = ?
		WHERE id = ? AND credential_ref = ? AND updated_at = ?`, name, newRef, timestamp, item.ID, oldRef, item.UpdatedAt)
	if err != nil {
		return result, fmt.Errorf("account: update: %w", err)
	}
	changed, err := updated.RowsAffected()
	if err != nil {
		return result, fmt.Errorf("account: update result: %w", err)
	}
	if changed != 1 {
		return result, ErrConflict
	}
	if input.StepFun != nil && (item.StepFun == nil || !item.StepFun.SamePool(*options)) {
		if err = saveStep(ctx, tx, item.ID, options); err != nil {
			return result, err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM account_models WHERE account_id=?`, item.ID); err != nil {
			return result, err
		}
	}
	if input.Kimi != nil && (item.Kimi == nil || item.Kimi.Region != kimiOptions.Region) {
		if err = saveKimi(ctx, tx, item.ID, kimiOptions); err != nil {
			return result, err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM account_models WHERE account_id=?`, item.ID); err != nil {
			return result, err
		}
	}
	if input.MiniMax != nil && (item.MiniMax == nil || !item.MiniMax.SamePool(*minimaxOptions)) {
		if err = saveMiniMax(ctx, tx, item.ID, minimaxOptions); err != nil {
			return result, err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM account_models WHERE account_id=?`, item.ID); err != nil {
			return result, err
		}
	}
	if input.MiMo != nil && (item.MiMo == nil || !item.MiMo.SamePool(*mimoOptions)) {
		if err = saveMiMo(ctx, tx, item.ID, mimoOptions); err != nil {
			return result, err
		}
		if _, err = tx.ExecContext(ctx, `DELETE FROM account_models WHERE account_id=?`, item.ID); err != nil {
			return result, err
		}
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("account: commit update: %w", err)
	}
	committed = true
	item.Name, item.UpdatedAt = name, timestamp
	item.StepFun = options
	item.Kimi = kimiOptions
	item.MiniMax = minimaxOptions
	item.MiMo = mimoOptions
	result.Account = item
	if keyChanged && oldRef != "" {
		// 已提交后不能把清理失败误报为未保存，否则重试可能再次替换凭证。
		cleanupErr := s.store.Delete(oldRef)
		result.CleanupWarning = cleanupErr != nil && !errors.Is(cleanupErr, credential.ErrNotFound)
	}
	return result, nil
}
