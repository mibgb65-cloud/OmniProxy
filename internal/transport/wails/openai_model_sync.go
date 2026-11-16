package wails

import (
	"context"
	"sync"

	"omniproxy/internal/application/modelcatalog"
	"omniproxy/internal/application/provider"
)

type openAIModelSyncState struct {
	mu     sync.Mutex
	auto   *modelcatalog.OpenAIAutoSync
	emit   func()
	closed bool
}

// startOpenAIModelSync 在后台补齐重新授权后失效的 OpenAI 目录；不阻塞启动和请求。
func (s *AppService) startOpenAIModelSync() {
	accounts, err := s.accountService()
	if err != nil {
		s.logOpenAIModelSyncError()
		return
	}
	subscription, err := s.subscriptionService()
	if err != nil {
		s.logOpenAIModelSyncError()
		return
	}
	source := modelcatalog.New(s.db, accounts, provider.NewService(s.db), subscription)
	s.openAISync.mu.Lock()
	if s.openAISync.closed || s.openAISync.auto != nil {
		s.openAISync.mu.Unlock()
		return
	}
	s.openAISync.mu.Unlock()
	auto := modelcatalog.StartOpenAIAutoSync(context.Background(), source, func(ctx context.Context) error {
		if s.agents != nil {
			return s.agents.RefreshCatalog(ctx)
		}
		return nil
	}, func(err error) {
		if err != nil {
			s.logOpenAIModelSyncError()
		}
		s.openAISync.mu.Lock()
		emit := s.openAISync.emit
		s.openAISync.mu.Unlock()
		if emit != nil {
			emit()
		}
	})
	s.openAISync.mu.Lock()
	if s.openAISync.closed {
		s.openAISync.mu.Unlock()
		auto.Close()
		return
	}
	s.openAISync.auto = auto
	s.openAISync.mu.Unlock()
}

func (s *AppService) requestOpenAIModelSync() {
	s.openAISync.mu.Lock()
	auto := s.openAISync.auto
	s.openAISync.mu.Unlock()
	if auto != nil {
		auto.Request()
	}
}

func (s *AppService) shutdownOpenAIModelSync() {
	s.openAISync.mu.Lock()
	s.openAISync.closed = true
	auto := s.openAISync.auto
	s.openAISync.mu.Unlock()
	if auto != nil {
		auto.Close()
	}
}

// openAIStatus 返回 OpenAI 后台目录同步的只读状态。
func (s *AppService) openAIStatus() modelcatalog.SyncStatus {
	s.openAISync.mu.Lock()
	auto := s.openAISync.auto
	s.openAISync.mu.Unlock()
	if auto == nil {
		return modelcatalog.SyncStatus{}
	}
	return auto.Status()
}

func (s *AppService) logOpenAIModelSyncError() {
	if s.log != nil {
		// 上游及本地配置错误可能包含路径或身份；只记录固定错误码，保留原快照供重试。
		s.log.Warn("OpenAI model auto sync incomplete; retry is available", "module", "modelcatalog", "code", "openai_auto_sync_failed")
	}
}
