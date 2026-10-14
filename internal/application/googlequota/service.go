// Package googlequota 展示 Gemini 项目绑定与本机观察，不伪造官方余额或剩余额度。
package googlequota

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"time"
	_ "time/tzdata"

	"omniproxy/internal/core/routing"
	"omniproxy/internal/provider/google"
)

var ErrInvalid = errors.New("google: invalid_input")
var ErrUnavailable = errors.New("google: unavailable")
var ErrChanged = errors.New("google: account_changed")
var projectPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)

type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }

type BindingInput struct {
	AccountID        string `json:"accountId"`
	ProjectID        string `json:"projectId"`
	AccountUpdatedAt string `json:"accountUpdatedAt"`
}

// Bind 项目由用户声明，未做 Cloud 身份验证；事务检查编辑期间账号是否更换。
func (s *Service) Bind(ctx context.Context, input BindingInput) error {
	if len(input.AccountID) == 0 || len(input.AccountID) > 128 || len(input.AccountUpdatedAt) > 64 || input.AccountUpdatedAt == "" || input.ProjectID != "" && !projectPattern.MatchString(input.ProjectID) {
		return ErrInvalid
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ErrUnavailable
	}
	defer tx.Rollback()
	var version, updated string
	if tx.QueryRowContext(ctx, `SELECT a.credential_ref || ':' || a.auth_account_id,a.updated_at FROM accounts a JOIN providers p ON p.id=a.provider_id WHERE a.id=? AND p.kind='google' AND a.auth_method='api_key'`, input.AccountID).Scan(&version, &updated) != nil {
		return ErrInvalid
	}
	if updated != input.AccountUpdatedAt {
		return ErrChanged
	}
	if input.ProjectID == "" {
		_, err = tx.ExecContext(ctx, `DELETE FROM google_project_bindings WHERE account_id=?`, input.AccountID)
	} else {
		_, err = tx.ExecContext(ctx, `INSERT INTO google_project_bindings(account_id,project_id,credential_version,updated_at) VALUES(?,?,?,?) ON CONFLICT(account_id) DO UPDATE SET project_id=excluded.project_id,credential_version=excluded.credential_version,updated_at=excluded.updated_at`, input.AccountID, input.ProjectID, version, time.Now().UTC().Format(time.RFC3339Nano))
	}
	if err != nil || tx.Commit() != nil {
		return ErrUnavailable
	}
	return nil
}

// Record 只保存官方错误中的安全字段，成功用量继续由现有 activity 统计。
func (s *Service) Record(ctx context.Context, id, updatedAt, model string, notice google.Failure) error {
	if len(id) == 0 || len(id) > 128 || updatedAt == "" || len(updatedAt) > 64 || !routing.ValidModel(model) || !google.ValidMetric(notice.Metric) || notice.RetrySeconds < 0 || notice.RetrySeconds > 86400 || notice.Limit != nil && (*notice.Limit < 0 || *notice.Limit > 1e12) {
		return ErrInvalid
	}
	switch notice.Category {
	case "rate_limited", "daily_exhausted", "payment_required", "authentication_failed", "permission_denied":
	default:
		return ErrInvalid
	}
	now := time.Now().UTC()
	retry := ""
	if notice.RetrySeconds > 0 {
		retry = now.Add(time.Duration(notice.RetrySeconds) * time.Second).Format(time.RFC3339Nano)
	}
	if notice.Category == "daily_exhausted" {
		retry = NextDailyReset(now).Format(time.RFC3339Nano)
	}
	result, err := s.db.ExecContext(ctx, `INSERT INTO google_resource_observations(account_id,model,credential_version,category,metric,quota_limit,retry_at,observed_at)
 SELECT a.id,?,a.credential_ref || ':' || a.auth_account_id,?,?,?,?,? FROM accounts a JOIN providers p ON p.id=a.provider_id WHERE a.id=? AND a.updated_at=? AND p.kind='google' AND a.auth_method='api_key'
 ON CONFLICT(account_id,model) DO UPDATE SET credential_version=excluded.credential_version,category=excluded.category,metric=excluded.metric,quota_limit=excluded.quota_limit,retry_at=excluded.retry_at,observed_at=excluded.observed_at`, model, notice.Category, notice.Metric, notice.Limit, retry, now.Format(time.RFC3339Nano), id, updatedAt)
	if err != nil {
		return ErrUnavailable
	}
	count, err := result.RowsAffected()
	if err != nil {
		return ErrUnavailable
	}
	if count != 1 {
		return ErrChanged
	}
	_, err = s.db.ExecContext(ctx, `DELETE FROM google_resource_observations WHERE rowid IN (SELECT rowid FROM google_resource_observations ORDER BY observed_at DESC LIMIT -1 OFFSET 1000)`)
	if err != nil {
		return ErrUnavailable
	}
	return nil
}

// NextDailyReset 使用 Google 的太平洋日历日，不能硬编码 UTC 偏移或自动宣称恢复满额。
func NextDailyReset(now time.Time) time.Time {
	location, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		panic("embedded Pacific timezone unavailable")
	}
	local := now.In(location)
	return time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, location).UTC()
}
