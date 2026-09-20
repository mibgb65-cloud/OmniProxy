package proxy

import (
	"net/url"
	"strings"

	"omniproxy/internal/claudedesktop"
	"omniproxy/internal/config"
	"omniproxy/internal/token"
)

type Router struct {
	cfg config.Config
}

type routeInfo struct {
	Provider       string
	CredentialType string
	Protocol       string
	ClientKey      string
	ClientName     string
	Model          string
	Path           string
	RawQuery       string
	Fallbacks      []routeInfo
}

func NewRouter(cfg config.Config) Router {
	return Router{cfg: config.Normalize(cfg)}
}

func (r Router) Route(incoming *url.URL, body []byte) routeInfo {
	path := incoming.Path
	model := requestModel(incoming, body)
	if isAnthropicRouterPath(path) {
		route := r.gatewayRoute(config.GatewayRouteClaude, model)
		return r.gatewayRouteInfo(route, "anthropic", stripPathPrefix(path, "/anthropic-router"), incoming.RawQuery)
	}
	if claudedesktop.IsGatewayPath(path) {
		route := r.gatewayRoute(config.GatewayRouteClaude, model)
		return r.gatewayRouteInfo(route, "anthropic", claudedesktop.StripGatewayPath(path), incoming.RawQuery)
	}
	if isOpenCodeRouterPath(path) {
		route := r.gatewayRoute(config.GatewayRouteOpenAI, model)
		return r.gatewayRouteInfo(route, "openai", stripPathPrefix(path, "/opencode-router"), incoming.RawQuery)
	}
	if isPiRouterPath(path) {
		route := r.gatewayRoute(config.GatewayRouteOpenAI, model)
		return r.gatewayRouteInfo(route, "openai", stripPathPrefix(path, "/pi-router"), incoming.RawQuery)
	}
	if isCodexProxyPath(path) {
		route := r.gatewayRoute(config.GatewayRouteCodex, model)
		return r.gatewayRouteInfo(route, "openai", stripPathPrefix(path, "/backend-api/codex"), incoming.RawQuery)
	}
	if isCodexGatewayPath(path) {
		route := r.gatewayRoute(config.GatewayRouteCodex, model)
		return r.gatewayRouteInfo(route, "openai", stripPathPrefix(path, "/codex"), incoming.RawQuery)
	}
	if isGeminiGatewayPath(path) {
		route := r.gatewayRoute(config.GatewayRouteGemini, model)
		return r.gatewayRouteInfo(route, "gemini", stripPathPrefix(path, "/gemini"), incoming.RawQuery)
	}

	defaultRoute := r.gatewayRoute(config.GatewayRouteOpenAI, model)
	provider := defaultRoute.Provider
	credentialType := defaultRoute.CredentialType
	model = defaultRoute.Model
	directProvider := false

	trimmed := strings.TrimPrefix(path, "/")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) > 0 {
		candidate := strings.ToLower(parts[0])
		switch candidate {
		case token.ProviderOpenAI, token.ProviderAnthropic, token.ProviderDeepSeek, token.ProviderKimi, token.ProviderXiaomi, token.ProviderZhipu, token.ProviderMiniMax, token.ProviderGemini, token.ProviderOpenRouter, token.ProviderFeatherless, token.ProviderTokenRouter, token.ProviderSub2API, token.ProviderNewAPI, token.ProviderAnyRouter, token.ProviderZo, token.ProviderPrem, token.ProviderAtria, token.ProviderCustom:
			directProvider = true
			provider = candidate
			credentialType = ""
			if len(parts) == 2 {
				path = "/" + parts[1]
			} else {
				path = "/"
			}
		}
	}
	if !directProvider {
		return r.gatewayRouteInfo(defaultRoute, "openai", path, incoming.RawQuery)
	}
	protocol := protocolForRoute(provider, &path)
	if provider == token.ProviderPrem {
		path = premProxyPath(protocol, path)
	} else if gatewayProviderUsesProtocolPrefixes(provider) && protocol == "openai" {
		path = versionedGatewayOpenAIPath(path)
	}
	return routeInfo{Provider: provider, CredentialType: credentialType, Protocol: protocol, Model: model, Path: path, RawQuery: incoming.RawQuery}
}

func (r Router) gatewayRoute(name string, requestModel string) config.GatewayRouteConfig {
	routes := config.Normalize(r.cfg).GatewayRoutes
	var route config.GatewayRouteConfig
	switch name {
	case config.GatewayRouteCodex:
		route = routes.Codex
	case config.GatewayRouteClaude:
		route = routes.Claude
	case config.GatewayRouteOpenAI:
		route = routes.OpenAI
	case config.GatewayRouteGemini:
		route = routes.Gemini
	default:
		route = routes.OpenAI
	}
	model := strings.TrimSpace(requestModel)
	if model == "" {
		model = route.Model
	}
	if model != "" {
		if modelRoute, ok := r.modelRoute(model); ok {
			return modelRoute
		}
		model = normalizeUpstreamModelID(model)
		if modelRoute, ok := r.modelRoute(model); ok {
			return modelRoute
		}
		route.Model = model
		for i := range route.Fallbacks {
			route.Fallbacks[i].Model = route.Model
		}
	}
	return inferGatewayRouteProvider(name, route)
}

func (r Router) modelRoute(model string) (config.GatewayRouteConfig, bool) {
	routes := config.Normalize(r.cfg).ModelRoutes
	if len(routes) == 0 {
		return config.GatewayRouteConfig{}, false
	}
	route, ok := routes[modelRouteKey(model)]
	if !ok {
		return config.GatewayRouteConfig{}, false
	}
	return route, true
}

func inferGatewayRouteProvider(name string, route config.GatewayRouteConfig) config.GatewayRouteConfig {
	model := strings.TrimSpace(route.Model)
	if model == "" {
		return route
	}

	current := token.NormalizeProvider(route.Provider)
	inferred := ""
	switch name {
	case config.GatewayRouteClaude:
		inferred = providerForModel(model)
		if inferred == token.ProviderAnthropic && current != token.ProviderAnthropic {
			return route
		}
		if !isDirectClaudeProvider(current) || !isDirectClaudeProvider(inferred) {
			return route
		}
	case config.GatewayRouteCodex, config.GatewayRouteOpenAI:
		inferred = providerForOpenCodeModel(model)
		if inferred == token.ProviderOpenAI && current != token.ProviderOpenAI {
			return route
		}
		if !isDirectOpenAIProvider(current) || !isDirectOpenAIProvider(inferred) {
			return route
		}
	default:
		return route
	}
	if inferred == "" || inferred == current {
		return route
	}
	route.Provider = inferred
	route.CredentialType = ""
	return route
}

func isDirectClaudeProvider(provider string) bool {
	switch token.NormalizeProvider(provider) {
	case token.ProviderAnthropic,
		token.ProviderDeepSeek,
		token.ProviderKimi,
		token.ProviderXiaomi,
		token.ProviderZhipu,
		token.ProviderMiniMax,
		token.ProviderZo,
		token.ProviderAtria:
		return true
	default:
		return false
	}
}

func isDirectOpenAIProvider(provider string) bool {
	switch token.NormalizeProvider(provider) {
	case token.ProviderOpenAI,
		token.ProviderDeepSeek,
		token.ProviderKimi,
		token.ProviderXiaomi,
		token.ProviderZhipu,
		token.ProviderMiniMax,
		token.ProviderOpenRouter,
		token.ProviderTokenRouter,
		token.ProviderZo,
		token.ProviderAtria,
		token.ProviderCustom:
		return true
	default:
		return false
	}
}

func (r Router) gatewayRouteInfo(route config.GatewayRouteConfig, protocol string, path string, rawQuery string) routeInfo {
	primary := gatewayBackendRoute(route, protocol, path, rawQuery)
	for _, fallback := range route.Fallbacks {
		primary.Fallbacks = append(primary.Fallbacks, gatewayBackendRoute(fallback, protocol, path, rawQuery))
	}
	return primary
}

func gatewayBackendRoute(route config.GatewayRouteConfig, protocol string, path string, rawQuery string) routeInfo {
	provider := token.NormalizeProvider(route.Provider)
	return routeInfo{
		Provider:       provider,
		CredentialType: route.CredentialType,
		Protocol:       protocol,
		Model:          route.Model,
		Path:           gatewayBackendPath(provider, protocol, path),
		RawQuery:       rawQuery,
	}
}

func gatewayBackendPath(provider string, protocol string, path string) string {
	if provider == token.ProviderPrem {
		return premProxyPath(protocol, path)
	}
	if gatewayProviderUsesProtocolPrefixes(provider) && protocol == "openai" {
		return versionedGatewayOpenAIPath(path)
	}
	return path
}
