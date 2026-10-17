package activity

import (
	"context"
	"database/sql"
)

func readDashboardProviders(ctx context.Context, tx *sql.Tx) ([]DashboardProvider, error) {
	// provider_id 是分组身份；名称优先当前配置，已删除资源仍保留历史快照。
	rows, err := tx.QueryContext(ctx, `SELECT u.provider_id,COALESCE(p.name,MAX(u.provider_name)),SUM(u.requests) AS total FROM usage_provider_models u LEFT JOIN providers p ON p.id=u.provider_id GROUP BY u.provider_id,CASE WHEN u.provider_id='' THEN u.provider_name ELSE '' END HAVING total>0 ORDER BY total DESC,u.provider_id,MAX(u.provider_name)`)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	items := []DashboardProvider{}
	for rows.Next() {
		var item DashboardProvider
		if rows.Scan(&item.ProviderID, &item.ProviderName, &item.Requests) != nil || item.Requests < 0 {
			return nil, ErrUnavailable
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	return items, nil
}
