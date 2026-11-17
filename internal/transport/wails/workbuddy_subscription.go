package wails

import (
	"context"
	"net/url"
	"strings"
	"sync"

	"omniproxy/internal/application/modelcatalog"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/application/workbuddy"
	"omniproxy/internal/provider/workbuddyai"
)

type workbuddySubscriptionState struct {
	mu         sync.Mutex
	service    *workbuddy.Service
	auto       *modelcatalog.WorkBuddyAutoSync
	emitModels func()
	closed     bool
}

func (s *AppService) workbuddySubscriptionService() (*workbuddy.Service, error) {
	s.workbuddyAuth.mu.Lock()
	defer s.workbuddyAuth.mu.Unlock()
	if s.workbuddyAuth.closed {
		return nil, workbuddyai.Failure("cancelled")
	}
	if s.workbuddyAuth.service == nil {
		accounts, err := s.accountService()
		if err != nil {
			return nil, workbuddyai.Failure("credential_unavailable")
		}
		manager := workbuddy.New(accounts, nil)
		// 登录在后台线程结束，失败只能通过这里留下诊断：只记阶段与稳定码。
		manager.SetWarn(func(message string, args ...any) {
			if s.log != nil {
				s.log.Warn(message, args...)
			}
		})
		manager.OnChange(func() {
			s.subscription.mu.Lock()
			emit := s.subscription.emit
			s.subscription.mu.Unlock()
			if emit != nil {
				emit()
			}
			s.requestWorkBuddyModelSync()
		})
		s.workbuddyAuth.service = manager
	}
	return s.workbuddyAuth.service, nil
}

func (s *AppService) workbuddyAccount(id string) bool {
	accounts, err := s.accountService()
	if err != nil {
		return false
	}
	item, err := accounts.Metadata(context.Background(), id)
	return err == nil && accounts.CheckWorkBuddyProvider(context.Background(), item.ProviderID) == nil
}

// openWorkBuddySubscription 只打开官方 HTTPS 登录页，不把 URL 交给前端或日志。
func (s *AppService) openWorkBuddySubscription(id string) error {
	manager, err := s.workbuddySubscriptionService()
	if err != nil {
		return err
	}
	view, err := manager.Read(id)
	u, parseErr := url.Parse(view.URL)
	if err != nil || view.State != "waiting" || parseErr != nil || u.Scheme != "https" || u.Host != "www.workbuddy.ai" || u.User != nil || u.Fragment != "" {
		return workbuddyai.Failure("invalid_input")
	}
	s.subscription.mu.Lock()
	open := s.subscription.openURL
	s.subscription.mu.Unlock()
	if open == nil || open(view.URL) != nil {
		return workbuddyai.Failure("browser_failed")
	}
	return nil
}

func workbuddySession(id string) bool { return strings.HasPrefix(id, "workbuddy_") }

// workbuddyProvider 判断内置服务商 ID 是否属于 WorkBuddy 的两个构建（国际版／国内版）。
func workbuddyProvider(providerID string) bool {
	return providerID == "builtin_workbuddy_ai" || providerID == "builtin_workbuddy"
}

// WorkBuddyQuota 返回官方 Credits 三窗口；失败时回退保留期内的最近快照。
func (s *AppService) WorkBuddyQuota(id string) (workbuddy.Quota, error) {
	manager, err := s.workbuddySubscriptionService()
	if err != nil {
		return workbuddy.Quota{}, workbuddyFailure(err)
	}
	ctx, cancel := subscriptionContext()
	defer cancel()
	quota, err := manager.ReadQuota(ctx, id)
	if err != nil {
		return workbuddy.Quota{}, workbuddyFailure(err)
	}
	return quota, nil
}

// workbuddyFailure 把跨层错误归一成 `workbuddy_ai: <code>`。
//
// 核心边界的 proxy.Error 文本是 `gateway: <code>`、账号层是 `account: ...`，
// 直接透传到前端只会被识别成通用失败；这里只保留稳定码。
func workbuddyFailure(err error) error {
	if err == nil {
		return nil
	}
	if code := workbuddyai.ErrorCode(err); code != "" {
		return workbuddyai.Failure(code)
	}
	return err
}

// WorkBuddyCachedQuota 只返回本机保留的公开额度快照，不调用上游。
func (s *AppService) WorkBuddyCachedQuota(id string) (*workbuddy.Quota, error) {
	manager, err := s.workbuddySubscriptionService()
	if err != nil {
		return nil, err
	}
	quota, ok := manager.CachedQuota(id)
	if !ok {
		return nil, nil
	}
	return &quota, nil
}

func (s *AppService) startWorkBuddyModelSync() {
	accounts, err := s.accountService()
	if err != nil {
		s.logWorkBuddyModelSyncError(err)
		return
	}
	source := modelcatalog.New(s.db, accounts, provider.NewService(s.db), nil)
	// 与手动同步共用同一套接线（Google／WorkBuddy），避免只接一路。
	if err := s.wireModelCatalog(source); err != nil {
		s.logWorkBuddyModelSyncError(err)
		return
	}
	s.workbuddyAuth.mu.Lock()
	defer s.workbuddyAuth.mu.Unlock()
	if s.workbuddyAuth.closed || s.workbuddyAuth.auto != nil {
		return
	}
	s.workbuddyAuth.auto = modelcatalog.StartWorkBuddyAutoSync(context.Background(), source, func(ctx context.Context) error {
		if s.agents != nil {
			return s.agents.RefreshCatalog(ctx)
		}
		return nil
	}, func(err error) {
		if err != nil {
			s.logWorkBuddyModelSyncError(err)
		}
		s.workbuddyAuth.mu.Lock()
		emit := s.workbuddyAuth.emitModels
		s.workbuddyAuth.mu.Unlock()
		if emit != nil {
			emit()
		}
	})
}

func (s *AppService) requestWorkBuddyModelSync() {
	s.workbuddyAuth.mu.Lock()
	auto := s.workbuddyAuth.auto
	s.workbuddyAuth.mu.Unlock()
	if auto != nil {
		auto.Request()
	}
}

func (s *AppService) shutdownWorkBuddySubscription() {
	s.workbuddyAuth.mu.Lock()
	s.workbuddyAuth.closed = true
	manager := s.workbuddyAuth.service
	auto := s.workbuddyAuth.auto
	s.workbuddyAuth.mu.Unlock()
	if manager != nil {
		manager.Close()
	}
	if auto != nil {
		auto.Close()
	}
}

func (s *AppService) logWorkBuddyModelSyncError(err error) {
	if s.log != nil {
		// 上游与本地配置错误可能含身份；只记录稳定错误码，旧快照继续可用。
		code := "workbuddy_auto_sync_failed"
		if stable := workbuddyai.ErrorCode(err); stable != "" {
			code = stable
		}
		s.log.Warn("WorkBuddy AI model auto sync incomplete; retry is available", "module", "modelcatalog", "code", code)
	}
}
