package subscription

import (
	"context"
	"time"
)

// quotaRefreshInterval 是同一账号＋凭据版本的最短刷新间隔，与自动激活的检测周期一致。
// 刷新只由真实参与流量的账号触发，不存在全量后台探测或固定轮询。
const quotaRefreshInterval = 5 * time.Minute

// ObserveQuota 在请求结束后按需刷新该账号的额度观察，使网关在下一条消息前就知道
// 它是否仍然可用。调用不阻塞请求，同一账号＋版本在一个间隔内只刷新一次。
func (s *Service) ObserveQuota(id, version string) {
	if id == "" || version == "" {
		return
	}
	key := id + "\x00" + version
	s.refreshMu.Lock()
	if s.refreshClosed || !s.refreshDueLocked(key, time.Now()) {
		s.refreshMu.Unlock()
		return
	}
	s.refreshMu.Unlock()
	go s.refreshQuotaObservation(id)
}

// refreshDueLocked 记录本次刷新时刻并返回是否可以开始刷新；缓存有界，超限后整体重置。
func (s *Service) refreshDueLocked(key string, now time.Time) bool {
	if s.refreshLast == nil {
		s.refreshLast = map[string]time.Time{}
	}
	if last, ok := s.refreshLast[key]; ok && now.Sub(last) < quotaRefreshInterval {
		return false
	}
	if len(s.refreshLast) >= maxQuotaObservations {
		s.refreshLast = map[string]time.Time{}
	}
	s.refreshLast[key] = now
	return true
}

// refreshQuotaObservation 复用界面和自动激活的读取路径，失败只丢弃这次观察。
func (s *Service) refreshQuotaObservation(id string) {
	ctx, cancel := context.WithTimeout(s.refreshContext(), operationTimeout)
	defer cancel()
	_, _ = s.ReadQuota(ctx, id)
}

// refreshContext 绑定应用生命周期：退出后不再启动新的官方查询。
func (s *Service) refreshContext() context.Context {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	if s.refreshCtx == nil {
		s.refreshCtx, s.refreshCancel = context.WithCancel(context.Background())
	}
	return s.refreshCtx
}

// stopQuotaRefresh 只停止后续后台刷新；已经开始的查询按各自的 30 秒上限结束。
func (s *Service) stopQuotaRefresh() {
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	s.refreshClosed = true
	if s.refreshCancel != nil {
		s.refreshCancel()
	}
}
