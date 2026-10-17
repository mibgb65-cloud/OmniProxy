package agentinstall

import (
	"context"
	"errors"
	"time"

	runtime "omniproxy/internal/infrastructure/agentinstall"
)

type updateChecker interface {
	CheckUpdate(context.Context, string, string) (runtime.UpdateCheck, error)
}

// CheckUpdate 按用户请求核对当前安装版本，与安装互斥，不执行任何升级操作。
func (s *Service) CheckUpdate(ctx context.Context, id string) (runtime.UpdateCheck, error) {
	if _, err := runtime.Find(id); err != nil {
		return runtime.UpdateCheck{}, err
	}
	if id != "claude-desktop" {
		return runtime.UpdateCheck{}, runtime.ErrUpdateUnsupported
	}
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	s.mu.Lock()
	busy := s.closed || s.active != ""
	s.mu.Unlock()
	if busy {
		return runtime.UpdateCheck{}, errors.New("agentinstall: busy")
	}
	checker, ok := s.backend.(updateChecker)
	if !ok {
		return runtime.UpdateCheck{}, runtime.ErrUpdateUnsupported
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	detected := s.backend.Inspect(ctx, id)
	if detected.State != "installed" {
		return runtime.UpdateCheck{}, runtime.ErrDetectionFailed
	}
	s.mu.Lock()
	item := s.items[id]
	item.Detection = detected
	s.items[id] = item
	s.mu.Unlock()
	return checker.CheckUpdate(ctx, id, detected.Version)
}
