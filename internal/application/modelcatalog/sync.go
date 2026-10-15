package modelcatalog

import (
	"context"
	"encoding/json"
	"omniproxy/internal/application/account"
	core "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/infrastructure/codex"
	"omniproxy/internal/provider/deepseek"
	"omniproxy/internal/provider/google"
	"omniproxy/internal/provider/kimi"
	"omniproxy/internal/provider/minimax"
	"omniproxy/internal/provider/stepfun"
	"time"
)

// Sync 失败时保留上次成功目录，账号凭据版本改变后旧目录不参与路由。
func (s *Service) Sync(ctx context.Context, id string) ([]Item, error) {
	if len(id) == 0 || len(id) > 128 {
		return nil, ErrUnavailable
	}
	accounts, err := s.accounts.List(ctx)
	if err != nil {
		return nil, ErrUnavailable
	}
	var current account.Item
	for _, a := range accounts {
		if a.ID == id {
			current = a
			break
		}
	}
	if current.ID == "" {
		return nil, ErrUnavailable
	}
	providers, err := s.providers.List(ctx)
	if err != nil {
		return nil, ErrUnavailable
	}
	kind, base := "", ""
	for _, p := range providers {
		if p.ID == current.ProviderID {
			kind, base = p.Kind, p.BaseURL
		}
	}
	models := []codex.Model{}
	var initialVersion string
	if s.db.QueryRowContext(ctx, `SELECT version FROM account_catalog_versions WHERE id=?`, id).Scan(&initialVersion) != nil {
		return nil, ErrUnavailable
	}
	if kind == "openai" && current.AuthMethod == account.AuthSubscription {
		if s.subscription == nil {
			return nil, ErrUnavailable
		}
		models, err = s.subscription.Models(ctx, id)
	} else if kind == "deepseek" && current.AuthMethod == account.AuthAPIKey {
		err = s.accounts.UseAPIKey(ctx, id, func(key []byte) error {
			ids, e := s.client.DiscoverModels(ctx, key)
			for _, m := range ids {
				models = append(models, codex.Model{ID: m, Name: m, Metadata: deepseek.ModelMetadata(m)})
			}
			return e
		})
	} else if kind == "google" && current.AuthMethod == account.AuthAPIKey {
		base := ""
		for _, item := range providers {
			if item.ID == current.ProviderID {
				base = item.BaseURL
			}
		}
		err = s.accounts.UseAPIKey(ctx, id, func(key []byte) error {
			discovered, discoveryError := google.New(nil).Models(ctx, base, key)
			for _, item := range discovered {
				models = append(models, codex.Model{ID: item.ID, Name: item.Name, Metadata: item.Metadata})
			}
			return discoveryError
		})
	} else if kind == minimax.Kind && current.AuthMethod == account.AuthAPIKey {
		models, err = s.syncMiniMax(ctx, current)
	} else if kind == kimi.Kind && current.AuthMethod == account.AuthAPIKey {
		models, err = s.syncKimi(ctx, current)
	} else if kind == stepfun.Kind && current.AuthMethod == account.AuthAPIKey {
		models, err = s.syncStep(ctx, current)
	} else if kind == "mimo" && current.AuthMethod == account.AuthAPIKey {
		models, err = s.syncMiMo(ctx, id, base)
	} else {
		return nil, ErrUnavailable
	}
	if err != nil || len(models) > 1000 {
		return nil, ErrUnavailable
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer tx.Rollback()
	var version string
	if tx.QueryRowContext(ctx, `SELECT version FROM account_catalog_versions WHERE id=?`, id).Scan(&version) != nil || version != initialVersion {
		return nil, ErrUnavailable
	}
	confirmed, err := confirmedModels(ctx, tx, id, version)
	if err != nil {
		return nil, ErrUnavailable
	}
	// 旧官方核心未列出已实际调用成功的 6.1；仅保留同授权版本的这项补充，其他消失模型正常移除。
	if extra, found := confirmed["gpt-6.1-sol"]; found {
		present := false
		for _, model := range models {
			present = present || model.ID == extra.ID
		}
		if !present {
			models = append(models, extra)
		}
	}
	if len(models) > core.MaxModels {
		return nil, ErrUnavailable
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM account_models WHERE account_id=?`, id); err != nil {
		return nil, ErrUnavailable
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, m := range models {
		if prior, found := confirmed[m.ID]; found {
			m.Metadata.ContextWindow, m.Metadata.ContextVerified = prior.Metadata.ContextWindow, true
		}
		if _, err = core.Normalize([]core.Entry{{ID: m.ID, Name: m.Name, Metadata: m.Metadata}}); err != nil {
			return nil, ErrUnavailable
		}
		metadata, err := json.Marshal(m.Metadata)
		if err != nil || len(metadata) > 16384 {
			return nil, ErrUnavailable
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO account_models(account_id,model,name,fetched_at,account_version,metadata_json) VALUES(?,?,?,?,?,?)`, id, m.ID, m.Name, now, version, string(metadata)); err != nil {
			return nil, ErrUnavailable
		}
	}
	if tx.Commit() != nil {
		return nil, ErrUnavailable
	}
	return s.List(ctx)
}
