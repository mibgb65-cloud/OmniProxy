package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"time"

	core "omniproxy/internal/core/routing"
)

const (
	maxPreferences        = 1000
	maxProviderOrder      = 100
	maxProviderOrderBytes = 14000
)

func validProviderOrder(ids []string) bool {
	if len(ids) > maxProviderOrder {
		return false
	}
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if validateID(id, false) != nil || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

func (s *Service) preferences(ctx context.Context) (map[string][]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT model,provider_id,provider_order FROM model_provider_preferences ORDER BY model COLLATE BINARY LIMIT ?`, maxPreferences+1)
	if err != nil {
		return nil, fmt.Errorf("routing: configuration unavailable")
	}
	defer rows.Close()
	items := make(map[string][]string)
	for rows.Next() {
		var model, providerID, encoded string
		if rows.Scan(&model, &providerID, &encoded) != nil || !core.ValidModel(model) || validateID(providerID, false) != nil || len(encoded) > maxProviderOrderBytes {
			return nil, fmt.Errorf("routing: configuration unavailable")
		}
		order := []string{providerID}
		// 空列兼容旧格式的单服务商写入；损坏的顺序必须报错，不能静默改变实际路由。
		if encoded != "" && json.Unmarshal([]byte(encoded), &order) != nil {
			return nil, fmt.Errorf("routing: configuration unavailable")
		}
		if !validProviderOrder(order) || len(order) == 0 || order[0] != providerID {
			return nil, fmt.Errorf("routing: configuration unavailable")
		}
		items[model] = order
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
// 已有多服务商顺序时，仅将所选服务商移到首位，保留其余顺序。
func (s *Service) SetPreferredProvider(ctx context.Context, model, providerID string) error {
	if !core.ValidModel(model) || validateID(providerID, true) != nil {
		return ErrInvalidInput
	}
	if providerID == "" {
		return s.SetProviderOrder(ctx, model, nil)
	}
	data, err := s.snapshot(ctx)
	if err != nil {
		return err
	}
	if !evaluate(model, core.Rule{ProviderID: providerID, Model: model}, true, data).Routable {
		return fmt.Errorf("routing: preferred provider unavailable")
	}
	order := []string{providerID}
	for _, id := range data.preferences[model] {
		if id != providerID {
			order = append(order, id)
		}
	}
	return s.SetProviderOrder(ctx, model, order)
}

// SetProviderOrder 保存同名模型的服务商使用顺序，空列表恢复自动选择。
func (s *Service) SetProviderOrder(ctx context.Context, model string, providerIDs []string) error {
	if !core.ValidModel(model) || !validProviderOrder(providerIDs) {
		return ErrInvalidInput
	}
	if len(providerIDs) == 0 {
		if _, err := s.db.ExecContext(ctx, `DELETE FROM model_provider_preferences WHERE model=?`, model); err != nil {
			return fmt.Errorf("routing: save failed")
		}
		return nil
	}
	data, err := s.snapshot(ctx)
	if err != nil {
		return err
	}
	for _, id := range providerIDs {
		// 可保留已保存的失效项供诊断，但不允许加入未知或不支持同名模型的新目标。
		if !slices.Contains(data.preferences[model], id) && !evaluate(model, core.Rule{ProviderID: id, Model: model}, true, data).Routable {
			return fmt.Errorf("routing: preferred provider unavailable")
		}
	}
	encoded, err := json.Marshal(providerIDs)
	if err != nil || len(encoded) > maxProviderOrderBytes {
		return ErrInvalidInput
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
	if _, err := tx.ExecContext(ctx, `INSERT INTO model_provider_preferences(model,provider_id,updated_at,provider_order) VALUES(?,?,?,?) ON CONFLICT(model) DO UPDATE SET provider_id=excluded.provider_id,updated_at=excluded.updated_at,provider_order=excluded.provider_order`, model, providerIDs[0], now, string(encoded)); err != nil {
		return fmt.Errorf("routing: save failed")
	}
	if tx.Commit() != nil {
		return fmt.Errorf("routing: save failed")
	}
	return nil
}
