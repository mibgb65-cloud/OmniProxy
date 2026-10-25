package subscription

import (
	"context"
	"errors"

	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/codex"
)

// UsageSource 读取官方额度接口的公开快照；实现不得返回令牌或原始响应。
type UsageSource interface {
	Read(context.Context, []byte, string) (codex.RateLimits, error)
}

// SetUsageSource 安装 HTTP 额度读取；未安装时只使用官方 stdio 链路。
func (s *Service) SetUsageSource(source UsageSource) { s.usage = source }

// SetWarn 安装脱敏告警回调；回调只接收固定文案与稳定码。
func (s *Service) SetWarn(warn func(string, ...any)) { s.warn = warn }

func (s *Service) warnf(message string, args ...any) {
	if s.warn != nil {
		s.warn(message, args...)
	}
}

// readQuotaHTTP 用隔离授权材料直接读取官方额度接口，不启动官方组件。
// 返回 ok 表示已得到确定答案（额度或授权被拒）；ok 为 false 时调用方回退 stdio 链路。
// 续期需求、传输失败与协议失败一律交由既有链路处理，因此不改变账号状态。
func (s *Service) readQuotaHTTP(ctx context.Context, item account.Item, profile string) (Quota, bool, error) {
	if s.usage == nil || item.AuthAccountID == "" || item.AuthEmail == "" {
		return Quota{}, false, nil
	}
	path, err := s.profilePath(profile)
	if err != nil {
		return Quota{}, false, nil
	}
	var access []byte
	var officialID string
	borrow := func(value []byte, accountID string) error {
		access, officialID = append([]byte(nil), value...), accountID
		return nil
	}
	if err := s.readToken(ctx, path, item.AuthAccountID, item.AuthEmail, borrow); err != nil {
		clear(access)
		return Quota{}, false, nil
	}
	defer clear(access)
	// 本地材料已按受保护存储核对成员与账号；不一致时不使用它访问网络。
	if officialID != item.AuthAccountID {
		return Quota{}, false, nil
	}
	result, err := s.usage.Read(ctx, access, officialID)
	switch {
	case err == nil:
	case errors.Is(err, codex.ErrUsageUnauthorized):
		return Quota{}, true, err
	default:
		return Quota{}, false, nil
	}
	quota, err := normalizeQuota(result, item.AuthPlan)
	if err != nil {
		return Quota{}, false, nil
	}
	return quota, true, nil
}

// publishQuota 记忆观察、保存显示快照并返回额度；快照写入失败只告警，不否决已读到的结果。
func (s *Service) publishQuota(ctx context.Context, item account.Item, profile string, quota Quota, display bool) (Quota, error) {
	s.rememberQuota(item.ID, item.UpdatedAt, quota)
	if s.quotaCache == nil {
		return quota, nil
	}
	snapshot, err := s.quotaCache.Save(ctx, item, profile, quota)
	if err != nil {
		s.warnf("subscription quota snapshot not saved", "module", "subscription", "operation", "quota", "code", ErrorCode(err))
		return quota, nil
	}
	if display {
		snapshot.ResetCredits = quota.ResetCredits
		return snapshot, nil
	}
	return quota, nil
}
