package mimo

import (
	"net"
	"net/url"
	"strconv"
	"strings"

	"omniproxy/internal/core/proxy"
)

func endpoint(base, path string) (string, error) {
	if base == "" {
		base = DefaultBaseURL
	}
	u, err := url.Parse(base)
	if err != nil || len(base) > 2048 || u.Opaque != "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.ForceQuery || u.RawPath != "" {
		return "", proxy.Failure("invalid_configuration", 503)
	}
	loopback := strings.EqualFold(u.Hostname(), "localhost")
	if ip := net.ParseIP(u.Hostname()); ip != nil {
		loopback = loopback || ip.IsLoopback()
	}
	// 普通 Key 只发往官方按量域名；回环地址用于本机代理及无凭证契约测试。
	if !loopback && (!strings.EqualFold(u.Hostname(), "api.xiaomimimo.com") || u.Scheme != "https" || u.Port() != "" && u.Port() != "443") {
		return "", proxy.Failure("invalid_configuration", 503)
	}
	if loopback && u.Scheme != "http" && u.Scheme != "https" {
		return "", proxy.Failure("invalid_configuration", 503)
	}
	if port := u.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return "", proxy.Failure("invalid_configuration", 503)
		}
	}
	if root := strings.TrimRight(u.Path, "/"); root != "" && root != "/v1" {
		return "", proxy.Failure("invalid_configuration", 503)
	}
	// Messages 在 /anthropic/v1，不能在已有 /v1 后追加而形成错误地址。
	u.Path, u.RawPath = path, ""
	return u.String(), nil
}

func validKey(key []byte) bool {
	if len(key) == 0 || len(key) > 8192 {
		return false
	}
	for _, ch := range key {
		if ch < 33 || ch > 126 {
			return false
		}
	}
	return true
}
