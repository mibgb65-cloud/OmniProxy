package gateway

import (
	"context"
	"strconv"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/core/proxy"
)

// SetGoogleAccountRotationMode 只在启动前安装设置读取器；每次请求读取持久化值以支持热切换。
func (s *Service) SetGoogleAccountRotationMode(read func(context.Context) (string, error)) {
	s.googleRotationMode = read
}

// rotateGoogleAccounts 只重排同池的 Google 订阅候选，不新增重试，也不改变显式固定账号。
//
// Google 订阅账号用同一份官方额度，均衡模式按账号健康状态平分新请求；没有可用账号时
// 仍返回冷却错误，不向已耗尽账号发送探测消息。
func (s *Service) rotateGoogleAccounts(ctx context.Context, initial target, request proxy.Request) ([]proxy.AccountCandidate, error) {
	candidates := request.Candidates
	if request.FixedAccount || s.googleRotationMode == nil {
		return candidates, nil
	}
	mode, err := s.googleRotationMode(ctx)
	if err != nil || !validRotationMode(mode) {
		return nil, proxy.Failure("configuration_unavailable", 503)
	}
	if mode == "queue" || len(candidates) < 2 {
		return candidates, nil
	}
	items, err := s.accounts.List(ctx)
	if err != nil {
		return nil, proxy.Failure("configuration_unavailable", 503)
	}
	saved := make(map[string]account.Item, len(items))
	for _, item := range items {
		saved[item.ID] = item
	}
	eligible, weights := []int{}, []int{}
	unique := []proxy.AccountCandidate{}
	seen := map[string]bool{}
	var wait time.Duration
	config := provider.Configuration{ID: initial.snapshot.ProviderID, Name: initial.snapshot.ProviderName, Kind: initial.kind}
	for _, candidate := range candidates {
		if seen[candidate.AccountID] {
			continue
		}
		seen[candidate.AccountID] = true
		item, ok := saved[candidate.AccountID]
		if !ok || !item.Enabled || item.ProviderID != config.ID || item.AuthMethod != account.AuthSubscription ||
			item.AuthState != "authenticated" || candidate.UpdatedAt != "" && item.UpdatedAt != candidate.UpdatedAt {
			return nil, proxy.Failure("route_configuration_changed", 503)
		}
		current := initial
		current.accountID, current.updatedAt, current.snapshot = item.ID, item.UpdatedAt, snapshotTarget(config, item)
		if !sameTargetPool(initial, current) {
			return nil, proxy.Failure("route_configuration_changed", 503)
		}
		index := len(unique)
		unique = append(unique, candidate)
		if delay := s.failover.wait(item.ID, item.UpdatedAt, request.Model); delay > 0 {
			if wait == 0 || delay < wait {
				wait = delay
			}
			continue
		}
		// 已观察到的额度耗尽同样不参与均分：不让它先撞一次再换号。
		if delay := s.googleQuotaWait(ctx, current, request.Model, s.failover.now()); delay > 0 {
			if wait == 0 || delay < wait {
				wait = delay
			}
			continue
		}
		eligible, weights = append(eligible, index), append(weights, 1)
	}
	if len(eligible) == 0 {
		return nil, &proxy.Error{Code: "account_pool_cooling", StatusCode: 429, RetryAfter: strconv.Itoa(max(1, int(wait.Seconds())))}
	}
	if len(eligible) > maxHealthEntries {
		return nil, proxy.Failure("configuration_unavailable", 503)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	selected := s.rotation.choose(unique, eligible, weights)
	ordered := []proxy.AccountCandidate{unique[selected]}
	for index, candidate := range unique {
		if index != selected {
			ordered = append(ordered, candidate)
		}
	}
	return ordered, nil
}
