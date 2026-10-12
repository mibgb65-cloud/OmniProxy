package routing

import (
	"sort"
	"time"
)

// Account 是选择账号所需的最小元数据，避免核心依赖账号服务或凭证。
type Account struct {
	ID, ProviderID, CreatedAt string
	Enabled                   bool
}

// SelectAccount 返回启用账号中的最早创建者，固定账号也必须属于候选提供商。
func SelectAccount(accounts []Account, providers map[string]bool, fixedID string) (Account, int, string) {
	return SelectPreferredAccount(accounts, providers, fixedID, "")
}

// SelectPreferredAccount 优先选择指定服务商的可用账号；偏好无候选时保留原有稳定顺序。
// fixedID 仍是强制账号范围，不受服务商偏好影响。
func SelectPreferredAccount(accounts []Account, providers map[string]bool, fixedID, preferredProviderID string) (Account, int, string) {
	candidates := make([]Account, 0)
	times := make(map[string]time.Time, len(accounts))
	for _, item := range accounts {
		created, err := time.Parse(time.RFC3339Nano, item.CreatedAt)
		if err != nil {
			return Account{}, 0, "configuration_unavailable"
		}
		times[item.ID] = created
		if item.Enabled && providers[item.ProviderID] {
			candidates = append(candidates, item)
		}
	}
	if fixedID != "" {
		for _, item := range accounts {
			if item.ID != fixedID {
				continue
			}
			if !providers[item.ProviderID] {
				return Account{}, 0, "account_mismatch"
			}
			if !item.Enabled {
				return Account{}, 0, "account_disabled"
			}
			return item, 1, ""
		}
		return Account{}, 0, "account_missing"
	}
	if len(candidates) == 0 {
		return Account{}, 0, "no_enabled_account"
	}
	sort.Slice(candidates, func(i, j int) bool {
		leftPreferred := candidates[i].ProviderID == preferredProviderID
		rightPreferred := candidates[j].ProviderID == preferredProviderID
		if leftPreferred != rightPreferred {
			return leftPreferred
		}
		left, right := times[candidates[i].ID], times[candidates[j].ID]
		if !left.Equal(right) {
			return left.Before(right)
		}
		return candidates[i].ID < candidates[j].ID
	})
	return candidates[0], len(candidates), ""
}
