package routing

import (
	"context"
	"encoding/json"
	"omniproxy/internal/application/account"
	catalog "omniproxy/internal/core/modelcatalog"
	core "omniproxy/internal/core/routing"
	"omniproxy/internal/provider/atria"
	"omniproxy/internal/provider/mimo"
	"omniproxy/internal/provider/river"
)

type discoveredModel struct {
	accountID, providerID, model, name string
	metadata                           catalog.Metadata
}

func (s *Service) discovered(ctx context.Context) ([]discoveredModel, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT m.account_id,a.provider_id,m.model,m.name,m.metadata_json,p.kind FROM account_models m JOIN accounts a ON a.id=m.account_id JOIN providers p ON p.id=a.provider_id JOIN account_catalog_versions v ON v.id=a.id AND v.version=m.account_version ORDER BY a.provider_id,m.model,a.id LIMIT 10001`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []discoveredModel{}
	for rows.Next() {
		var m discoveredModel
		var metadata, kind string
		if err := rows.Scan(&m.accountID, &m.providerID, &m.model, &m.name, &metadata, &kind); err != nil {
			return nil, err
		}
		if len(metadata) > 16384 || json.Unmarshal([]byte(metadata), &m.metadata) != nil || catalog.ValidateMetadata(m.metadata) != nil || len(result) >= 10000 {
			return nil, catalog.ErrInvalid
		}
		if kind == river.Kind {
			m.metadata = river.ModelMetadata(river.UpstreamModel(m.model))
		}
		result = append(result, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result, err = s.withStepPresets(ctx, result)
	if err != nil {
		return nil, err
	}
	result, err = s.withKimiPresets(ctx, result)
	if err != nil {
		return nil, err
	}
	result, err = s.withMiniMaxPresets(ctx, result)
	if err != nil {
		return nil, err
	}
	result, err = s.withRiverPresets(ctx, result)
	if err != nil {
		return nil, err
	}
	return s.withMiMoPresets(ctx, result)
}
func targetSupports(data snapshot, providerID, accountID, model string) bool {
	for _, p := range data.providers {
		if p.ID == providerID && p.Kind == river.Kind {
			return riverTargetSupports(data, providerID, accountID, model)
		}
		if p.ID == providerID && p.Kind == "minimax" {
			return minimaxTargetSupports(data, providerID, accountID, model)
		}
		if p.ID == providerID && p.Kind == "kimi" {
			return kimiTargetSupports(data, providerID, accountID, model)
		}
		if p.ID == providerID && p.Kind == "stepfun" {
			return stepTargetSupports(data, providerID, accountID, model)
		}
		if p.ID == providerID && p.Kind == mimo.Kind {
			return mimoTargetSupports(data, providerID, accountID, model)
		}
	}
	for _, m := range data.models {
		if m.providerID == providerID && m.model == model && (accountID == "" || m.accountID == accountID) {
			return true
		}
	}
	for _, p := range data.providers {
		if p.ID == providerID && p.Kind == "deepseek" {
			return supportedModel(model)
		}
		if p.ID == providerID && p.Kind == atria.Kind {
			return model == atria.ModelID
		}
	}
	return false
}
func eligible(data snapshot, a account.Item, model string) bool {
	for _, p := range data.providers {
		if p.ID != a.ProviderID || !p.Enabled {
			continue
		}
		if p.Kind == "deepseek" && a.AuthMethod != account.AuthSubscription {
			return targetSupports(data, p.ID, a.ID, model)
		}
		if (p.Kind == atria.Kind || p.Kind == mimo.Kind || p.Kind == river.Kind) && a.AuthMethod == account.AuthAPIKey {
			return targetSupports(data, p.ID, a.ID, model)
		}
		if p.Kind == "minimax" && a.AuthMethod == account.AuthAPIKey {
			return a.MiniMax != nil && a.MiniMax.Valid() && targetSupports(data, p.ID, a.ID, model)
		}
		if p.Kind == "kimi" && a.AuthMethod == account.AuthAPIKey {
			return a.Kimi != nil && a.Kimi.Valid() && targetSupports(data, p.ID, a.ID, model)
		}
		if p.Kind == "stepfun" && a.AuthMethod == account.AuthAPIKey {
			return a.StepFun != nil && a.StepFun.Valid() && targetSupports(data, p.ID, a.ID, model)
		}
		if p.Kind == "google" && a.AuthMethod == account.AuthAPIKey {
			return targetSupports(data, p.ID, a.ID, model)
		}
		if p.Kind == "google" && a.AuthMethod == account.AuthSubscription && a.AuthState == "authenticated" {
			return targetSupports(data, p.ID, a.ID, model)
		}
		if p.Kind == "openai" && codexLogin(a) {
			return targetSupports(data, p.ID, a.ID, model)
		}
	}
	return false
}

// codexLogin 覆盖两种订阅登录来源：官方隔离登录与网关自己保管的 auth.json。
func codexLogin(a account.Item) bool {
	return (a.AuthMethod == account.AuthSubscription || a.AuthMethod == account.AuthCodexFile) && a.AuthState == "authenticated"
}

// Models 只公布至少存在可用账号的模型，供网关和本地 Agent 选择。
func (s *Service) Models(ctx context.Context) ([]string, error) {
	data, err := s.snapshot(ctx)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	seen := map[string]bool{}
	for _, m := range data.models {
		id := publicModel(data, m.providerID, m.model)
		for _, a := range data.accounts {
			if a.ID == m.accountID && a.Enabled && eligible(data, a, id) && !seen[id] {
				ids = append(ids, id)
				seen[id] = true
			}
		}
	}
	for _, m := range originalModels(data) {
		if !seen[m] && evaluate(m, core.Rule{}, false, data).Routable {
			ids = append(ids, m)
		}
	}
	return ids, nil
}
