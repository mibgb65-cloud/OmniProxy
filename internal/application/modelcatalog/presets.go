package modelcatalog

import (
	"context"

	"omniproxy/internal/provider/atria"
	"omniproxy/internal/provider/mimo"
)

// presetItems 不读取凭证，也不伪造查询时间；预设与账号发现的来源在 UI 中区分。
func (s *Service) presetItems(ctx context.Context, cached []Item) ([]Item, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT a.id,a.provider_id,p.kind FROM accounts a
		JOIN providers p ON p.id=a.provider_id
		WHERE p.kind IN (?,?) AND a.auth_method='api_key' ORDER BY a.provider_id,a.id LIMIT 10001`, atria.Kind, mimo.Kind)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	items := []Item{}
	synced := map[string]bool{}
	for _, item := range cached {
		synced[item.AccountID] = true
	}
	for rows.Next() {
		var item Item
		var kind string
		if rows.Scan(&item.AccountID, &item.ProviderID, &kind) != nil || len(items) >= 10000 {
			return nil, ErrUnavailable
		}
		if kind == mimo.Kind {
			if synced[item.AccountID] {
				continue
			}
			for _, id := range mimo.Models() {
				item.Model, item.Name = id, id
				item.Metadata, item.Source = mimo.ModelMetadata(id), "preset"
				items = append(items, item)
			}
			continue
		}
		item.Model, item.Name = atria.ModelID, atria.ModelID
		item.Metadata, item.Source = atria.ModelMetadata(atria.ModelID), "preset"
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	return items, nil
}
