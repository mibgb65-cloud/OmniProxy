package agentinstall

import (
	"context"
	"errors"
	"time"

	runtime "omniproxy/internal/infrastructure/agentinstall"
)

type updateInstaller interface {
	Update(context.Context, string, func(runtime.Progress)) error
}

// StartUpdate 显式核对官方目标后启动后台更新，不接受前端指定版本或软件包。
func (s *Service) StartUpdate(ctx context.Context, id string) (Snapshot, error) {
	checked, err := s.updateTarget(ctx, id)
	if err != nil {
		return Snapshot{}, err
	}
	if checked.State == "current" {
		return s.Refresh(ctx), nil
	}
	if checked.State != "available" || !runtime.DesktopVersionAtLeast(checked.LatestVersion, checked.CurrentVersion) || runtime.DesktopVersionAtLeast(checked.CurrentVersion, checked.LatestVersion) {
		return Snapshot{}, errors.New("agentinstall: update_unavailable")
	}
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	s.mu.Lock()
	if s.closed || s.active != "" {
		s.mu.Unlock()
		return Snapshot{}, errors.New("agentinstall: busy")
	}
	installer, ok := s.backend.(updateInstaller)
	if !ok || !s.backend.Available() {
		s.mu.Unlock()
		return Snapshot{}, runtime.ErrUnavailable
	}
	if ctx.Err() != nil {
		s.mu.Unlock()
		return Snapshot{}, runtime.ErrUpdateInstallFailed
	}
	jobCtx, cancel := context.WithTimeout(context.Background(), s.installTimeout)
	item := s.items[id]
	item.Operation, item.UpdateVersion = "update", checked.LatestVersion
	item.InstallState, item.ErrorCode = "installing", ""
	item.Progress = &runtime.Progress{Phase: "preparing"}
	item.StartedAt, item.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano), ""
	s.items[id], s.active = item, id
	s.cancel, s.done = cancel, make(chan struct{})
	done := s.done
	s.mu.Unlock()
	go func() {
		defer close(done)
		defer cancel()
		s.runUpdate(jobCtx, installer, id, checked.LatestVersion)
	}()
	return s.snapshot(), nil
}

func (s *Service) runUpdate(ctx context.Context, installer updateInstaller, id, target string) {
	err := installer.Update(ctx, id, func(progress runtime.Progress) { s.updateProgress(id, progress) })
	s.mu.Lock()
	item := s.items[id]
	item.InstallState, item.Progress = "verifying", &runtime.Progress{Phase: "verifying"}
	s.items[id] = item
	s.mu.Unlock()
	checkCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	detected := s.backend.Inspect(checkCtx, id)
	cancel()
	code := ""
	if detected.State != "installed" || !runtime.DesktopVersionAtLeast(detected.Version, target) {
		code = "update_unavailable"
		if err != nil && !errors.Is(err, runtime.ErrUpdateUnavailable) {
			code = "update_failed"
		}
		if ctx.Err() != nil {
			code = "update_interrupted"
		}
	}
	s.finish(id, detected, code)
}
