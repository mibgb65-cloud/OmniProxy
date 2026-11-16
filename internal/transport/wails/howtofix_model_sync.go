package wails

import (
	"context"
	"sync"

	"omniproxy/internal/application/modelcatalog"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/provider/howtofix"
)

// howToFixModelSyncState 保存 HowToFix 后台目录同步的唯一实例。
type howToFixModelSyncState struct {
	mu     sync.Mutex
	auto   *modelcatalog.AutoSync
	emit   func()
	closed bool
}

// startHowToFixModelSync 在后台补齐缺失的 HowToFix 账号目录；不阻塞启动和请求。
// 目录按账号所属分组发放，只有同步过的账号才有可路由模型。
func (s *AppService) startHowToFixModelSync() {
	accounts, err := s.accountService()
	if err != nil {
		s.logHowToFixModelSyncError()
		return
	}
	source := modelcatalog.New(s.db, accounts, provider.NewService(s.db), nil)
	s.howtofixSync.mu.Lock()
	if s.howtofixSync.closed || s.howtofixSync.auto != nil {
		s.howtofixSync.mu.Unlock()
		return
	}
	s.howtofixSync.mu.Unlock()
	auto := modelcatalog.StartAPIKeyAutoSync(context.Background(), source, howtofix.Kind, func(ctx context.Context) error {
		if s.agents != nil {
			return s.agents.RefreshCatalog(ctx)
		}
		return nil
	}, func(err error) {
		if err != nil {
			s.logHowToFixModelSyncError()
		}
		s.howtofixSync.mu.Lock()
		emit := s.howtofixSync.emit
		s.howtofixSync.mu.Unlock()
		if emit != nil {
			emit()
		}
	})
	s.howtofixSync.mu.Lock()
	if s.howtofixSync.closed {
		s.howtofixSync.mu.Unlock()
		auto.Close()
		return
	}
	s.howtofixSync.auto = auto
	s.howtofixSync.mu.Unlock()
}

func (s *AppService) requestHowToFixModelSync() {
	s.howtofixSync.mu.Lock()
	auto := s.howtofixSync.auto
	s.howtofixSync.mu.Unlock()
	if auto != nil {
		auto.Request()
	}
}

func (s *AppService) shutdownHowToFixModelSync() {
	s.howtofixSync.mu.Lock()
	s.howtofixSync.closed = true
	auto := s.howtofixSync.auto
	s.howtofixSync.mu.Unlock()
	if auto != nil {
		auto.Close()
	}
}

func (s *AppService) howToFixSyncStatus() modelcatalog.SyncStatus {
	s.howtofixSync.mu.Lock()
	auto := s.howtofixSync.auto
	s.howtofixSync.mu.Unlock()
	if auto == nil {
		return modelcatalog.SyncStatus{}
	}
	return auto.Status()
}

func (s *AppService) logHowToFixModelSyncError() {
	if s.log != nil {
		// 上游及本地配置错误可能包含路径或身份；只记录固定错误码，保留原快照供重试。
		s.log.Warn("HowToFix model auto sync incomplete; retry is available", "module", "modelcatalog", "code", "howtofix_auto_sync_failed")
	}
}
