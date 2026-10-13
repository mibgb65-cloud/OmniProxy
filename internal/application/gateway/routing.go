package gateway

import (
	"context"
	catalog "omniproxy/internal/core/modelcatalog"
	"strings"

	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/core/proxy"
	core "omniproxy/internal/core/routing"
	"omniproxy/internal/provider/deepseek"
)

// Router 只解析配置，不读取密钥；安装后规则变更在每次请求时即时生效。
type Router interface {
	Resolve(context.Context, string) (routes.Preview, error)
	Aliases(context.Context) ([]string, error)
}

// SetRouter 仅在启动前安装服务，避免运行中变更依赖对象。
func (s *Service) SetRouter(router Router) { s.router = router }

// RouteRequest 解析规则后改写模型，路由异常不回退到其他提供商或账号。
func (s *Service) RouteRequest(ctx context.Context, request proxy.Request) (proxy.Request, error) {
	if request.Protocol == "messages" && (strings.HasPrefix(request.Model, catalog.DesktopAliasPrefix) || strings.HasSuffix(request.Model, catalog.ClaudeLongContextSuffix)) {
		entries, err := s.Catalog(ctx)
		if err != nil {
			return request, proxy.Failure("model_catalog_unavailable", 503)
		}
		model, found := catalog.ResolveClaudeModel(entries, request.Model)
		if !found {
			return request, proxy.Failure("unsupported_model", 400)
		}
		body, err := core.RewriteModel(request.Body, model)
		if err != nil {
			return request, proxy.Failure("invalid_request", 400)
		}
		request.Body, request.Model = body, model
	}
	if s.router == nil {
		return request, nil
	}
	decision, err := s.router.Resolve(ctx, request.Model)
	if err != nil {
		return request, proxy.Failure("routing_configuration_unavailable", 503)
	}
	if !decision.Routable {
		if decision.ErrorCode == "unsupported_model" && !decision.Matched {
			return request, proxy.Failure("unsupported_model", 400)
		}
		return request, proxy.Failure("route_"+decision.ErrorCode, 503)
	}
	if decision.Matched {
		body, err := core.RewriteModel(request.Body, decision.Model)
		if err != nil {
			return request, proxy.Failure("invalid_request", 400)
		}
		request.Body, request.Model = body, decision.Model
	}
	request.ProviderID, request.AccountID = decision.ProviderID, decision.AccountID
	return request, nil
}

// AvailableModels 合并原始模型和可用别名，保留已有发现协议并去重。
func (s *Service) AvailableModels(ctx context.Context) ([]string, error) {
	if source, ok := s.router.(interface {
		Catalog(context.Context) ([]catalog.Entry, error)
	}); ok {
		entries, err := source.Catalog(ctx)
		if err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(entries))
		for _, entry := range entries {
			ids = append(ids, entry.ID)
		}
		return ids, nil
	}
	models := deepseek.Models()
	if catalog, ok := s.router.(interface {
		Models(context.Context) ([]string, error)
	}); ok {
		var err error
		models, err = catalog.Models(ctx)
		if err != nil {
			return nil, err
		}
	}
	if s.router == nil {
		return models, nil
	}
	aliases, err := s.router.Aliases(ctx)
	if err != nil {
		return nil, proxy.Failure("routing_configuration_unavailable", 503)
	}
	seen := make(map[string]bool)
	for _, model := range models {
		seen[model] = true
	}
	for _, alias := range aliases {
		if !seen[alias] {
			models = append(models, alias)
			seen[alias] = true
		}
	}
	return models, nil
}

func (s *Service) selectRoutedTarget(ctx context.Context, request proxy.Request) (target, error) {
	providers, err := s.providers.List(ctx)
	if err != nil {
		return target{}, proxy.Failure("configuration_unavailable", 503)
	}
	accounts, err := s.accounts.List(ctx)
	if err != nil {
		return target{}, proxy.Failure("configuration_unavailable", 503)
	}
	for _, provider := range providers {
		if provider.ID != request.ProviderID || (provider.Kind != "deepseek" && provider.Kind != "openai") || !provider.Enabled {
			continue
		}
		for _, account := range accounts {
			if account.ID == request.AccountID && account.ProviderID == provider.ID && account.Enabled {
				if provider.Kind == "openai" && (account.AuthMethod != "subscription_login" || account.AuthState != "authenticated") {
					return target{}, proxy.Failure("route_account_unavailable", 503)
				}
				return target{accountID: account.ID, baseURL: provider.BaseURL, kind: provider.Kind, snapshot: snapshotTarget(provider, account)}, nil
			}
		}
		return target{}, proxy.Failure("route_account_unavailable", 503)
	}
	return target{}, proxy.Failure("route_provider_unavailable", 503)
}
