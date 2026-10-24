package proxy

import (
	"net/url"
	"strings"
)

// MaxPassthroughBytes 与 v1 一致：透明转发按 32 MiB 读取正文，不解析内容。
const MaxPassthroughBytes = 32 << 20

// codexVersionPrefixes 是拼接上游基址前逐个剥离的版本段，顺序与 v1 一致：
// 先去掉本地入口遗留的完整前缀，再去掉客户端追加的 /codex 与 /v1。
var codexVersionPrefixes = []string{"/backend-api/codex", "/codex", "/v1"}

// PassthroughURL 按 v1 语义拼出上游地址：剥离版本段后合并基址路径与查询串。
func PassthroughURL(baseURL, path, query string) (string, error) {
	base, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || base.Scheme != "http" && base.Scheme != "https" || base.Host == "" {
		return "", Failure("invalid_configuration", 503)
	}
	// 只允许在固定上游主机内换路径，不能让 .. 把请求带到别的路径。
	if strings.Contains(path, "..") || strings.ContainsAny(path, "\r\n\x00") {
		return "", Failure("invalid_request", 400)
	}
	out := *base
	out.Path = joinURLPath(base.Path, passthroughPath(base.Path, path))
	out.RawQuery = query
	return out.String(), nil
}

// PassthroughSocketURL 与 PassthroughURL 同一套路径规则，只把协议换成 WebSocket。
func PassthroughSocketURL(baseURL, path, query string) (string, error) {
	target, err := PassthroughURL(baseURL, path, query)
	if err != nil {
		return "", err
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return "", Failure("invalid_configuration", 503)
	}
	switch parsed.Scheme {
	case "https":
		parsed.Scheme = "wss"
	case "http":
		parsed.Scheme = "ws"
	default:
		return "", Failure("invalid_configuration", 503)
	}
	return parsed.String(), nil
}

// passthroughPath 去掉客户端追加的版本段；基址已带版本后缀时不再重复。
func passthroughPath(basePath, path string) string {
	next := path
	for _, prefix := range codexVersionPrefixes {
		next = stripPathSegment(next, prefix)
	}
	if next == "" {
		return "/"
	}
	if baseHasVersionSuffix(basePath) && strings.HasPrefix(next, "/v1/") {
		return "/" + strings.TrimPrefix(next, "/v1/")
	}
	return next
}

// stripPathSegment 只按完整路径段剥离，避免 /v1abc 之类被误当作版本段。
func stripPathSegment(path, prefix string) string {
	if path == prefix {
		return "/"
	}
	if strings.HasPrefix(path, prefix+"/") {
		return strings.TrimPrefix(path, prefix)
	}
	return path
}

// joinURLPath 合并基址路径与请求路径，保持单斜杠。
func joinURLPath(base, path string) string {
	switch {
	case base == "":
		return path
	case path == "":
		return base
	case strings.HasSuffix(base, "/") && strings.HasPrefix(path, "/"):
		return base + path[1:]
	case !strings.HasSuffix(base, "/") && !strings.HasPrefix(path, "/"):
		return base + "/" + path
	}
	return base + path
}

// baseHasVersionSuffix 判断基址最后一段是否是 v1、v2 之类版本号。
func baseHasVersionSuffix(path string) bool {
	trimmed := strings.Trim(strings.TrimSpace(path), "/")
	if trimmed == "" {
		return false
	}
	parts := strings.Split(trimmed, "/")
	last := strings.ToLower(parts[len(parts)-1])
	if len(last) < 2 || last[0] != 'v' {
		return false
	}
	for _, char := range last[1:] {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// passthroughProtocol 只用于记账分类，不改变转发语义。
func passthroughProtocol(path string) string {
	switch {
	case strings.HasSuffix(path, "/responses/compact"):
		return "responses/compact"
	case strings.HasSuffix(path, "/responses"):
		return "responses"
	case strings.HasSuffix(path, "/chat/completions"):
		return "chat"
	case strings.HasSuffix(path, "/messages"):
		return "messages"
	default:
		return "responses"
	}
}
