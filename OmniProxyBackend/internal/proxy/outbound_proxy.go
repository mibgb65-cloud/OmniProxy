package proxy

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"omniproxy/internal/config"
	"omniproxy/internal/token"
)

func outboundProxyURL(cfg config.Config) (*url.URL, error) {
	cfg = config.Normalize(cfg)
	if !cfg.OutboundProxyEnabled {
		return nil, nil
	}
	raw := strings.TrimSpace(cfg.OutboundProxyURL)
	if raw == "" {
		return nil, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid outbound proxy url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid outbound proxy url %q", raw)
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "socks5", "socks5h":
		return parsed, nil
	default:
		return nil, fmt.Errorf("unsupported outbound proxy scheme %q", parsed.Scheme)
	}
}

func newHTTPClient(timeout time.Duration, proxyURL *url.URL) *http.Client {
	var transport http.RoundTripper
	if base, ok := http.DefaultTransport.(*http.Transport); ok {
		cloned := base.Clone()
		if proxyURL != nil {
			cloned.Proxy = http.ProxyURL(proxyURL)
		}
		transport = cloned
	} else if proxyURL != nil {
		transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	} else {
		transport = http.DefaultTransport
	}
	if transport == nil {
		transport = &http.Transport{}
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

func NewTokenHTTPClient(cfg config.Config, selected token.Token, timeout time.Duration) (*http.Client, error) {
	cfg = config.Normalize(cfg)
	outboundProxy, err := outboundProxyURL(cfg)
	if err != nil {
		return nil, err
	}
	if outboundProxy != nil && outboundProxyMatchesToken(selected, cfg) {
		return newHTTPClient(timeout, outboundProxy), nil
	}
	return newHTTPClient(timeout, nil), nil
}

func outboundProxyMatchesRoute(route routeInfo, cfg config.Config) bool {
	cfg = config.Normalize(cfg)
	return outboundProxyMatchesProvider(route.Provider, cfg.OutboundProxyProviders)
}

func outboundProxyMatchesToken(selected token.Token, cfg config.Config) bool {
	return outboundProxyMatchesRoute(routeInfo{
		Provider:       selected.Provider,
		CredentialType: selected.CredentialType,
	}, cfg)
}

func outboundProxyMatchesProvider(provider string, providers []string) bool {
	provider = normalizeOutboundProxyProvider(provider)
	if provider == "" {
		return false
	}
	for _, item := range providers {
		if normalizeOutboundProxyProvider(item) == provider {
			return true
		}
	}
	return false
}

func normalizeOutboundProxyProvider(provider string) string {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case token.ProviderOpenAI, "codex":
		return token.ProviderOpenAI
	case token.ProviderAnthropic, "claude":
		return token.ProviderAnthropic
	case token.ProviderDeepSeek:
		return token.ProviderDeepSeek
	case token.ProviderKimi:
		return token.ProviderKimi
	case token.ProviderXiaomi, "mimo":
		return token.ProviderXiaomi
	case token.ProviderZhipu, "glm":
		return token.ProviderZhipu
	case token.ProviderMiniMax:
		return token.ProviderMiniMax
	case token.ProviderGemini, "google":
		return token.ProviderGemini
	case token.ProviderOpenRouter:
		return token.ProviderOpenRouter
	case token.ProviderFeatherless, "featherless-ai", "featherless ai":
		return token.ProviderFeatherless
	case token.ProviderTokenRouter:
		return token.ProviderTokenRouter
	case token.ProviderSub2API:
		return token.ProviderSub2API
	case token.ProviderNewAPI, "new-api", "new api":
		return token.ProviderNewAPI
	case token.ProviderAnyRouter, "any-router", "any router":
		return token.ProviderAnyRouter
	case token.ProviderZo, "zocomputer", "zo-computer":
		return token.ProviderZo
	case token.ProviderPrem, "premai", "prem-ai", "prem ai":
		return token.ProviderPrem
	case token.ProviderAtria, "atria-asi", "atria asi":
		return token.ProviderAtria
	case token.ProviderCustom:
		return token.ProviderCustom
	default:
		return ""
	}
}
