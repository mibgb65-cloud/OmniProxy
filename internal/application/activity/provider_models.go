package activity

import (
	"context"
	"database/sql"
	"omniproxy/internal/core/proxy"
)

// ProviderModel 保存具体提供商与实际模型的累计，跨账号汇总且保留历史名称。
type ProviderModel struct {
	ProviderID   string  `json:"providerId"`
	ProviderName string  `json:"providerName"`
	Model        string  `json:"model"`
	Summary      Summary `json:"summary"`
}

func updateProviderModel(ctx context.Context, tx *sql.Tx, event proxy.RequestEvent, newRequest bool, inputDelta, outputDelta int64, completeDelta int) error {
	if event.Target.ProviderID == "" {
		return nil
	}
	requests := 0
	if newRequest {
		requests = 1
	}
	// 用同一事务和同一差值更新，每个快照只计入一次；名称变化不改变分组身份。
	_, err := tx.ExecContext(ctx, `INSERT INTO usage_provider_models(provider_id,provider_name,model,requests,completed_usage,input_tokens,output_tokens) VALUES(?,?,?,?,?,?,?) ON CONFLICT(provider_id,model) DO UPDATE SET provider_name=excluded.provider_name,requests=requests+excluded.requests,completed_usage=completed_usage+excluded.completed_usage,input_tokens=input_tokens+excluded.input_tokens,output_tokens=output_tokens+excluded.output_tokens`, event.Target.ProviderID, event.Target.ProviderName, event.Model, requests, completeDelta, inputDelta, outputDelta)
	return err
}
func (s *Service) providerModels(ctx context.Context) ([]ProviderModel, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT provider_id,provider_name,model,input_tokens,output_tokens,requests,completed_usage FROM usage_provider_models ORDER BY provider_name,provider_id,input_tokens+output_tokens DESC,model`)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	items := []ProviderModel{}
	for rows.Next() {
		var item ProviderModel
		var input, output, requests, complete int64
		if rows.Scan(&item.ProviderID, &item.ProviderName, &item.Model, &input, &output, &requests, &complete) != nil {
			return nil, ErrUnavailable
		}
		item.Summary = summary(input, output, requests, complete)
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	return items, nil
}
