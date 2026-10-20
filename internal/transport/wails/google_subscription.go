package wails

import (
	"context"
	"net/url"
	"strings"
	"sync"

	"omniproxy/internal/application/googlesubscription"
	"omniproxy/internal/application/modelcatalog"
	"omniproxy/internal/provider/antigravity"
)

type googleSubscriptionState struct {
	mu         sync.Mutex
	service    *googlesubscription.Service
	auto       *modelcatalog.GoogleAutoSync
	emitModels func()
	closed     bool
}

func (s *AppService) googleSubscriptionService() (*googlesubscription.Service, error) {
	s.googleAuth.mu.Lock()
	defer s.googleAuth.mu.Unlock()
	if s.googleAuth.closed {
		return nil, antigravity.Failure("cancelled")
	}
	if s.googleAuth.service == nil {
		accounts, err := s.accountService()
		if err != nil {
			return nil, antigravity.Failure("credential_unavailable")
		}
		manager := googlesubscription.New(accounts, nil)
		manager.SetQuotaCache(googlesubscription.NewQuotaCache(s.db))
		manager.OnChange(func() {
			s.subscription.mu.Lock()
			emit := s.subscription.emit
			s.subscription.mu.Unlock()
			if emit != nil {
				emit()
			}
			s.requestGoogleModelSync()
		})
		s.googleAuth.service = manager
	}
	return s.googleAuth.service, nil
}

func (s *AppService) googleAccount(id string) bool {
	accounts, err := s.accountService()
	if err != nil {
		return false
	}
	item, err := accounts.Metadata(context.Background(), id)
	return err == nil && accounts.CheckGoogleProvider(context.Background(), item.ProviderID) == nil
}

func (s *AppService) openGoogleSubscription(id string) error {
	manager, err := s.googleSubscriptionService()
	if err != nil {
		return err
	}
	view, err := manager.Read(id)
	u, parseErr := url.Parse(view.URL)
	if err != nil || view.State != "waiting" || parseErr != nil || u.Scheme != "https" || u.Host != "accounts.google.com" || u.Path != "/o/oauth2/v2/auth" || u.User != nil || u.Fragment != "" {
		return antigravity.Failure("invalid_input")
	}
	s.subscription.mu.Lock()
	open := s.subscription.openURL
	s.subscription.mu.Unlock()
	if open == nil || open(view.URL) != nil {
		return antigravity.Failure("browser_failed")
	}
	return nil
}

func googleSession(id string) bool { return strings.HasPrefix(id, "google_") }

// GoogleSubscriptionQuota 返回官方公开额度窗口，不返回身份、项目或授权凭据。
func (s *AppService) GoogleSubscriptionQuota(id string) (antigravity.Quota, error) {
	manager, err := s.googleSubscriptionService()
	if err != nil {
		return antigravity.Quota{}, err
	}
	ctx, cancel := subscriptionContext()
	defer cancel()
	return manager.ReadQuota(ctx, id)
}

// GoogleSubscriptionCachedQuota 恢复当前身份公开快照，不调用上游。
func (s *AppService) GoogleSubscriptionCachedQuota(id string) (*antigravity.Quota, error) {
	manager, err := s.googleSubscriptionService()
	if err != nil {
		return nil, err
	}
	ctx, cancel := subscriptionContext()
	defer cancel()
	return manager.CachedQuota(ctx, id)
}

// GoogleSubscriptionModels 返回当前账号官方选择器模型与本机适配状态。
func (s *AppService) GoogleSubscriptionModels(id string) ([]antigravity.AccountModel, error) {
	manager, err := s.googleSubscriptionService()
	if err != nil {
		return nil, err
	}
	ctx, cancel := subscriptionContext()
	defer cancel()
	models, err := manager.ReadAccountModels(ctx, id)
	if err == nil {
		s.requestGoogleModelSync()
	}
	return models, err
}

func (s *AppService) shutdownGoogleSubscription() {
	s.googleAuth.mu.Lock()
	s.googleAuth.closed = true
	manager := s.googleAuth.service
	auto := s.googleAuth.auto
	s.googleAuth.mu.Unlock()
	if manager != nil {
		manager.Close()
	}
	if auto != nil {
		auto.Close()
	}
}
