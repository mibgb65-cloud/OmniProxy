package proxy

import (
	"net/http"
	"strings"

	"omniproxy/internal/core/agentendpoint"
)

// scopedAgentPath 支持内置 Codex 提供商，丢弃原认证头且禁止记录秘密路径。
func scopedAgentPath(r *http.Request) (string, string, bool) {
	path := r.URL.Path
	if !strings.HasPrefix(path, "/agent/") {
		return path, "", true
	}
	parts := strings.Split(path, "/")
	compact := len(parts) == 7 && parts[5] == "responses" && parts[6] == "compact"
	if (len(parts) != 6 && !compact) || parts[2] != "codex" || len(parts[3]) != 68 || !strings.HasPrefix(parts[3], "opk_") || parts[4] != "v1" {
		return "", "", false
	}
	for _, ch := range parts[3][4:] {
		if !(ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f') {
			return "", "", false
		}
	}
	if compact {
		return "/v1/responses/compact", parts[3], true
	}
	return "/v1/" + parts[5], parts[3], true
}

// resolveAgentPath 把入口折叠成协议规范路径，并返回该入口允许的受管客户端。
//
// 每个受管客户端有固定前缀（例如 Codex 的 /codex/v1、Claude 的 /claude/v1/messages）；
// 前缀只做路由分组与归属校验，协议处理仍只看折叠后的 /v1/... 路径。
// 无前缀的 /v1/... 保留给升级前写入的客户端配置，归属完全由接入 Key 决定。
func resolveAgentPath(r *http.Request) (path, pathKey, prefix string, clients []string, ok bool) {
	path = r.URL.Path
	if strings.HasPrefix(path, "/agent/") {
		scoped, key, valid := scopedAgentPath(r)
		if !valid {
			return "", "", "", nil, false
		}
		return scoped, key, "", []string{"codex"}, true
	}
	candidate, rest, found := splitAgentPrefix(path)
	if !found {
		return path, "", "", nil, true
	}
	return rest, "", candidate, agentendpoint.Clients(candidate), true
}

// splitAgentPrefix 按完整路径段匹配固定前缀，拒绝 /codexanything 这类近似路径。
func splitAgentPrefix(path string) (prefix, rest string, ok bool) {
	if len(path) < 2 || path[0] != '/' {
		return "", "", false
	}
	trimmed := path[1:]
	segment := trimmed
	rest = "/"
	if index := strings.IndexByte(trimmed, '/'); index >= 0 {
		segment = trimmed[:index]
		rest = trimmed[index:]
	}
	prefix = "/" + segment
	if agentendpoint.Clients(prefix) == nil {
		return "", "", false
	}
	return prefix, rest, true
}

// clientAllowed 判断已鉴权的客户端是否允许使用该入口前缀。
func clientAllowed(clients []string, client string) bool {
	for _, candidate := range clients {
		if candidate == client {
			return true
		}
	}
	return false
}
