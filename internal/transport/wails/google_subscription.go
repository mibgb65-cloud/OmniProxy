package wails

import (
	"context"
	"net/url"
	"strings"
	"sync"

	"omniproxy/internal/application/googlesubscription"
	"omniproxy/internal/provider/antigravity"
)

type googleSubscriptionState struct {
	mu      sync.Mutex
	service *googlesubscription.Service
	closed  bool
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
		manager.OnChange(func() {
			s.subscription.mu.Lock()
			emit := s.subscription.emit
			s.subscription.mu.Unlock()
			if emit != nil {
				emit()
			}
			s.refreshAgentCatalog()
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

func (s *AppService) shutdownGoogleSubscription() {
	s.googleAuth.mu.Lock()
	s.googleAuth.closed = true
	manager := s.googleAuth.service
	s.googleAuth.mu.Unlock()
	if manager != nil {
		manager.Close()
	}
}
