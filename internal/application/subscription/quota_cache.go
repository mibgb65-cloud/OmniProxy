package subscription

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"omniproxy/internal/application/account"
)

// QuotaCache 仅持久化当前 profile 数据库内的公开显示快照。
type QuotaCache struct{ db *sql.DB }

func NewQuotaCache(db *sql.DB) *QuotaCache { return &QuotaCache{db: db} }

type quotaCacheQuery interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func readQuotaCache(ctx context.Context, db quotaCacheQuery, id string, scope quotaCacheScope) (*Quota, error) {
	var saved quotaCacheScope
	var raw []byte
	var fetched string
	err := db.QueryRowContext(ctx, `SELECT identity_key,profile_key,workspace_known,fetched_at,snapshot_json FROM subscription_quota_snapshots WHERE account_id=?`, id).
		Scan(&saved.identity, &saved.profile, &saved.workspaceKnown, &fetched, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, failure("quota_cache_unavailable")
	}
	if !scope.matches(saved) {
		return nil, nil
	}
	value, err := decodeQuotaSnapshot(raw)
	if err != nil || value.FetchedAt != fetched {
		return nil, failure("quota_cache_unavailable")
	}
	return &value, nil
}

func (s *QuotaCache) Read(ctx context.Context, item account.Item, profile string) (*Quota, error) {
	if s.db == nil {
		return nil, failure("quota_cache_unavailable")
	}
	return readQuotaCache(ctx, s.db, item.ID, quotaScope(item, profile))
}

// Save 在同一事务内合并并写入，失败不清除旧快照；账号改动后不能写回旧身份。
func (s *QuotaCache) Save(ctx context.Context, item account.Item, profile string, value Quota) (Quota, error) {
	value.ResetCredits = nil
	if s.db == nil || validateQuotaSnapshot(value) != nil {
		return Quota{}, failure("quota_cache_unavailable")
	}
	value.FromCache = false
	scope := quotaScope(item, profile)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Quota{}, failure("quota_cache_unavailable")
	}
	defer tx.Rollback()
	previous, err := readQuotaCache(ctx, tx, item.ID, scope)
	if err != nil {
		return Quota{}, err
	}
	if previous != nil {
		oldTime, _ := time.Parse(time.RFC3339, previous.FetchedAt)
		newTime, _ := time.Parse(time.RFC3339, value.FetchedAt)
		if newTime.Before(oldTime) {
			previous.FromCache = true
			return *previous, nil
		}
		value = mergeQuotaSnapshot(*previous, value)
	} else if value.FiveHour == nil && value.Weekly == nil {
		return value, nil // 尚未获得任何窗口，不把空结果覆盖为“成功历史”。
	}
	data, err := json.Marshal(value)
	if err != nil || len(data) > 16384 {
		return Quota{}, failure("quota_cache_unavailable")
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO subscription_quota_snapshots
		(account_id,identity_key,profile_key,workspace_known,fetched_at,snapshot_json)
		SELECT id,?,?,?,?,? FROM accounts WHERE id=? AND updated_at=? AND auth_method=? AND provider_id=? AND auth_email=? AND auth_account_id=?
		ON CONFLICT(account_id) DO UPDATE SET identity_key=excluded.identity_key,profile_key=excluded.profile_key,
		workspace_known=excluded.workspace_known,fetched_at=excluded.fetched_at,snapshot_json=excluded.snapshot_json`,
		scope.identity, scope.profile, scope.workspaceKnown, value.FetchedAt, string(data), item.ID, item.UpdatedAt, account.AuthSubscription, item.ProviderID, item.AuthEmail, item.AuthAccountID)
	if err != nil {
		return Quota{}, failure("quota_cache_unavailable")
	}
	count, err := result.RowsAffected()
	if err != nil || count != 1 || tx.Commit() != nil {
		return Quota{}, failure("quota_cache_unavailable")
	}
	return value, nil
}
