package agentinstall

import (
	"context"
	"errors"
	"time"

	runtime "omniproxy/internal/infrastructure/agentinstall"
)

const updateCheckTTL = 5 * time.Minute

// updateTarget 复用刚验证的发布结果，避免检查成功后又因第二次联网失败而无法更新。
// 每次仍复核本机安装；过期或安装版本变化时重新查询，目标不来自前端。
func (s *Service) updateTarget(ctx context.Context, id string) (runtime.UpdateCheck, error) {
	if _, err := runtime.Find(id); err != nil {
		return runtime.UpdateCheck{}, err
	}
	if id != "claude-desktop" {
		return runtime.UpdateCheck{}, runtime.ErrUpdateUnsupported
	}
	s.refreshMu.Lock()
	s.mu.Lock()
	if s.closed || s.active != "" {
		s.mu.Unlock()
		s.refreshMu.Unlock()
		return runtime.UpdateCheck{}, errors.New("agentinstall: busy")
	}
	cached := s.lastUpdate
	s.mu.Unlock()
	if cached != nil && time.Since(cached.at) >= 0 && time.Since(cached.at) < updateCheckTTL {
		checkCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		detected := s.backend.Inspect(checkCtx, id)
		cancel()
		s.mu.Lock()
		item := s.items[id]
		item.Detection = detected
		s.items[id] = item
		if detected.State != "installed" || detected.Version != cached.result.CurrentVersion {
			s.lastUpdate = nil
		}
		s.mu.Unlock()
		if detected.State != "installed" {
			s.refreshMu.Unlock()
			return runtime.UpdateCheck{}, runtime.ErrDetectionFailed
		}
		if detected.Version == cached.result.CurrentVersion {
			s.refreshMu.Unlock()
			return cached.result, nil
		}
		if runtime.DesktopVersionAtLeast(detected.Version, cached.result.LatestVersion) {
			s.refreshMu.Unlock()
			return runtime.UpdateCheck{State: "current", CurrentVersion: detected.Version, LatestVersion: cached.result.LatestVersion}, nil
		}
	}
	s.refreshMu.Unlock()
	return s.CheckUpdate(ctx, id)
}
