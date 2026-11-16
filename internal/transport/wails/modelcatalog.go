package wails

import (
	"context"

	"omniproxy/internal/application/modelcatalog"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/provider/workbuddyai"
)

// wireModelCatalog 给目录服务接上「按账号发现目录」的订阅通道。
//
// 手动同步（ModelCatalog／ModelSync）与后台自动同步各建一套 service，但接线必须一致：
// 少接一路时对应服务商的同步会只返回 unavailable，界面只能显示「同步失败」。
func (s *AppService) wireModelCatalog(service *modelcatalog.Service) error {
	google, err := s.googleSubscriptionService()
	if err != nil {
		return err
	}
	service.SetGoogleSubscription(google)
	workbuddy, err := s.workbuddySubscriptionService()
	if err != nil {
		return err
	}
	service.SetWorkBuddySubscription(workbuddy)
	return nil
}

func (s *AppService) modelCatalog() (*modelcatalog.Service, error) {
	accounts, err := s.accountService()
	if err != nil {
		return nil, err
	}
	subscription, err := s.subscriptionService()
	if err != nil {
		return nil, err
	}
	service := modelcatalog.New(s.db, accounts, provider.NewService(s.db), subscription)
	if err := s.wireModelCatalog(service); err != nil {
		return nil, err
	}
	return service, nil
}

func (s *AppService) ModelCatalog() ([]modelcatalog.Item, error) {
	service, err := s.modelCatalog()
	if err != nil {
		return nil, err
	}
	return service.List(context.Background())
}

func (s *AppService) ModelSync(id string) ([]modelcatalog.Item, error) {
	service, err := s.modelCatalog()
	if err != nil {
		return nil, err
	}
	ctx, cancel := subscriptionContext()
	defer cancel()
	items, err := service.Sync(ctx, id)
	if err != nil {
		// Sync 对外只给 unavailable；底层原因从服务里取，日志才有稳定码可看。
		s.logModelSyncError(service.LastSyncError())
		return nil, err
	}
	s.refreshAgentCatalog()
	s.requestModelSync()
	return items, nil
}

// logModelSyncError 记录手动同步失败；目录层已把错误统一成 unavailable，
// 这里尽量取出适配器的稳定码，避免只有一个通用提示。
func (s *AppService) logModelSyncError(err error) {
	if s.log == nil || err == nil {
		return
	}
	code := "model_sync_failed"
	// 适配器与核心边界的错误都带稳定码（workbuddyai.ErrorCode 同时识别 proxy.Error），
	// 取不到时才回退到通用码。
	if stable := workbuddyai.ErrorCode(err); stable != "" {
		code = stable
	}
	s.log.Warn("model catalog sync failed", "module", "modelcatalog", "operation", "manual", "code", code)
}
