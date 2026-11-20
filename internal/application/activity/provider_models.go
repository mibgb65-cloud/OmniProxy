package activity

import (
	"context"
	"database/sql"
	"sort"
	"strings"

	"omniproxy/internal/core/proxy"
)

// ProviderModel 保存具体提供商与实际模型的累计，跨账号汇总且保留历史名称。
// Model 是统计显示名：推理档位变体已按家族归并；Variants 仅在被归并时给出原始型号 ID。
type ProviderModel struct {
	ProviderID   string   `json:"providerId"`
	ProviderName string   `json:"providerName"`
	Model        string   `json:"model"`
	Variants     []string `json:"variants,omitempty"`
	Summary      Summary  `json:"summary"`
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

// providerModels 读取按提供商与模型的累计，并按模型家族归并推理档位变体。
// 归并在读取时完成，不修改历史数据；同一个上游模型只有思考程度不同，统计上不拆行。
func (s *Service) providerModels(ctx context.Context, families map[string]string) ([]ProviderModel, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT provider_id,provider_name,model,input_tokens,output_tokens,requests,completed_usage FROM usage_provider_models ORDER BY provider_name,provider_id,input_tokens+output_tokens DESC,model`)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	type providerGroup struct {
		id    string
		name  string
		total map[string]*modelTotals
		order []string
	}
	groups := []*providerGroup{}
	index := map[string]*providerGroup{}
	variants := map[string]map[string]bool{}
	for rows.Next() {
		var providerID, providerName, model string
		var input, output, requests, complete int64
		if rows.Scan(&providerID, &providerName, &model, &input, &output, &requests, &complete) != nil {
			return nil, ErrUnavailable
		}
		group := index[providerID]
		if group == nil {
			group = &providerGroup{id: providerID, name: providerName, total: map[string]*modelTotals{}}
			index[providerID] = group
			groups = append(groups, group)
		}
		name := modelFamilyName(families, model)
		total := group.total[name]
		if total == nil {
			total = &modelTotals{}
			group.total[name] = total
			group.order = append(group.order, name)
		}
		total.add(input, output, requests, complete)
		key := providerID + "\x00" + name
		if variants[key] == nil {
			variants[key] = map[string]bool{}
		}
		variants[key][model] = true
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	items := []ProviderModel{}
	for _, group := range groups {
		sort.SliceStable(group.order, func(i, j int) bool {
			left, right := group.total[group.order[i]], group.total[group.order[j]]
			if left.weightsMore(*right) || right.weightsMore(*left) {
				return left.weightsMore(*right)
			}
			return strings.ToLower(group.order[i]) < strings.ToLower(group.order[j])
		})
		for _, name := range group.order {
			item := ProviderModel{ProviderID: group.id, ProviderName: group.name, Model: name}
			item.Summary = group.total[name].summary()
			item.Variants = variantNames(variants[group.id+"\x00"+name], name)
			items = append(items, item)
		}
	}
	return items, nil
}
