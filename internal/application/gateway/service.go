// Package gateway 协调账号凭证、提供商适配器和独立代理核心。
package gateway

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/core/proxy"
	core "omniproxy/internal/core/routing"
	"omniproxy/internal/provider/atria"
	"omniproxy/internal/provider/chatgpt"
	"omniproxy/internal/provider/deepseek"
	"omniproxy/internal/provider/google"
	"omniproxy/internal/provider/kimi"
	"omniproxy/internal/provider/mimo"
	"omniproxy/internal/provider/stepfun"
)

// ProviderReader 只读取已保存的提供商，不修改其他功能的配置。
type ProviderReader interface {
	List(context.Context) ([]provider.Configuration, error)
}

// AccountAccess 只在回调内使用已有凭证，不向调用方返回完整密钥。
type AccountAccess interface {
	List(context.Context) ([]account.Item, error)
	UseAPIKey(context.Context, string, func([]byte) error) error
}

// Info 包含运行快照和本版实际支持的协议。
type Info struct {
	Runtime               proxy.Status `json:"runtime"`
	Models                []string     `json:"models"`
	Protocol              string       `json:"protocol"`
	RequestTimeoutSeconds int          `json:"requestTimeoutSeconds"`
}

// Service 复用账号与提供商服务，不持有或缓存明文密钥。
type Service struct {
	server          *proxy.Server
	providers       ProviderReader
	accounts        AccountAccess
	client          *deepseek.Client
	log             *slog.Logger
	timeout         time.Duration
	router          Router
	subscription    SubscriptionAccess
	chatgpt         *chatgpt.Client
	atria           *atria.Client
	google          *google.Client
	mimo            *mimo.Client
	stepfun         *stepfun.Client
	kimi            *kimi.Client
	googleResources GoogleResourceObserver
}

// New 创建尚未启动的网关；测试可注入本机上游传输与临时数据库。
func New(opts proxy.Options, providers ProviderReader, accounts AccountAccess, client *deepseek.Client) *Service {
	if client == nil {
		client = deepseek.New(nil)
	}
	if opts.RequestTimeout <= 0 {
		opts.RequestTimeout = 120 * time.Second
	}
	s := &Service{providers: providers, accounts: accounts, client: client, chatgpt: chatgpt.New(nil), log: opts.Logger, timeout: opts.RequestTimeout}
	s.atria = atria.New(nil)
	s.google = google.New(nil)
	s.mimo = mimo.New(nil)
	s.stepfun = stepfun.New(nil)
	s.kimi = kimi.New(nil)
	opts.Upstream, opts.Models = s, deepseek.Models()
	s.server = proxy.New(opts)
	return s
}

// Info 只读取内存运行状态，不进行上游请求。
func (s *Service) Info() Info {
	models, err := s.AvailableModels(context.Background())
	if err != nil {
		models = []string{}
		if s.log != nil {
			s.log.Warn("gateway model catalog unavailable", "module", "gateway", "code", "catalog_unavailable")
		}
	}
	return Info{Runtime: s.server.Status(), Models: models, Protocol: "Chat Completions / Responses / Messages", RequestTimeoutSeconds: int(s.timeout / time.Second)}
}

// Start 启动本机服务，配置不齐全时仍可启动并返回明确的请求错误。
func (s *Service) Start() (Info, error) { _, err := s.server.Start(); return s.Info(), err }

// Stop 关闭监听并取消正在进行的上游调用。
func (s *Service) Stop(ctx context.Context) (Info, error) {
	_, err := s.server.Stop(ctx)
	return s.Info(), err
}

// OnStatus 将核心状态变更交给桌面适配层，不让 Core 依赖 Wails。
func (s *Service) OnStatus(listener func(proxy.Status)) { s.server.OnStatus(listener) }

// Forward 每次重新检查启用状态，只选择一个账号；不进行隐式重试或故障切换。
func (s *Service) Forward(ctx context.Context, request proxy.Request, consume func(proxy.Response) error) error {
	var target target
	var err error
	if request.ProviderID != "" {
		target, err = s.selectRoutedTarget(ctx, request)
	} else {
		target, err = s.selectTarget(ctx)
	}
	if err != nil {
		return err
	}
	if err := request.Trace.Update(ctx, request, target.snapshot); err != nil {
		return err
	}
	err = s.forwardAgent(ctx, target, request, consume)
	if err == nil || ctx.Err() != nil {
		return err
	}
	var failure *proxy.Error
	if errors.As(err, &failure) {
		return failure
	}
	return proxy.Failure("credential_unavailable", 503)
}

type target struct {
	stepfun   *stepfun.Options
	kimi      *kimi.Options
	updatedAt string
	snapshot  proxy.TargetSnapshot
	accountID string
	baseURL   string
	kind      string
}

func (s *Service) selectTarget(ctx context.Context) (target, error) {
	if s.providers == nil || s.accounts == nil {
		return target{}, proxy.Failure("invalid_configuration", 503)
	}
	providers, err := s.providers.List(ctx)
	if err != nil {
		return target{}, proxy.Failure("configuration_unavailable", 503)
	}
	accounts, err := s.accounts.List(ctx)
	if err != nil {
		return target{}, proxy.Failure("configuration_unavailable", 503)
	}
	allowed := make(map[string]bool)
	baseURLs := make(map[string]string)
	for _, provider := range providers {
		if provider.Enabled && provider.Kind == "deepseek" {
			allowed[provider.ID] = true
			baseURLs[provider.ID] = provider.BaseURL
		}
	}
	if len(allowed) == 0 {
		return target{}, proxy.Failure("no_enabled_deepseek_provider", 503)
	}
	candidates := make([]core.Account, 0, len(accounts))
	for _, item := range accounts {
		candidates = append(candidates, core.Account{ID: item.ID, ProviderID: item.ProviderID, CreatedAt: item.CreatedAt, Enabled: item.Enabled})
	}
	selected, _, code := core.SelectAccount(candidates, allowed, "")
	if code != "" {
		if code == "configuration_unavailable" {
			return target{}, proxy.Failure(code, 503)
		}
		return target{}, proxy.Failure("no_enabled_deepseek_account", 503)
	}
	result := target{accountID: selected.ID, baseURL: baseURLs[selected.ProviderID]}
	for _, p := range providers {
		for _, a := range accounts {
			if a.ID == selected.ID && p.ID == selected.ProviderID {
				result.snapshot = snapshotTarget(p, a)
			}
		}
	}
	return result, nil
}
