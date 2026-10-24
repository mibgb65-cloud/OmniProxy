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
	// selection 记录客户端选中的目录条目，用于把固定档位对应回用户选择的那一行。
	selection := ""
	if request.Protocol == "messages" && (strings.HasPrefix(request.Model, catalog.DesktopAliasPrefix) || strings.HasPrefix(request.Model, catalog.ClaudeEffortPrefix) || strings.HasSuffix(request.Model, catalog.ClaudeLongContextSuffix)) {
		entries, err := s.Catalog(ctx)
		if err != nil {
			return request, proxy.Failure("model_catalog_unavailable", 503)
		}
		// 推理强度来自客户端原生 effort 控件（output_config.effort），模型 ID 只承载模型本身；
		// 历史 omniproxy-effort-* 选择仍可解码，保证已保存的旧选择继续可用。
		effort := claudeRequestedEffort(request.Body)
		model, effort, found := catalog.ResolveClaudeSelection(entries, request.Model, effort)
		if !found {
			if source, ok := s.router.(interface {
				CompatibilityCatalog(context.Context) ([]catalog.Entry, error)
			}); ok {
				legacy, err := source.CompatibilityCatalog(ctx)
				if err != nil {
					return request, proxy.Failure("model_catalog_unavailable", 503)
				}
				model, effort, found = catalog.ResolveClaudeSelection(legacy, request.Model, effort)
			}
		}
		if !found {
			return request, proxy.Failure("unsupported_model", 400)
		}
		body, err := core.RewriteModel(request.Body, model)
		if err != nil {
			return request, proxy.Failure("invalid_request", 400)
		}
		if effort != "" {
			body, err = withClaudeEffort(body, effort)
			if err != nil {
				return request, err
			}
		}
		request.Body, request.Model = body, model
		selection = model
	}
	if s.router == nil {
		return request, nil
	}
	decision, candidates, fixed, err := s.resolvePreview(ctx, request.Model)
	if err != nil {
		return request, proxy.Failure("routing_configuration_unavailable", 503)
	}
	if !decision.Routable && decision.ErrorCode == "unsupported_model" && !decision.Matched {
		// 目录折叠后旧的档位变体 ID 不再单独发布；用户已保存的选择仍必须可用。
		updated, applied, legacyErr := s.applyLegacyVariant(ctx, request)
		if legacyErr != nil {
			return request, legacyErr
		}
		if applied {
			request = updated
			decision, candidates, fixed, err = s.resolvePreview(ctx, request.Model)
			if err != nil {
				return request, proxy.Failure("routing_configuration_unavailable", 503)
			}
		}
	}
	request.Candidates = candidates
	request.FixedAccount = fixed
	if !decision.Routable {
		if decision.ErrorCode == "unsupported_model" && !decision.Matched {
			return request, proxy.Failure("unsupported_model", 400)
		}
		return request, proxy.Failure("route_"+decision.ErrorCode, 503)
	}
	if decision.Matched || decision.Model != request.Model {
		body, err := core.RewriteModel(request.Body, decision.Model)
		if err != nil {
			return request, proxy.Failure("invalid_request", 400)
		}
		request.Body, request.Model = body, decision.Model
	}
	request.ProviderID, request.AccountID = decision.ProviderID, decision.AccountID
	request.AccountUpdatedAt = decision.AccountUpdatedAt
	// 模型固定档位是用户在应用内的明确选择，覆盖客户端自带或沿用的档位。
	if request.Protocol == "messages" {
		pins := s.effortPins(ctx)
		pinned := pins[request.Model]
		if pinned == "" && selection != "" && selection != request.Model {
			pinned = pins[selection]
		}
		if pinned != "" {
			body, err := withClaudeEffort(request.Body, pinned)
			if err != nil {
				return request, err
			}
			request.Body = body
		}
	}
	// 档位变体（Google 的 “…-low” 等）在目录里已折叠成一行模型；请求时必须换回
	// 实际存在的上游型号，适配器才不会收到它不认识的档位字段。
	request, err = s.applyEffortVariant(ctx, request)
	if err != nil {
		return request, err
	}
	return request, nil
}

// resolvePreview 只解析一次路由决策；候选账号每次重新生成，重试时不叠加。
func (s *Service) resolvePreview(ctx context.Context, model string) (routes.Preview, []proxy.AccountCandidate, bool, error) {
	if source, ok := s.router.(interface {
		ResolvePlan(context.Context, string) (routes.Plan, error)
	}); ok {
		plan, err := source.ResolvePlan(ctx, model)
		if err != nil {
			return routes.Preview{}, nil, false, err
		}
		candidates := make([]proxy.AccountCandidate, 0, len(plan.Candidates))
		for _, candidate := range plan.Candidates {
			candidates = append(candidates, proxy.AccountCandidate{AccountID: candidate.AccountID, UpdatedAt: candidate.AccountUpdatedAt})
		}
		return plan.Decision, candidates, plan.Fixed, nil
	}
	decision, err := s.router.Resolve(ctx, model)
	if err != nil {
		return routes.Preview{}, nil, false, err
	}
	return decision, nil, true, nil
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
		if provider.ID != request.ProviderID || (provider.Kind != "deepseek" && provider.Kind != "openai" && provider.Kind != "atria" && provider.Kind != "river" && provider.Kind != "google" && provider.Kind != "mimo" && provider.Kind != "stepfun" && provider.Kind != "kimi" && provider.Kind != "minimax") || !provider.Enabled {
			continue
		}
		for _, account := range accounts {
			if account.ID == request.AccountID && account.ProviderID == provider.ID && account.Enabled {
				if request.AccountUpdatedAt != "" && account.UpdatedAt != request.AccountUpdatedAt {
					return target{}, proxy.Failure("route_configuration_changed", 503)
				}
				if provider.Kind == "mimo" && (account.MiMo == nil || !account.MiMo.Valid()) {
					return target{}, proxy.Failure("route_account_unavailable", 503)
				}
				if provider.Kind == "minimax" && (account.AuthMethod != "api_key" || account.MiniMax == nil || !account.MiniMax.Valid()) {
					return target{}, proxy.Failure("route_account_unavailable", 503)
				}
				if provider.Kind == "kimi" && (account.AuthMethod != "api_key" || account.Kimi == nil || !account.Kimi.Valid()) {
					return target{}, proxy.Failure("route_account_unavailable", 503)
				}
				if provider.Kind == "stepfun" && (account.AuthMethod != "api_key" || account.StepFun == nil || !account.StepFun.Valid()) {
					return target{}, proxy.Failure("route_account_unavailable", 503)
				}
				if (provider.Kind == "atria" || provider.Kind == "mimo" || provider.Kind == "river") && account.AuthMethod != "api_key" {
					return target{}, proxy.Failure("route_account_unavailable", 503)
				}
				if provider.Kind == "google" && account.AuthMethod != "api_key" && !(account.AuthMethod == "subscription_login" && account.AuthState == "authenticated") {
					return target{}, proxy.Failure("route_account_unavailable", 503)
				}
				// 订阅账号有两种凭据来源：官方隔离登录与网关自己保管的 auth.json。
				if provider.Kind == "openai" && (!codexLoginMethod(account.AuthMethod) || account.AuthState != "authenticated") {
					return target{}, proxy.Failure("route_account_unavailable", 503)
				}
				return target{authMethod: account.AuthMethod, accountID: account.ID, updatedAt: account.UpdatedAt, baseURL: provider.BaseURL, kind: provider.Kind, snapshot: snapshotTarget(provider, account), stepfun: account.StepFun, kimi: account.Kimi, minimax: account.MiniMax, mimo: account.MiMo}, nil
			}
		}
		return target{}, proxy.Failure("route_account_unavailable", 503)
	}
	return target{}, proxy.Failure("route_provider_unavailable", 503)
}
