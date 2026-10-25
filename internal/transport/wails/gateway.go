package wails

import (
	"context"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/application/agentconfig"
	"omniproxy/internal/application/gateway"
	"omniproxy/internal/application/googlequota"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/application/quotaactivation"
	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/codex"
	"omniproxy/internal/infrastructure/credential"
)

func (s *AppService) initGateway() {
	if s.prof == nil || s.db == nil || s.prof.ProxyPort < 1 || s.prof.ProxyPort > 65535 {
		return
	}
	store, err := credential.New(s.prof)
	if err != nil {
		return
	}
	paths, err := agentconfig.DefaultPaths()
	if err != nil {
		return
	}
	s.activity = activity.New(s.db)
	if err := s.activity.Recover(context.Background()); err != nil && s.log != nil {
		s.log.Error("request recovery unavailable", "module", "gateway", "code", "accounting_unavailable")
	}
	s.gateway = gateway.New(proxy.Options{Port: s.prof.ProxyPort, Logger: s.log, Observer: s.activity, FailoverEnabled: gateway.NewFailoverSettingsService(s.db).Enabled, WebSocketAllowed: gateway.NewCodexSettingsService(s.db).WebSocketAllowed, PrefixAccess: func(ctx context.Context, prefix string) (string, bool) {
		if s.agents == nil {
			return "", false
		}
		return s.agents.PrefixAccess(ctx, prefix)
	}, Authorize: func(ctx context.Context, key string) (string, bool) {
		if s.agents == nil {
			return "", false
		}
		return s.agents.Client(ctx, key)
	}}, provider.NewService(s.db), account.NewService(s.db, store), nil)
	s.gateway.SetAccountRotationMode(gateway.NewCodexSettingsService(s.db).AccountRotationMode)
	s.gateway.SetCodexUpstreamBase(gateway.NewCodexSettingsService(s.db).UpstreamBaseURL)
	s.gateway.SetRiverAccountRotationMode(gateway.NewRiverSettingsService(s.db).AccountRotationMode)
	s.agents = agentconfig.New(s.db, store, paths, s.prof.ProxyPort, s.gateway)
	// v1 语义：一键配置时把池账号写进客户端登录，网关拥有登录而不是借用户登录。
	s.agents.SetCodexLogin(s.codexLoginSource())
	versionContext, versionCancel := context.WithTimeout(context.Background(), 3*time.Second)
	version, versionError := codex.InstalledVersion(versionContext)
	versionCancel()
	if versionError == nil {
		s.agents.SetCodexVersion(version)
	} else if s.log != nil {
		s.log.Warn("client catalog version unavailable", "module", "agentconfig", "code", "codex_unavailable")
	}
	s.gateway.SetRouter(routes.New(s.db, provider.NewService(s.db), account.NewService(s.db, store)))
	s.gateway.SetGoogleResources(googlequota.New(s.db))
	google, googleError := s.googleSubscriptionService()
	if googleError == nil {
		s.gateway.SetGoogleSubscription(google)
	} else if s.log != nil {
		s.log.Warn("Google subscription unavailable", "module", "gateway", "code", "google_subscription_unavailable")
	}
	subscription, err := s.subscriptionService()
	if err == nil {
		s.gateway.SetSubscription(subscription)
		accounts := account.NewService(s.db, store)
		providers := provider.NewService(s.db)
		s.quotaActivation = quotaactivation.New(s.db, accounts, providers, subscription, quotaactivation.NewMessageSender(accounts, providers, subscription, nil))
		s.quotaActivation.SetLogger(s.log)
		s.quotaActivation.OnChange(func() {
			s.subscription.mu.Lock()
			emit := s.subscription.emitQuota
			s.subscription.mu.Unlock()
			if emit != nil {
				emit()
			}
		})
		s.quotaActivation.OnStatus(s.notifyActivation)
	}
}

// GatewayInfo 返回真实运行状态和本版支持的协议，不读取密钥或调用模型。
func (s *AppService) GatewayInfo() (gateway.Info, error) {
	if s.gateway == nil {
		return gateway.Info{}, proxy.Failure("invalid_configuration", 503)
	}
	return s.gateway.Info(), nil
}

// GatewayStart 仅启动 loopback 监听器。
func (s *AppService) GatewayStart() (gateway.Info, error) {
	if s.gateway == nil {
		return gateway.Info{}, proxy.Failure("invalid_configuration", 503)
	}
	return s.gateway.Start()
}

// GatewayStop 取消当前请求并在有限时间内停止代理。
func (s *AppService) GatewayStop() (gateway.Info, error) {
	if s.gateway == nil {
		return gateway.Info{}, proxy.Failure("invalid_configuration", 503)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.gateway.Stop(ctx)
}

func (s *AppService) shutdownGateway(ctx context.Context) error {
	if s.gateway == nil {
		return nil
	}
	_, err := s.gateway.Stop(ctx)
	if err != nil && s.log != nil {
		s.log.Warn("gateway shutdown failed", "code", "stop_timeout")
	}
	return err
}
