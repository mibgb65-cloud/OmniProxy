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
	"time"
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
}
type Service struct {
	db           *sql.DB
	accounts     *account.Service
	providers    *provider.Service
	subscription Subscription
	client       *deepseek.Client
}

func New(db *sql.DB, accounts *account.Service, providers *provider.Service, subscription Subscription) *Service {
	return &Service{db, accounts, providers, subscription, deepseek.New(nil)}
}

var ErrUnavailable = errors.New("models: unavailable")

func (s *Service) List(ctx context.Context) ([]Item, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT m.account_id,a.provider_id,m.model,m.name,m.fetched_at,m.metadata_json FROM account_models m JOIN accounts a ON a.id=m.account_id AND (a.credential_ref || ':' || a.auth_account_id)=m.account_version ORDER BY a.provider_id,m.model LIMIT 10001`)
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
	return items, nil
}

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
	kind := ""
	for _, p := range providers {
		if p.ID == current.ProviderID {
			kind = p.Kind
		}
	}
	models := []codex.Model{}
	var initialVersion string
	if s.db.QueryRowContext(ctx, `SELECT credential_ref || ':' || auth_account_id FROM accounts WHERE id=?`, id).Scan(&initialVersion) != nil {
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
	if tx.QueryRowContext(ctx, `SELECT credential_ref || ':' || auth_account_id FROM accounts WHERE id=?`, id).Scan(&version) != nil || version != initialVersion {
		return nil, ErrUnavailable
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM account_models WHERE account_id=?`, id); err != nil {
		return nil, ErrUnavailable
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, m := range models {
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
