package googlesubscription

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/provider/antigravity"
)

type QuotaCache struct{ db *sql.DB }

func NewQuotaCache(db *sql.DB) *QuotaCache { return &QuotaCache{db: db} }

func googleQuotaIdentity(item account.Item) string {
	raw, _ := json.Marshal([]string{item.ProviderID, strings.ToLower(strings.TrimSpace(item.AuthEmail)), item.AuthAccountID})
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func (s *QuotaCache) Read(ctx context.Context, item account.Item, version string) (*antigravity.Quota, error) {
	if s.db == nil {
		return nil, antigravity.Failure("quota_cache_unavailable")
	}
	var raw []byte
	var fetched string
	err := s.db.QueryRowContext(ctx, `SELECT q.snapshot_json,q.fetched_at FROM google_subscription_quota_snapshots q
		JOIN accounts a ON a.id=q.account_id JOIN account_catalog_versions v ON v.id=a.id
		WHERE q.account_id=? AND q.identity_key=? AND q.account_version=? AND v.version=q.account_version
		AND a.auth_method=? AND a.auth_state='authenticated' AND a.provider_id=? AND lower(trim(a.auth_email))=? AND a.auth_account_id=?`,
		item.ID, googleQuotaIdentity(item), version, account.AuthSubscription, item.ProviderID, strings.ToLower(strings.TrimSpace(item.AuthEmail)), item.AuthAccountID).Scan(&raw, &fetched)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil || len(raw) > 131072 {
		return nil, antigravity.Failure("quota_cache_unavailable")
	}
	var value antigravity.Quota
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&value) != nil || decoder.Decode(new(any)) != io.EOF ||
		!antigravity.ValidateQuotaSnapshot(value) || value.FetchedAt != fetched {
		return nil, antigravity.Failure("quota_cache_unavailable")
	}
	value.FromCache = true
	return &value, nil
}

func (s *QuotaCache) Save(ctx context.Context, item account.Item, version string, value antigravity.Quota) (antigravity.Quota, error) {
	if s.db == nil || !antigravity.ValidateQuotaSnapshot(value) {
		return antigravity.Quota{}, antigravity.Failure("quota_cache_unavailable")
	}
	stamp, _ := time.Parse(time.RFC3339Nano, value.FetchedAt)
	// 固定小数位和时区后才能用 SQLite 文本比较保证较旧响应不覆盖新快照。
	value.FetchedAt = stamp.UTC().Format("2006-01-02T15:04:05.000000000Z")
	value.FromCache = false
	previous, err := s.Read(ctx, item, version)
	if err != nil {
		return antigravity.Quota{}, err
	}
	if previous != nil {
		oldTime, _ := time.Parse(time.RFC3339Nano, previous.FetchedAt)
		newTime, _ := time.Parse(time.RFC3339Nano, value.FetchedAt)
		if newTime.Before(oldTime) {
			return *previous, nil
		}
		value = mergeQuotaSnapshot(*previous, value)
	}
	if len(value.Groups) == 0 {
		return value, nil
	}
	raw, err := json.Marshal(value)
	if err != nil || len(raw) > 131072 || !antigravity.ValidateQuotaSnapshot(value) {
		return antigravity.Quota{}, antigravity.Failure("quota_cache_unavailable")
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO google_subscription_quota_snapshots(account_id,identity_key,account_version,fetched_at,snapshot_json)
		SELECT a.id,?,?,?,? FROM accounts a JOIN account_catalog_versions v ON v.id=a.id
		WHERE a.id=? AND v.version=? AND a.updated_at=? AND a.auth_method=? AND a.auth_state='authenticated'
		ON CONFLICT(account_id) DO UPDATE SET identity_key=excluded.identity_key,account_version=excluded.account_version,
		fetched_at=excluded.fetched_at,snapshot_json=excluded.snapshot_json
		WHERE google_subscription_quota_snapshots.account_version!=excluded.account_version
		OR google_subscription_quota_snapshots.identity_key!=excluded.identity_key
		OR google_subscription_quota_snapshots.fetched_at<=excluded.fetched_at`,
		googleQuotaIdentity(item), version, value.FetchedAt, string(raw), item.ID, version, item.UpdatedAt, account.AuthSubscription)
	if err != nil {
		return antigravity.Quota{}, antigravity.Failure("quota_cache_unavailable")
	}
	if count, err := result.RowsAffected(); err != nil || count != 1 {
		return antigravity.Quota{}, antigravity.Failure("quota_changed")
	}
	return value, nil
}
