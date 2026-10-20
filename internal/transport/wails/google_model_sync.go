package wails

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
	"omniproxy/internal/application/modelcatalog"
	"omniproxy/internal/application/provider"
)

func init() { application.RegisterEvent[SubscriptionNotice]("modelcatalog:updated") }

func (s *AppService) startGoogleModelSync() {
	accounts, err := s.accountService()
	if err != nil {
		s.logGoogleModelSyncError()
		return
	}
	google, err := s.googleSubscriptionService()
	if err != nil {
		s.logGoogleModelSyncError()
		return
	}
	source := modelcatalog.New(s.db, accounts, provider.NewService(s.db), nil)
	source.SetGoogleSubscription(google)
	s.googleAuth.mu.Lock()
	defer s.googleAuth.mu.Unlock()
	if s.googleAuth.closed || s.googleAuth.auto != nil {
		return
	}
	s.googleAuth.auto = modelcatalog.StartGoogleAutoSync(context.Background(), source, func(ctx context.Context) error {
		if s.agents != nil {
			return s.agents.RefreshCatalog(ctx)
		}
		return nil
	}, func(err error) {
		if err != nil {
			s.logGoogleModelSyncError()
		}
		s.googleAuth.mu.Lock()
		emit := s.googleAuth.emitModels
		s.googleAuth.mu.Unlock()
		if emit != nil {
			emit()
		}
	})
}

func (s *AppService) requestGoogleModelSync() {
	s.googleAuth.mu.Lock()
	auto := s.googleAuth.auto
	s.googleAuth.mu.Unlock()
	if auto != nil {
		auto.Request()
	}
}

func (s *AppService) bindModelCatalogEvents(app *application.App) {
	s.googleAuth.mu.Lock()
	s.googleAuth.emitModels = func() { app.Event.Emit("modelcatalog:updated", SubscriptionNotice{Changed: true}) }
	s.googleAuth.mu.Unlock()
}

func (s *AppService) logGoogleModelSyncError() {
	if s.log != nil {
		// 上游及本地配置错误可能包含路径或身份；只记录固定错误码，保留原快照供重试。
		s.log.Warn("Google model auto sync incomplete; retry is available", "module", "modelcatalog", "code", "google_auto_sync_failed")
	}
}
