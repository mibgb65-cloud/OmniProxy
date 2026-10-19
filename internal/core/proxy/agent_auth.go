package proxy

import (
	"net/http"
	"strings"
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
