package agentinstall

import (
	"context"
	"errors"
	"time"

	runtime "omniproxy/internal/infrastructure/agentinstall"
)

func (s *Service) install(ctx context.Context, id string) {
	err := s.backend.Install(ctx, id, func(progress runtime.Progress) { s.updateProgress(id, progress) })
	s.mu.Lock()
	item := s.items[id]
	item.InstallState = "verifying"
	item.Progress = &runtime.Progress{Phase: "verifying"}
	s.items[id] = item
	s.mu.Unlock()
	// 即使安装器返回错误，也重新检测实际结果，不能把已安装软件误报为失败。
	checkCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	detection := s.backend.Inspect(checkCtx, id)
	cancel()
	code := ""
	if detection.State != "installed" {
		code = "verification_failed"
		if err != nil {
			code = "install_failed"
		}
		if ctx.Err() != nil {
			code = "install_interrupted"
		}
	}
	s.finish(id, detection, code)
	if code == "" {
		s.mu.Lock()
		callback := s.afterInstalled
		s.mu.Unlock()
		if callback != nil {
			callback(id, detection)
		}
	}
}

// Start 校验 ID 和真实安装状态后启动后台任务；既有版本不会被升级或覆盖。
func (s *Service) Start(ctx context.Context, id string) (Snapshot, error) {
	product, err := runtime.Find(id)
	if err != nil {
		return Snapshot{}, err
	}
	if product.PackageID == "" {
		return Snapshot{}, runtime.ErrUnavailable
	}
	s.refreshMu.Lock()
	defer s.refreshMu.Unlock()
	s.mu.Lock()
	if s.closed || s.active != "" {
		s.mu.Unlock()
		return Snapshot{}, errors.New("agentinstall: busy")
	}
	s.active = id
	item := s.items[id]
	item.InstallState, item.ErrorCode = "checking", ""
	item.Progress, item.StartedAt, item.FinishedAt = &runtime.Progress{Phase: "preparing"}, time.Now().UTC().Format(time.RFC3339Nano), ""
	s.items[id] = item
	s.mu.Unlock()
	checkCtx, cancelCheck := context.WithTimeout(ctx, 8*time.Second)
	detection := s.backend.Inspect(checkCtx, id)
	cancelCheck()
	if detection.State != "not_installed" || !s.backend.Available() {
		code := ""
		if detection.State == "unknown" {
			code = "detection_failed"
		} else if detection.State != "installed" {
			code = "installer_unavailable"
		}
		s.finish(id, detection, code)
		if code != "" {
			return s.snapshot(), errors.New("agentinstall: " + code)
		}
		return s.snapshot(), nil
	}
	jobCtx, cancel := context.WithTimeout(context.Background(), s.installTimeout)
	s.mu.Lock()
	if s.closed {
		s.active = ""
		s.mu.Unlock()
		cancel()
		return Snapshot{}, errors.New("agentinstall: busy")
	}
	s.cancel, s.done = cancel, make(chan struct{})
	item = s.items[id]
	item.Detection, item.InstallState = detection, "installing"
	s.items[id] = item
	done := s.done
	s.mu.Unlock()
	go func() { defer close(done); defer cancel(); s.install(jobCtx, id) }()
	return s.snapshot(), nil
}

func (s *Service) finish(id string, detection runtime.Detection, code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.items[id]
	item.Detection, item.ErrorCode = detection, code
	item.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	item.Progress = nil
	item.InstallState = "completed"
	if code != "" {
		item.InstallState = "failed"
	}
	s.items[id], s.active = item, ""
}

// Close 在退出时停止等待安装任务；中断不代表软件已回滚，下一次启动重新检测。
func (s *Service) Close(ctx context.Context) error {
	s.mu.Lock()
	s.closed = true
	cancel, done := s.cancel, s.done
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-ctx.Done():
			return errors.New("agentinstall: shutdown_timeout")
		}
	}
	return nil
}
