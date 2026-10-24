package proxy

import (
	"context"
	"net/http"
	"strings"

	"omniproxy/internal/core/agentendpoint"
)

// entry 是折叠后的入口：协议路径、入口前缀与允许归属的受管客户端。
type entry struct {
	path        string
	pathKey     string
	prefix      string
	clients     []string
	passthrough bool
}

// resolveEntry 先判定 Codex 透明入口（v1 语义：前缀之后任意路径），
// 其余路径沿用固定协议入口解析，保持其它客户端的既有行为。
func resolveEntry(r *http.Request, passthrough bool) (entry, bool) {
	if passthrough {
		if rest, key, ok := codexPassthroughPath(r.URL.Path); ok {
			return entry{path: rest, pathKey: key, prefix: "/codex", clients: agentendpoint.Clients("/codex"), passthrough: true}, true
		}
	}
	path, key, prefix, clients, ok := resolveAgentPath(r)
	if !ok {
		return entry{}, false
	}
	return entry{path: path, pathKey: key, prefix: prefix, clients: clients}, true
}

// authorizeEntry 复刻既有准入顺序：接入 Key 优先，固定前缀在无 Key 时按归属放行。
// 返回的 key 只用于 WebSocket 帧级复核，byPrefix 表示本次放行来自前缀归属。
func (s *Server) authorizeEntry(ctx context.Context, r *http.Request, e entry) (string, string, bool, error) {
	if s.opts.Authorize == nil {
		return "", e.pathKey, false, nil
	}
	key := e.pathKey
	if key == "" {
		key = r.Header.Get("X-Api-Key")
	}
	if key == "" {
		key = r.Header.Get("X-OmniProxy-Key")
	}
	if key == "" {
		key = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	client := ""
	authorized := false
	// 超过上限的凭据绝不进入比较；Codex 官方登录令牌本身就超过该上限。
	if len(key) <= 512 {
		client, authorized = s.opts.Authorize(ctx, key)
	}
	if authorized && client != "" && len(e.clients) > 0 && !clientAllowed(e.clients, client) {
		return "", "", false, Failure("client_mismatch", 403)
	}
	byPrefix := false
	if !authorized && e.prefix != "" && s.opts.PrefixAccess != nil {
		// Codex 保留官方 auth.json，网关无法用接入 Key 校验其登录令牌。
		client, authorized = s.opts.PrefixAccess(ctx, e.prefix)
		byPrefix = authorized
	}
	if !authorized {
		return "", "", false, Failure("unauthorized", 401)
	}
	return client, key, byPrefix, nil
}

// codexPassthroughPath 接受 /codex/...、/backend-api/codex/... 与
// /agent/codex/opk_<64hex>/... 三种形态；前两种与 v1 的两种客户端基址对应。
func codexPassthroughPath(path string) (string, string, bool) {
	if rest, ok := cutPathPrefix(path, "/codex"); ok {
		return normalizeEntryPath(rest), "", true
	}
	if rest, ok := cutPathPrefix(path, "/backend-api/codex"); ok {
		return normalizeEntryPath(rest), "", true
	}
	if !strings.HasPrefix(path, "/agent/codex/") {
		return "", "", false
	}
	key, rest, found := strings.Cut(strings.TrimPrefix(path, "/agent/codex/"), "/")
	if !validAccessKey(key) {
		return "", "", false
	}
	if !found {
		rest = "/"
	}
	return normalizeEntryPath(rest), key, true
}

// isTransparentPath 判定是否走透明转发：协议转换与本地模型目录之外全部转发，
// 这是 v1 的入口语义——不再逐个枚举端点。
func isTransparentPath(path string) bool {
	switch path {
	case "/v1/chat/completions", "/v1/messages", "/v1/codex-models":
		return false
	default:
		return true
	}
}

// cutPathPrefix 只按完整路径段匹配前缀，拒绝 /codexanything 这类近似路径。
func cutPathPrefix(path, prefix string) (string, bool) {
	if path == prefix {
		return "/", true
	}
	if strings.HasPrefix(path, prefix+"/") {
		return strings.TrimPrefix(path, prefix), true
	}
	return "", false
}

func normalizeEntryPath(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}
