package quotaactivation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/subscription"
)

type quotaWindow struct {
	kind  string
	value *subscription.QuotaWindow
}

func windows(quota subscription.Quota) []quotaWindow {
	result := []quotaWindow{{"weekly", quota.Weekly}}
	if quota.FiveHourEnabled {
		result = append(result, quotaWindow{"five_hour", quota.FiveHour})
	}
	return result
}
func identityVersion(item account.Item) string {
	sum := sha256.Sum256([]byte(item.AuthMethod + "\x00" + item.ProviderID + "\x00" + item.AuthAccountID + "\x00" + strings.ToLower(strings.TrimSpace(item.AuthEmail))))
	return hex.EncodeToString(sum[:])
}
func notStarted(window *subscription.QuotaWindow) bool {
	return window != nil && window.RemainingPercent == 100 && window.ResetsAt == ""
}

// NeedsActivation 只识别已返回且与页面未开始状态一致的窗口。
func NeedsActivation(quota subscription.Quota) bool {
	for _, window := range windows(quota) {
		if window.value != nil && (window.value.RemainingPercent <= 0 || window.value.RemainingPercent > 100) {
			return false
		}
		if window.value != nil && window.value.ResetsAt != "" {
			reset, err := time.Parse(time.RFC3339, window.value.ResetsAt)
			if err != nil || reset.Unix() <= 0 {
				return false
			}
		}
	}
	for _, window := range windows(quota) {
		if notStarted(window.value) {
			return true
		}
	}
	return false
}

// observe 保存官方重置目标；时间缺失或格式异常都不会猜测一个周期。
func (s *Service) observe(ctx context.Context, item account.Item, quota subscription.Quota) error {
	for _, window := range windows(quota) {
		if window.value == nil {
			continue
		}
		reset := window.value.ResetsAt
		if reset != "" {
			parsed, err := time.Parse(time.RFC3339, reset)
			if err != nil || parsed.Unix() <= 0 {
				continue
			}
		}
		_, err := s.db.ExecContext(ctx, `INSERT INTO codex_quota_activation_windows(account_id,window_kind,identity_version,observed_reset) VALUES(?,?,?,?)
ON CONFLICT(account_id,window_kind) DO UPDATE SET identity_version=excluded.identity_version,
observed_reset=CASE WHEN identity_version<>excluded.identity_version OR excluded.observed_reset<>'' THEN excluded.observed_reset ELSE observed_reset END,
attempted_cycle=CASE WHEN identity_version<>excluded.identity_version THEN '' ELSE attempted_cycle END`, item.ID, window.kind, identityVersion(item), reset)
		if err != nil {
			return failure("record_failed")
		}
	}
	return nil
}

// claim 在请求前登记所有未开始窗口；一条消息可同时激活两种窗口。
type windowClaim struct{ kind, cycle string }

func (s *Service) claim(ctx context.Context, item account.Item, quota subscription.Quota) ([]windowClaim, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, failure("record_failed")
	}
	defer tx.Rollback()
	claimed := []windowClaim{}
	for _, window := range windows(quota) {
		if !notStarted(window.value) {
			continue
		}
		var observed, attempted, version string
		if err := tx.QueryRowContext(ctx, `SELECT observed_reset,attempted_cycle,identity_version FROM codex_quota_activation_windows WHERE account_id=? AND window_kind=?`, item.ID, window.kind).Scan(&observed, &attempted, &version); err != nil {
			return nil, failure("record_failed")
		}
		if version != identityVersion(item) {
			return nil, failure("identity_changed")
		}
		cycle := observed
		if cycle == "" {
			cycle = "initial"
		} else {
			reset, err := time.Parse(time.RFC3339, cycle)
			// 官方刚给出未来目标又暂时缺失时，不将延迟快照当作新窗口重复激活。
			if err != nil {
				return nil, failure("record_failed")
			}
			if reset.After(s.now()) {
				continue
			}
		}
		if cycle == attempted {
			continue
		}
		if _, err := tx.ExecContext(ctx, `UPDATE codex_quota_activation_windows SET attempted_cycle=?,attempted_at=? WHERE account_id=? AND window_kind=?`, cycle, s.now().UTC().Format(time.RFC3339), item.ID, window.kind); err != nil {
			return nil, failure("record_failed")
		}
		claimed = append(claimed, windowClaim{window.kind, cycle})
	}
	if err := tx.Commit(); err != nil {
		return nil, failure("record_failed")
	}
	return claimed, nil
}

func activationConfirmed(before, after subscription.Quota, now time.Time) bool {
	for _, window := range windows(before) {
		if !notStarted(window.value) {
			continue
		}
		var fresh *subscription.QuotaWindow
		if window.kind == "weekly" {
			fresh = after.Weekly
		} else {
			fresh = after.FiveHour
		}
		if fresh == nil || fresh.ResetsAt == "" {
			return false
		}
		if reset, err := time.Parse(time.RFC3339, fresh.ResetsAt); err != nil || !reset.After(now) {
			return false
		}
	}
	return true
}

func windowsActive(quota subscription.Quota, now time.Time) bool {
	for _, window := range windows(quota) {
		if window.value == nil {
			return false
		}
		reset, err := time.Parse(time.RFC3339, window.value.ResetsAt)
		if err != nil || !reset.After(now) {
			return false
		}
	}
	return true
}

func (s *Service) reportQuota(item account.Item, quota subscription.Quota) error {
	// 只比较窗口，不含 FetchedAt；否则每次后台读取都会触发页面刷新与再次检测。
	body, err := json.Marshal(struct {
		Identity   string
		Five, Week *subscription.QuotaWindow
	}{identityVersion(item), quota.FiveHour, quota.Weekly})
	if err != nil {
		return failure("quota_unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.observations[item.ID] != string(body) {
		s.changed = true
		s.observations[item.ID] = string(body)
	}
	return nil
}
