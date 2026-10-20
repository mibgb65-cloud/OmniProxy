package activity

import (
	"context"
	"database/sql"
	"math"
	"strings"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/subscription"
	"omniproxy/internal/core/tokenprice"
)

// QuotaEstimate 按旧版公式推算当前窗口，仅基于保留的本机代理样本和基础参考价。
type QuotaEstimate struct {
	FetchedAt string               `json:"fetchedAt"`
	FiveHour  *QuotaWindowEstimate `json:"fiveHour,omitempty"`
	Weekly    *QuotaWindowEstimate `json:"weekly,omitempty"`
}

type QuotaWindowEstimate struct {
	Status           string  `json:"status"`
	TotalMin         float64 `json:"totalMin"`
	TotalMax         float64 `json:"totalMax"`
	RemainingMin     float64 `json:"remainingMin"`
	RemainingMax     float64 `json:"remainingMax"`
	UsedMin          float64 `json:"usedMin"`
	UsedMax          float64 `json:"usedMax"`
	Requests         int     `json:"requests"`
	UnpricedRequests int     `json:"unpricedRequests"`
	ResetAdjusted    bool    `json:"resetAdjusted"`
}

// EstimateQuota 不读取凭据、网络或其他账号的用量；与官方快照时刻对齐，避免新 Token 配旧百分比。
func (s *Service) EstimateQuota(ctx context.Context, item account.Item, quota subscription.Quota, now time.Time) (QuotaEstimate, error) {
	result := QuotaEstimate{FetchedAt: quota.FetchedAt}
	if item.ProviderID != "builtin_openai" || item.AuthMethod != account.AuthSubscription || item.AuthEmail == "" || item.AuthAccountID == "" {
		return result, ErrInvalid
	}
	var err error
	if quota.FiveHourEnabled {
		result.FiveHour, err = s.estimateWindow(ctx, item, quota, quota.FiveHour, "five_hour", 5*time.Hour, now)
		if err != nil {
			return result, err
		}
	}
	result.Weekly, err = s.estimateWindow(ctx, item, quota, quota.Weekly, "weekly", 7*24*time.Hour, now)
	return result, err
}

func (s *Service) estimateWindow(ctx context.Context, item account.Item, quota subscription.Quota, window *subscription.QuotaWindow, name string, duration time.Duration, now time.Time) (*QuotaWindowEstimate, error) {
	result := &QuotaWindowEstimate{Status: "missing_window"}
	if window == nil {
		return result, nil
	}
	observedAt := window.ObservedAt
	if observedAt == "" {
		if quota.FromCache {
			return result, nil // 旧版合并快照缺少独立窗口时刻，不用新 Token 配旧百分比。
		}
		observedAt = quota.FetchedAt
	}
	reset, err := time.Parse(time.RFC3339, window.ResetsAt)
	fetched, fetchErr := time.Parse(time.RFC3339, observedAt)
	if err != nil || fetchErr != nil || fetched.After(now.Add(time.Minute)) || !reset.After(fetched) || reset.Sub(fetched) > duration {
		return result, nil
	}
	if !reset.After(now) {
		result.Status = "expired"
		return result, nil
	}
	used := float64(100 - window.RemainingPercent)
	if window.UsedPercent != nil {
		used = *window.UsedPercent
	}
	if math.IsNaN(used) || math.IsInf(used, 0) || used < 5 || used > 100 {
		result.Status = "insufficient_usage"
		return result, nil
	}
	start := reset.Add(-duration)
	var baseline string
	err = s.db.QueryRowContext(ctx, `SELECT start_at FROM subscription_quota_estimate_baselines WHERE account_id=? AND window_type=? AND reset_at=?`, item.ID, name, window.ResetsAt).Scan(&baseline)
	if err != nil && err != sql.ErrNoRows {
		return nil, ErrUnavailable
	}
	if err == nil {
		cutoff, parseErr := time.Parse(time.RFC3339, baseline)
		if parseErr != nil || cutoff.After(fetched) {
			return result, nil
		}
		if cutoff.After(start) {
			start, result.ResetAdjusted = cutoff, true
		}
	}
	if err := s.windowCost(ctx, item, start, fetched, result); err != nil {
		return nil, err
	}
	if result.Status != "ready" {
		return result, nil
	}
	// 与旧版相同：窗口基础等值成本 / 上游已用比例，不用账号累计 Token 代替当前窗口。
	result.TotalMin, result.TotalMax = result.UsedMin*100/used, result.UsedMax*100/used
	result.RemainingMin, result.RemainingMax = result.TotalMin*(100-used)/100, result.TotalMax*(100-used)/100
	return result, nil
}

func (s *Service) windowCost(ctx context.Context, item account.Item, start, end time.Time, result *QuotaWindowEstimate) error {
	rows, err := s.db.QueryContext(ctx, `SELECT model,input_tokens,output_tokens,cached_input_tokens,usage_complete,state FROM request_records
		WHERE provider_id=? AND account_id=? AND lower(trim(email))=? AND workspace_id=?
		AND started_at>=? AND started_at<=? AND (finished_at='' OR finished_at<=?)
		AND (state IN ('success','pending') OR input_known=1 OR output_known=1)`,
		item.ProviderID, item.ID, strings.ToLower(strings.TrimSpace(item.AuthEmail)), item.AuthAccountID, timeText(start), timeText(end), timeText(end))
	if err != nil {
		return ErrUnavailable
	}
	defer rows.Close()
	incomplete := false
	for rows.Next() {
		var model, state string
		var input, output int64
		var cached sql.NullInt64
		var complete bool
		if rows.Scan(&model, &input, &output, &cached, &complete, &state) != nil {
			return ErrUnavailable
		}
		result.Requests++
		if !complete || state == "pending" {
			incomplete = true
			continue
		}
		price, ok := tokenprice.Lookup(model)
		if !ok {
			result.UnpricedRequests++
			continue
		}
		var cachedInput *int64
		if cached.Valid {
			cachedInput = &cached.Int64
		}
		low, high := price.Cost(input, output, cachedInput)
		result.UsedMin += low
		result.UsedMax += high
	}
	if rows.Err() != nil {
		return ErrUnavailable
	}
	switch {
	case result.UnpricedRequests > 0:
		result.Status = "unknown_price"
	case incomplete:
		result.Status = "missing_usage"
	case result.Requests == 0 || result.UsedMax <= 0:
		result.Status = "no_usage"
	default:
		result.Status = "ready"
	}
	return nil
}
