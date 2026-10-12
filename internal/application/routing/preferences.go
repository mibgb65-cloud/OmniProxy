package routing

import (
	"context"
	"fmt"
	"time"

	core "omniproxy/internal/core/routing"
)

const maxPreferences = 1000

func (s *Service) preferences(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT model,provider_id FROM model_provider_preferences ORDER BY model COLLATE BINARY LIMIT ?`, maxPreferences+1)
	if err != nil {
		return nil, fmt.Errorf("routing: configuration unavailable")
	}
	defer rows.Close()
	items := make(map[string]string)
	for rows.Next() {
		var model, providerID string
		if rows.Scan(&model, &providerID) != nil || !core.ValidModel(model) || validateID(providerID, false) != nil {
			return nil, fmt.Errorf("routing: configuration unavailable")
		}
		items[model] = providerID
	}
	if rows.Err() != nil {
		return nil, fmt.Errorf("routing: configuration unavailable")
	}
	if len(items) > maxPreferences {
		return nil, ErrLimitReached
	}
	return items, nil
}

// SetPreferredProvider 保存模型的服务商偏好；空 providerID 恢复自动选择。
func (s *Service) SetPreferredProvider(ctx context.Context, model, providerID string) error {
	if !core.ValidModel(model) || validateID(providerID, true) != nil {
		return ErrInvalidInput
	}
	if providerID == "" {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM model_provider_preferences WHERE model=?`, model); err != nil {
			return fmt.Errorf("routing: save failed")
		}
		return nil
	}
	data, err := s.snapshot(ctx)
	if err != nil {
		return err
	}
	// 只接受此模型当前可用的服务商，不允许把偏好变成未验证的跨模型映射。
	if !evaluate(model, core.Rule{ProviderID: providerID, Model: model}, true, data).Routable {
		return fmt.Errorf("routing: preferred provider unavailable")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("routing: save failed")
	}
	defer tx.Rollback()
	var count, existing int
	if tx.QueryRowContext(ctx, `SELECT COUNT(*),COUNT(CASE WHEN model=? THEN 1 END) FROM model_provider_preferences`, model).Scan(&count, &existing) != nil {
		return fmt.Errorf("routing: save failed")
	}
	if count >= maxPreferences && existing == 0 {
		return ErrLimitReached
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := tx.ExecContext(ctx, `INSERT INTO model_provider_preferences(model,provider_id,updated_at) VALUES(?,?,?) ON CONFLICT(model) DO UPDATE SET provider_id=excluded.provider_id,updated_at=excluded.updated_at`, model, providerID, now); err != nil {
		return fmt.Errorf("routing: save failed")
	}
	if tx.Commit() != nil {
		return fmt.Errorf("routing: save failed")
	}
	return nil
}
