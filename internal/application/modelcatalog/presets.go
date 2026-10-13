package modelcatalog

import (
	"context"

	"omniproxy/internal/provider/atria"
)

// presetItems 不读取凭证，也不伪造查询时间；预设与账号发现的来源在 UI 中区分。
func (s *Service) presetItems(ctx context.Context) ([]Item, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT a.id,a.provider_id FROM accounts a
		JOIN providers p ON p.id=a.provider_id
		WHERE p.kind=? AND a.auth_method='api_key' ORDER BY a.provider_id,a.id LIMIT 10001`, atria.Kind)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	items := []Item{}
	for rows.Next() {
		var item Item
		if rows.Scan(&item.AccountID, &item.ProviderID) != nil || len(items) >= 10000 {
			return nil, ErrUnavailable
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
