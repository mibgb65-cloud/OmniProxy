package subscription

import "context"

// SetQuotaCache 在启动阶段安装持久快照存储，不改变实际额度观察的有效期。
func (s *Service) SetQuotaCache(cache *QuotaCache) { s.quotaCache = cache }

// CachedQuota 只读本地快照，不打开官方组件或查询网络；退出登录后仍留存但不展示。
func (s *Service) CachedQuota(ctx context.Context, id string) (*Quota, error) {
	item, profile, err := s.accounts.SubscriptionProfile(ctx, id)
	if err != nil {
		return nil, failure("invalid_account")
	}
	if item.AuthState == "logged_out" || item.AuthState == "reauth_required" || item.AuthEmail == "" || s.quotaCache == nil {
		return nil, nil
	}
	value, err := s.quotaCache.Read(ctx, item, profile)
	if value != nil {
		value.FromCache = true
	}
	return value, err
}
