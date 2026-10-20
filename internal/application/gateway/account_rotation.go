package gateway

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/core/proxy"
)

type quotaBalanceSource interface {
	QuotaBalanceWeight(string, string, time.Time) (int, bool)
}

type rotationKey struct{ id, version string }
type rotationState struct {
	mu     sync.Mutex
	scores map[rotationKey]int64
}

// SetAccountRotationMode 仅在启动前安装设置读取器；每次请求读取持久化值以支持热切换。
func (s *Service) SetAccountRotationMode(read func(context.Context) (string, error)) {
	s.rotationMode = read
}

// rotateAccounts 只重排已经解析的同池候选，不新增重试，也不改变显式固定账号。
func (s *Service) rotateAccounts(ctx context.Context, initial target, request proxy.Request) ([]proxy.AccountCandidate, error) {
	if initial.kind == "river" {
		return s.rotateRiverAccounts(ctx, initial, request)
	}
	candidates := request.Candidates
	if initial.kind != "openai" || initial.authMethod != account.AuthSubscription || request.FixedAccount || s.rotationMode == nil {
		return candidates, nil
	}
	mode, err := s.rotationMode(ctx)
	if err != nil || !validRotationMode(mode) {
		return nil, proxy.Failure("configuration_unavailable", 503)
	}
	if mode == "queue" || len(candidates) < 2 {
		return candidates, nil
	}
	now := s.failover.now()
	eligible := make([]int, 0, len(candidates))
	weights := make([]int, 0, len(candidates))
	allFresh := true
	var wait time.Duration
	rememberWait := func(delay time.Duration) {
		if wait == 0 || delay < wait {
			wait = delay
		}
	}
	seen := map[string]bool{}
	unique := map[int]bool{}
	identities := map[string]bool{}
	source, observesQuota := s.subscription.(quotaBalanceSource)
	// 一次读取账号快照，避免 N 个候选重复列举整个数据库；实际发送前仍逐次核验。
	items, err := s.accounts.List(ctx)
	if err != nil {
		return nil, proxy.Failure("configuration_unavailable", 503)
	}
	accounts := make(map[string]account.Item, len(items))
	for _, item := range items {
		accounts[item.ID] = item
	}
	config := provider.Configuration{ID: initial.snapshot.ProviderID, Name: initial.snapshot.ProviderName, Kind: initial.kind}
	for index, candidate := range candidates {
		if seen[candidate.AccountID] {
			continue
		}
		seen[candidate.AccountID] = true
		saved, exists := accounts[candidate.AccountID]
		if !exists || !saved.Enabled || saved.ProviderID != initial.snapshot.ProviderID || saved.AuthMethod != account.AuthSubscription || saved.AuthState != "authenticated" || candidate.UpdatedAt != "" && saved.UpdatedAt != candidate.UpdatedAt {
			return nil, proxy.Failure("route_configuration_changed", 503)
		}
		item := initial
		item.accountID, item.updatedAt, item.snapshot = saved.ID, saved.UpdatedAt, snapshotTarget(config, saved)
		if !sameTargetPool(initial, item) {
			return nil, proxy.Failure("route_configuration_changed", 503)
		}
		if item.snapshot.WorkspaceID != "" {
			identity := item.snapshot.WorkspaceID + "\x00" + strings.ToLower(strings.TrimSpace(item.snapshot.Email))
			if identities[identity] {
				continue
			}
			identities[identity] = true
		}
		unique[index] = true
		if delay := max(s.failover.wait(item.accountID, item.updatedAt, request.Model), s.subscriptionQuotaWait(item, now)); delay > 0 {
			rememberWait(delay)
			continue
		}
		weight, fresh := 1, false
		if observesQuota {
			weight, fresh = source.QuotaBalanceWeight(item.accountID, item.updatedAt, now)
		}
		if fresh && weight == 0 {
			rememberWait(time.Second)
			continue
		}
		if !fresh || weight < 1 || weight > 100 {
			allFresh, weight = false, 1
		}
		eligible, weights = append(eligible, index), append(weights, weight)
	}
	if len(eligible) == 0 {
		// 即使关闭单请求故障重试，平衡模式也不向已知耗尽账号发送探测消息。
		return nil, &proxy.Error{Code: "account_pool_cooling", StatusCode: 429, RetryAfter: strconv.Itoa(max(1, int(wait.Seconds())))}
	}
	if len(eligible) > 4096 {
		return nil, proxy.Failure("configuration_unavailable", 503)
	}
	if !allFresh {
		for index := range weights {
			weights[index] = 1
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	selected := s.rotation.choose(candidates, eligible, weights)
	ordered := make([]proxy.AccountCandidate, 0, len(candidates))
	ordered = append(ordered, candidates[selected])
	for index, candidate := range candidates {
		if index != selected && unique[index] {
			ordered = append(ordered, candidate)
		}
	}
	return ordered, nil
}

// 平滑加权轮询在锁内完成分配，并发请求不会一起读取同一个轮询位置。
// 分数只含公开账号标识和版本，不保存凭据或消息内容，重启后重新均衡。
func (s *rotationState) choose(candidates []proxy.AccountCandidate, eligible, weights []int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.scores == nil {
		s.scores = map[rotationKey]int64{}
	}
	missing := 0
	for _, offset := range eligible {
		candidate := candidates[offset]
		if _, exists := s.scores[rotationKey{candidate.AccountID, candidate.UpdatedAt}]; !exists {
			missing++
		}
	}
	if len(s.scores)+missing > 4096 {
		s.scores = map[rotationKey]int64{}
	}
	selected := eligible[0]
	var selectedKey rotationKey
	var highest, total int64
	for index, offset := range eligible {
		candidate := candidates[offset]
		key := rotationKey{candidate.AccountID, candidate.UpdatedAt}
		value := s.scores[key] + int64(weights[index])
		total += int64(weights[index])
		if index == 0 || value > highest {
			highest, selected, selectedKey = value, offset, key
		}
		s.scores[key] = value
	}
	s.scores[selectedKey] -= total
	return selected
}
