package proxy

import (
	"net/http"
	"strings"

	"omniproxy/internal/claudedesktop"
)

type ClientInfo struct {
	Key  string
	Name string
}

const (
	clientCodex         = "codex"
	clientClaude        = "claude"
	clientClaudeDesktop = "claude-desktop"
	clientDeepSeekTUI   = "deepseek-tui"
	clientOpenCode      = "opencode"
	clientPi            = "pi"
	clientGemini        = "gemini"
	clientOpenRouter    = "openrouter"
	clientTokenRouter   = "tokenrouter"
	clientSub2API       = "sub2api"
	clientNewAPI        = "newapi"
	clientAtria         = "atria"
	clientCursor        = "cursor"
	clientVSCode        = "vscode"
	clientWindsurf      = "windsurf"
	clientAider         = "aider"
	clientContinue      = "continue"
	clientCustom        = "custom"
	clientAPI           = "api"
	clientUnknown       = "unknown"
)

func clientInfoForRequest(r *http.Request, route routeInfo) ClientInfo {
	if r == nil {
		return knownClient(clientUnknown)
	}
	if info, ok := explicitClientInfo(r.Header); ok {
		return info
	}
	if info, ok := userAgentClientInfo(r.UserAgent()); ok {
		return info
	}
	path := ""
	if r.URL != nil {
		path = r.URL.Path
	}
	if info, ok := pathClientInfo(path, route); ok {
		return info
	}
	return knownClient(clientAPI)
}

func routeWithClient(r *http.Request, route routeInfo) routeInfo {
	info := clientInfoForRequest(r, route)
	route.ClientKey = info.Key
	route.ClientName = info.Name
	return route
}

func explicitClientInfo(header http.Header) (ClientInfo, bool) {
	for _, key := range []string{"X-OmniProxy-Client", "X-Client-Name", "X-Source-Client"} {
		value := strings.TrimSpace(header.Get(key))
		if value == "" {
			continue
		}
		return clientInfoFromLabel(value), true
	}
	return ClientInfo{}, false
}

func userAgentClientInfo(userAgent string) (ClientInfo, bool) {
	ua := strings.ToLower(strings.TrimSpace(userAgent))
	if ua == "" {
		return ClientInfo{}, false
	}
	switch {
	case strings.Contains(ua, "opencode"):
		return knownClient(clientOpenCode), true
	case strings.Contains(ua, "deepseek-tui") || strings.Contains(ua, "deepseek tui"):
		return knownClient(clientDeepSeekTUI), true
	case strings.Contains(ua, "pi-coding-agent") || strings.Contains(ua, "pi.dev"):
		return knownClient(clientPi), true
	case strings.Contains(ua, "codex"):
		return knownClient(clientCodex), true
	case strings.Contains(ua, "claude"):
		return knownClient(clientClaude), true
	case strings.Contains(ua, "gemini"):
		return knownClient(clientGemini), true
	case strings.Contains(ua, "cursor"):
		return knownClient(clientCursor), true
	case strings.Contains(ua, "windsurf"):
		return knownClient(clientWindsurf), true
	case strings.Contains(ua, "aider"):
		return knownClient(clientAider), true
	case strings.Contains(ua, "continue"):
		return knownClient(clientContinue), true
	case strings.Contains(ua, "vscode") || strings.Contains(ua, "visual studio code"):
		return knownClient(clientVSCode), true
	default:
		return ClientInfo{}, false
	}
}

func pathClientInfo(path string, route routeInfo) (ClientInfo, bool) {
	switch {
	case isOpenCodeRouterPath(path):
		return knownClient(clientOpenCode), true
	case isPiRouterPath(path):
		return knownClient(clientPi), true
	case isCodexProxyPath(path) || strings.HasPrefix(path, "/codex/") || path == "/codex" || route.CredentialType == "codex_auth_json":
		return knownClient(clientCodex), true
	case claudedesktop.IsGatewayPath(path):
		return knownClient(clientClaudeDesktop), true
	case isAnthropicRouterPath(path):
		return knownClient(clientClaude), true
	case strings.HasPrefix(path, "/gemini/") || path == "/gemini":
		return knownClient(clientGemini), true
	case strings.HasPrefix(path, "/openrouter/") || path == "/openrouter":
		return knownClient(clientOpenRouter), true
	case strings.HasPrefix(path, "/tokenrouter/") || path == "/tokenrouter":
		return knownClient(clientTokenRouter), true
	case strings.HasPrefix(path, "/sub2api/") || path == "/sub2api":
		return knownClient(clientSub2API), true
	case strings.HasPrefix(path, "/newapi/") || path == "/newapi":
		return knownClient(clientNewAPI), true
	case strings.HasPrefix(path, "/atria/") || path == "/atria":
		return knownClient(clientAtria), true
	case strings.HasPrefix(path, "/custom/") || path == "/custom":
		return knownClient(clientCustom), true
	default:
		return ClientInfo{}, false
	}
}

func clientInfoFromLabel(value string) ClientInfo {
	normalized := strings.ToLower(strings.TrimSpace(value))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.ReplaceAll(normalized, " ", "-")
	switch {
	case strings.Contains(normalized, "opencode"):
		return knownClient(clientOpenCode)
	case strings.Contains(normalized, "deepseek-tui"):
		return knownClient(clientDeepSeekTUI)
	case strings.Contains(normalized, "pi-coding-agent") || normalized == "pi":
		return knownClient(clientPi)
	case strings.Contains(normalized, "codex"):
		return knownClient(clientCodex)
	case strings.Contains(normalized, "claude-desktop") || strings.Contains(normalized, "claude-code-desktop"):
		return knownClient(clientClaudeDesktop)
	case strings.Contains(normalized, "claude"):
		return knownClient(clientClaude)
	case strings.Contains(normalized, "gemini"):
		return knownClient(clientGemini)
	case strings.Contains(normalized, "openrouter"):
		return knownClient(clientOpenRouter)
	case strings.Contains(normalized, "tokenrouter"):
		return knownClient(clientTokenRouter)
	case strings.Contains(normalized, "sub2api"):
		return knownClient(clientSub2API)
	case strings.Contains(normalized, "newapi") || strings.Contains(normalized, "new-api"):
		return knownClient(clientNewAPI)
	case strings.Contains(normalized, "atria"):
		return knownClient(clientAtria)
	case strings.Contains(normalized, "cursor"):
		return knownClient(clientCursor)
	case strings.Contains(normalized, "windsurf"):
		return knownClient(clientWindsurf)
	case strings.Contains(normalized, "aider"):
		return knownClient(clientAider)
	case strings.Contains(normalized, "continue"):
		return knownClient(clientContinue)
	case strings.Contains(normalized, "vscode") || strings.Contains(normalized, "visual-studio-code"):
		return knownClient(clientVSCode)
	}
	key := compactClientKey(normalized)
	if key == "" {
		return knownClient(clientUnknown)
	}
	return ClientInfo{Key: key, Name: strings.TrimSpace(value)}
}

func knownClient(key string) ClientInfo {
	switch key {
	case clientCodex:
		return ClientInfo{Key: clientCodex, Name: "Codex"}
	case clientClaude:
		return ClientInfo{Key: clientClaude, Name: "Claude Code"}
	case clientClaudeDesktop:
		return ClientInfo{Key: clientClaudeDesktop, Name: "Claude Code Desktop"}
	case clientDeepSeekTUI:
		return ClientInfo{Key: clientDeepSeekTUI, Name: "DeepSeek-TUI"}
	case clientOpenCode:
		return ClientInfo{Key: clientOpenCode, Name: "OpenCode"}
	case clientPi:
		return ClientInfo{Key: clientPi, Name: "Pi Coding Agent"}
	case clientGemini:
		return ClientInfo{Key: clientGemini, Name: "Gemini CLI"}
	case clientOpenRouter:
		return ClientInfo{Key: clientOpenRouter, Name: "OpenRouter"}
	case clientTokenRouter:
		return ClientInfo{Key: clientTokenRouter, Name: "TokenRouter"}
	case clientSub2API:
		return ClientInfo{Key: clientSub2API, Name: "sub2api"}
	case clientNewAPI:
		return ClientInfo{Key: clientNewAPI, Name: "new-api"}
	case clientAtria:
		return ClientInfo{Key: clientAtria, Name: "Atria"}
	case clientCursor:
		return ClientInfo{Key: clientCursor, Name: "Cursor"}
	case clientVSCode:
		return ClientInfo{Key: clientVSCode, Name: "VS Code"}
	case clientWindsurf:
		return ClientInfo{Key: clientWindsurf, Name: "Windsurf"}
	case clientAider:
		return ClientInfo{Key: clientAider, Name: "Aider"}
	case clientContinue:
		return ClientInfo{Key: clientContinue, Name: "Continue"}
	case clientCustom:
		return ClientInfo{Key: clientCustom, Name: "Custom Gateway"}
	case clientUnknown:
		return ClientInfo{Key: clientUnknown, Name: "Unknown Client"}
	default:
		return ClientInfo{Key: clientAPI, Name: "API Client"}
	}
}

func compactClientKey(value string) string {
	var builder strings.Builder
	for _, char := range value {
		switch {
		case char >= 'a' && char <= 'z':
			builder.WriteRune(char)
		case char >= '0' && char <= '9':
			builder.WriteRune(char)
		case char == '-':
			if builder.Len() > 0 {
				builder.WriteRune(char)
			}
		}
	}
	return strings.Trim(builder.String(), "-")
}
