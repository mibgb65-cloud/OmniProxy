package quotaactivation

import (
	"context"
	"time"
)

// Start 启动一个后台循环；默认关闭时不会查询账号额度或调用模型。
func (s *Service) Start(ctx context.Context) {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return
	}
	runCtx, cancel := context.WithCancel(ctx)
	s.cancel, s.done = cancel, make(chan struct{})
	s.mu.Unlock()
	go s.run(runCtx)
	s.Wake(true)
}

func (s *Service) run(ctx context.Context) {
	defer close(s.done)
	timer := time.NewTicker(detectionInterval)
	defer timer.Stop()
	var last time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.force.Store(true)
		case <-s.wake:
		}
		if !s.force.Swap(false) && s.now().Sub(last) < wakeDebounce {
			continue
		}
		last = s.now()
		if err := s.Check(ctx); err != nil && ctx.Err() == nil {
			s.mu.Lock()
			s.status.ErrorCode = "check_failed"
			s.mu.Unlock()
		}
	}
}

// Wake 合并额度读取等信号，避免每个页面组件都启动一次批量检测。
func (s *Service) Wake(force bool) {
	if force {
		s.force.Store(true)
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

// SettingsChanged 关闭时取消当前批次；开启保存后立即检测。
func (s *Service) SettingsChanged(enabled bool) {
	s.mu.Lock()
	if !enabled && s.activeCancel != nil {
		s.activeCancel()
	}
	s.mu.Unlock()
	if enabled {
		s.Wake(true)
	}
}

// Close 停止后续发送并等待后台结束，遵守应用退出的时间边界。
func (s *Service) Close(ctx context.Context) error {
	s.Stop()
	s.mu.Lock()
	done := s.done
	s.mu.Unlock()
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return failure("stop_timeout")
	}
}

// Stop 先取消后台上下文，不等待订阅操作锁，允许网关停止后再统一等待退出。
func (s *Service) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}
