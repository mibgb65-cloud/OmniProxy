package proxy

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"omniproxy/internal/token"
)

func protocolForRoute(provider string, path *string) string {
	protocol := "openai"
	if provider == token.ProviderAnthropic {
		protocol = "anthropic"
	}
	switch provider {
	case token.ProviderGemini:
		protocol = "gemini"
	case token.ProviderSub2API, token.ProviderNewAPI:
		if stripProtocolPrefix(path, "/anthropic") {
			protocol = "anthropic"
		} else if stripProtocolPrefix(path, "/gemini") {
			protocol = "gemini"
		}
	case token.ProviderAnyRouter:
		if stripProtocolPrefix(path, "/anthropic") {
			protocol = "anthropic"
		}
	case token.ProviderPrem:
		if stripProtocolPrefix(path, "/anthropic") {
			protocol = "anthropic"
		} else if stripProtocolPrefix(path, "/openai") {
			protocol = "openai"
		} else if isAnthropicMessagePath(*path) {
			protocol = "anthropic"
		}
	case token.ProviderDeepSeek, token.ProviderKimi, token.ProviderXiaomi, token.ProviderZhipu, token.ProviderMiniMax, token.ProviderZo, token.ProviderCustom:
		if stripProtocolPrefix(path, "/anthropic") {
			protocol = "anthropic"
		}
	}
	if provider == token.ProviderZo && isAnthropicMessagePath(*path) {
		protocol = "anthropic"
	}
	return protocol
}

func isAnthropicMessagePath(path string) bool {
	path = strings.TrimSpace(path)
	return path == "/messages" || path == "/v1/messages" || path == "/v1/v1/messages"
}

func stripProtocolPrefix(path *string, prefix string) bool {
	if *path == prefix {
		*path = "/"
		return true
	}
	withSlash := prefix + "/"
	if strings.HasPrefix(*path, withSlash) {
		*path = "/" + strings.TrimPrefix(*path, withSlash)
		return true
	}
	return false
}

func gatewayProviderUsesProtocolPrefixes(provider string) bool {
	return provider == token.ProviderSub2API || provider == token.ProviderNewAPI || provider == token.ProviderAnyRouter
}

func versionedGatewayOpenAIPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" || path == "/" || path == "/v1" || strings.HasPrefix(path, "/v1/") {
		return path
	}
	return singleJoiningSlash("/v1", path)
}

func premProxyPath(protocol string, path string) string {
	if protocol == "anthropic" {
		return singleJoiningSlash("/anthropic", versionedGatewayOpenAIPath(path))
	}
	return singleJoiningSlash("/openai", versionedGatewayOpenAIPath(path))
}

func isAnthropicRouterPath(path string) bool {
	return path == "/anthropic-router" || strings.HasPrefix(path, "/anthropic-router/")
}

func isOpenCodeRouterPath(path string) bool {
	return path == "/opencode-router" || strings.HasPrefix(path, "/opencode-router/")
}

func isPiRouterPath(path string) bool {
	return path == "/pi-router" || strings.HasPrefix(path, "/pi-router/")
}

func isGeminiGatewayPath(path string) bool {
	return path == "/gemini" || strings.HasPrefix(path, "/gemini/")
}

func isCodexGatewayPath(path string) bool {
	return path == "/codex" || strings.HasPrefix(path, "/codex/")
}

func isAnthropicRouterProbe(r *http.Request) bool {
	if r.URL == nil || r.URL.Path != "/anthropic-router" {
		return false
	}
	return r.Method == http.MethodHead || r.Method == http.MethodGet
}

func isOpenCodeRouterProbe(r *http.Request) bool {
	if r.URL == nil || r.URL.Path != "/opencode-router" {
		return false
	}
	return r.Method == http.MethodHead || r.Method == http.MethodGet
}

func isPiRouterProbe(r *http.Request) bool {
	if r.URL == nil || r.URL.Path != "/pi-router" {
		return false
	}
	return r.Method == http.MethodHead || r.Method == http.MethodGet
}

func isCodexResponsesProbe(r *http.Request) bool {
	if r.URL == nil || (r.Method != http.MethodHead && r.Method != http.MethodGet) {
		return false
	}
	if isWebSocketUpgrade(r) {
		return false
	}
	path := stripPathPrefix(r.URL.Path, "/backend-api/codex")
	path = stripPathPrefix(path, "/codex")
	path = stripPathPrefix(path, "/v1")
	return path == "/responses"
}

func isCodexWebSocket(r *http.Request) bool {
	if r.URL == nil || r.Method != http.MethodGet || !isWebSocketUpgrade(r) {
		return false
	}
	path := stripPathPrefix(r.URL.Path, "/backend-api/codex")
	path = stripPathPrefix(path, "/codex")
	path = stripPathPrefix(path, "/v1")
	return path == "/responses" || path == "/live"
}

func isWebSocketUpgrade(r *http.Request) bool {
	return tokenListContains(r.Header.Get("Connection"), "upgrade") &&
		strings.EqualFold(strings.TrimSpace(r.Header.Get("Upgrade")), "websocket")
}

func requestModel(incoming *url.URL, body []byte) string {
	var payload struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(body, &payload); err == nil {
		if model := strings.TrimSpace(payload.Model); model != "" {
			return model
		}
	}
	if incoming == nil {
		return ""
	}
	if model := strings.TrimSpace(incoming.Query().Get("model")); model != "" {
		return model
	}
	return requestModelFromPath(incoming.Path)
}

func requestModelFromPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	const marker = "/models/"
	index := strings.Index(path, marker)
	if index < 0 {
		return ""
	}
	model := path[index+len(marker):]
	if slash := strings.Index(model, "/"); slash >= 0 {
		model = model[:slash]
	}
	if colon := strings.Index(model, ":"); colon >= 0 {
		model = model[:colon]
	}
	return strings.TrimSpace(model)
}

func stripPathPrefix(path string, prefix string) string {
	if path == prefix {
		return "/"
	}
	withSlash := prefix + "/"
	if strings.HasPrefix(path, withSlash) {
		return "/" + strings.TrimPrefix(path, withSlash)
	}
	return path
}

func isCodexProxyPath(path string) bool {
	return path == "/backend-api/codex" || strings.HasPrefix(path, "/backend-api/codex/")
}

func tokenListContains(value string, token string) bool {
	for _, part := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(part), token) {
			return true
		}
	}
	return false
}

func singleJoiningSlash(a, b string) string {
	aslash := strings.HasSuffix(a, "/")
	bslash := strings.HasPrefix(b, "/")
	switch {
	case aslash && bslash:
		return a + b[1:]
	case !aslash && !bslash:
		return a + "/" + b
	default:
		return a + b
	}
}
