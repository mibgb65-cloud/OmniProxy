// Package modelcatalog 保存按账号隔离的官方模型快照。
package modelcatalog

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	core "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/infrastructure/codex"
	"omniproxy/internal/provider/deepseek"
	"omniproxy/internal/provider/kimi"
	"omniproxy/internal/provider/mimo"
	"omniproxy/internal/provider/minimax"
	"omniproxy/internal/provider/stepfun"
)

type Subscription interface {
	Models(context.Context, string) ([]codex.Model, error)
}
type Item struct {
	AccountID  string        `json:"accountId"`
	ProviderID string        `json:"providerId"`
	Model      string        `json:"model"`
	Name       string        `json:"name"`
	FetchedAt  string        `json:"fetchedAt"`
	Metadata   core.Metadata `json:"metadata,omitempty"`
	Source     string        `json:"source,omitempty"`
}
type Service struct {
	db           *sql.DB
	accounts     *account.Service
	providers    *provider.Service
	subscription Subscription
	client       *deepseek.Client
	stepfun      *stepfun.Client
	kimi         *kimi.Client
	minimax      *minimax.Client
	mimo         *mimo.Client
}

func New(db *sql.DB, accounts *account.Service, providers *provider.Service, subscription Subscription) *Service {
	return &Service{db, accounts, providers, subscription, deepseek.New(nil), stepfun.New(nil), kimi.New(nil), minimax.New(nil), mimo.New(nil)}
}

var ErrUnavailable = errors.New("models: unavailable")

func (s *Service) List(ctx context.Context) ([]Item, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT m.account_id,a.provider_id,m.model,m.name,m.fetched_at,m.metadata_json FROM account_models m JOIN accounts a ON a.id=m.account_id JOIN account_catalog_versions v ON v.id=a.id AND v.version=m.account_version ORDER BY a.provider_id,m.model LIMIT 10001`)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	items := []Item{}
	for rows.Next() {
		var i Item
		var metadata string
		if rows.Scan(&i.AccountID, &i.ProviderID, &i.Model, &i.Name, &i.FetchedAt, &metadata) != nil || len(metadata) > 16384 || json.Unmarshal([]byte(metadata), &i.Metadata) != nil || core.ValidateMetadata(i.Metadata) != nil {
			return nil, ErrUnavailable
		}
		items = append(items, i)
	}
	if rows.Err() != nil || len(items) > 10000 {
		return nil, ErrUnavailable
	}
	presets, err := s.presetItems(ctx, items)
	if err != nil || len(items)+len(presets) > 10000 {
		return nil, ErrUnavailable
	}
	step, err := s.stepPresets(ctx, items)
	if err != nil || len(items)+len(presets)+len(step) > 10000 {
		return nil, ErrUnavailable
	}
	kimiItems, err := s.kimiPresets(ctx, items)
	if err != nil || len(items)+len(presets)+len(step)+len(kimiItems) > 10000 {
		return nil, ErrUnavailable
	}
	items = append(append(append(items, presets...), step...), kimiItems...)
	miniItems, err := s.minimaxPresets(ctx, items)
	if err != nil || len(items)+len(miniItems) > 10000 {
		return nil, ErrUnavailable
	}
	return append(items, miniItems...), nil
}
