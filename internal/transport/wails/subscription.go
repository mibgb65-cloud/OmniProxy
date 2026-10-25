package wails

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"omniproxy/internal/application/account"
	"omniproxy/internal/application/subscription"
	"omniproxy/internal/infrastructure/codex"
)

type subscriptionState struct {
	mu        sync.Mutex
	service   *subscription.Service
	openURL   func(string) error
	emit      func()
	emitQuota func()
	closed    bool
}

// SubscriptionNotice 只发送状态信号，不携带个人身份或授权 URL。
type SubscriptionNotice struct {
	Changed bool `json:"changed"`
}

func init() { application.RegisterEvent[SubscriptionNotice]("subscription:updated") }

func (s *AppService) subscriptionService() (*subscription.Service, error) {
	s.subscription.mu.Lock()
	defer s.subscription.mu.Unlock()
	if s.subscription.closed {
		return nil, errors.New("subscription: operation_cancelled")
	}
	if s.subscription.service != nil {
		return s.subscription.service, nil
	}
	accounts, err := s.accountService()
	if err != nil {
		return nil, errors.New("subscription: initialization_failed")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return nil, errors.New("subscription: initialization_failed")
	}
	// 系统 keyring 按 CODEX_HOME 定位；不跟随可迁移的数据目录，开发与生产分开。
	root := filepath.Join(cache, "OmniProxy-CodexAuth", string(s.prof.Kind))
	manager := subscription.New(accounts, root, nil)
	manager.SetQuotaCache(subscription.NewQuotaCache(s.db))
	// 额度优先走官方 HTTP 接口；不可用时由服务自身回退现有 stdio 链路。
	manager.SetUsageSource(codex.NewUsageClient(nil))
	manager.SetWarn(func(message string, args ...any) {
		if s.log != nil {
			s.log.Warn(message, args...)
		}
	})
	manager.OnChange(func() {
		diagnostic := manager.DiagnosticState()
		if s.log != nil {
			s.log.Info("subscription authorization state changed", "module", "subscription", "state", diagnostic.State, "code", diagnostic.Code)
		}
		s.subscription.mu.Lock()
		emit := s.subscription.emit
		s.subscription.mu.Unlock()
		if emit != nil {
			emit()
		}
	})
	s.subscription.service = manager
	return manager, nil
}

func subscriptionContext() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// SubscriptionBegin 按提供商启动独立的 Codex 或 Google 浏览器授权。
func (s *AppService) SubscriptionBegin(input subscription.Input) (subscription.State, error) {
	if input.ProviderID == "builtin_google" {
		manager, err := s.googleSubscriptionService()
		if err != nil {
			return subscription.State{}, err
		}
		ctx, cancel := subscriptionContext()
		defer cancel()
		return manager.Begin(ctx, input)
	}
	service, err := s.subscriptionService()
	if err != nil {
		s.logSubscriptionBeginError(err)
		return subscription.State{}, err
	}
	ctx, cancel := subscriptionContext()
	defer cancel()
	view, err := service.Begin(ctx, input)
	s.logSubscriptionBeginError(err)
	return view, err
}

func (s *AppService) logSubscriptionBeginError(err error) {
	if err != nil && s.log != nil {
		s.log.Warn("subscription authorization request failed", "module", "subscription", "operation", "begin", "code", subscription.ErrorCode(err))
	}
}

// SubscriptionRead 读取当前会话，不返回任何令牌。
func (s *AppService) SubscriptionRead(id string) (subscription.State, error) {
	if googleSession(id) {
		manager, err := s.googleSubscriptionService()
		if err != nil {
			return subscription.State{}, err
		}
		return manager.Read(id)
	}
	service, err := s.subscriptionService()
	if err != nil {
		return subscription.State{}, err
	}
	return service.Read(id)
}

// SubscriptionCurrent 恢复仍在运行的后台授权，不启动或刷新登录。
func (s *AppService) SubscriptionCurrent(providerID, accountID string) (subscription.State, error) {
	if providerID == "builtin_google" {
		manager, err := s.googleSubscriptionService()
		if err != nil {
			return subscription.State{}, err
		}
		return manager.Current(providerID, accountID)
	}
	service, err := s.subscriptionService()
	if err != nil {
		return subscription.State{}, err
	}
	if providerID != "builtin_openai" || len(accountID) > 128 {
		return subscription.State{}, errors.New("subscription: invalid_input")
	}
	return service.Current(providerID, accountID)
}

// SubscriptionCancel 取消隔离授权并清理临时系统凭证。
func (s *AppService) SubscriptionCancel(id string) (subscription.State, error) {
	if googleSession(id) {
		manager, err := s.googleSubscriptionService()
		if err != nil {
			return subscription.State{}, err
		}
		return manager.Cancel(id)
	}
	service, err := s.subscriptionService()
	if err != nil {
		return subscription.State{}, err
	}
	return service.Cancel(id)
}

// SubscriptionOpen 只打开由官方组件返回且仍在等待中的授权位置。
func (s *AppService) SubscriptionOpen(id string) error {
	if googleSession(id) {
		return s.openGoogleSubscription(id)
	}
	view, err := s.SubscriptionRead(id)
	if err != nil || view.State != "waiting" || !codex.ValidAuthorizationURL(view.URL) {
		return errors.New("subscription: invalid_session")
	}
	s.subscription.mu.Lock()
	open := s.subscription.openURL
	s.subscription.mu.Unlock()
	if open == nil || open(view.URL) != nil {
		return errors.New("subscription: browser_failed")
	}
	return nil
}

// SubscriptionRefresh 请求官方组件刷新授权并更新账号状态。
func (s *AppService) SubscriptionRefresh(id string) (account.Item, error) {
	if s.googleAccount(id) {
		manager, err := s.googleSubscriptionService()
		if err != nil {
			return account.Item{}, err
		}
		ctx, cancel := subscriptionContext()
		defer cancel()
		return manager.Refresh(ctx, id)
	}
	service, err := s.subscriptionService()
	if err != nil {
		return account.Item{}, err
	}
	ctx, cancel := subscriptionContext()
	defer cancel()
	return service.Refresh(ctx, id)
}

// SubscriptionLogout 清除该账号系统凭证并保留账号元数据。
func (s *AppService) SubscriptionLogout(id string) (account.Item, error) {
	if s.googleAccount(id) {
		manager, err := s.googleSubscriptionService()
		if err != nil {
			return account.Item{}, err
		}
		ctx, cancel := subscriptionContext()
		defer cancel()
		return manager.Logout(ctx, id)
	}
	service, err := s.subscriptionService()
	if err != nil {
		return account.Item{}, err
	}
	ctx, cancel := subscriptionContext()
	defer cancel()
	return service.Logout(ctx, id)
}

func (s *AppService) bindSubscriptionEvents(app *application.App) {
	s.bindModelCatalogEvents(app)
	s.subscription.openURL = app.Browser.OpenURL
	s.subscription.emit = func() { app.Event.Emit("subscription:updated", SubscriptionNotice{Changed: true}) }
	s.subscription.emitQuota = func() { app.Event.Emit("subscription:quota-updated", SubscriptionNotice{Changed: true}) }
}

func (s *AppService) shutdownSubscription() {
	s.subscription.mu.Lock()
	s.subscription.closed = true
	manager := s.subscription.service
	s.subscription.mu.Unlock()
	if manager != nil {
		manager.Close()
	}
}
