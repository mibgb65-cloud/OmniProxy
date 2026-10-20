package googlesubscription

import (
	"context"

	"omniproxy/internal/application/account"
	"omniproxy/internal/provider/antigravity"
)

func (s *Service) SetQuotaCache(cache *QuotaCache) { s.quotaCache = cache }

// CachedQuota 只恢复公开快照，不等待可能正在联网刷新的 ops 锁。
// 存储查询同时核对当前身份和授权版本，重新授权在途的旧读取不能跨身份恢复。
func (s *Service) CachedQuota(ctx context.Context, id string) (*antigravity.Quota, error) {
	item, err := s.accounts.Metadata(ctx, id)
	if err != nil || item.AuthMethod != account.AuthSubscription || s.accounts.CheckGoogleProvider(ctx, item.ProviderID) != nil {
		return nil, antigravity.Failure("invalid_input")
	}
	if item.AuthState != "authenticated" || s.busy(id) || s.quotaCache == nil {
		return nil, nil
	}
	version, err := s.accounts.CatalogVersion(ctx, id)
	if err != nil {
		return nil, antigravity.Failure("quota_cache_unavailable")
	}
	return s.quotaCache.Read(ctx, item, version)
}

func (s *Service) persistQuota(ctx context.Context, item account.Item, version string, value antigravity.Quota) (antigravity.Quota, error) {
	if s.quotaCache == nil {
		return value, nil
	}
	return s.quotaCache.Save(ctx, item, version, value)
}

func mergeQuotaSnapshot(old, next antigravity.Quota) antigravity.Quota {
	if len(next.Groups) == 0 {
		return old
	}
	for _, previous := range old.Groups {
		groupIndex := -1
		for index, group := range next.Groups {
			if group.Name == previous.Name {
				groupIndex = index
				break
			}
		}
		if groupIndex < 0 {
			next.Groups = append(next.Groups, previous)
			next.FromCache = true
			continue
		}
		for _, saved := range previous.Buckets {
			found := false
			for index, bucket := range next.Groups[groupIndex].Buckets {
				if bucket.ID != saved.ID {
					continue
				}
				found = true
				// 新重置窗口不能继承旧窗口的百分比。
				if bucket.Window != saved.Window || bucket.ResetsAt != "" && saved.ResetsAt != "" && bucket.ResetsAt != saved.ResetsAt {
					break
				}
				if bucket.RemainingPercent == nil && saved.RemainingPercent != nil {
					next.Groups[groupIndex].Buckets[index].RemainingPercent = saved.RemainingPercent
					next.FromCache = true
				}
				if bucket.ResetsAt == "" && saved.ResetsAt != "" {
					next.Groups[groupIndex].Buckets[index].ResetsAt = saved.ResetsAt
					next.FromCache = true
				}
				break
			}
			if !found {
				next.Groups[groupIndex].Buckets = append(next.Groups[groupIndex].Buckets, saved)
				next.FromCache = true
			}
		}
	}
	return next
}
