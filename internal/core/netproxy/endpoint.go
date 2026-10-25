package netproxy

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// 默认端口对齐常见客户端：mixed/socks 10808，http 10809。
const (
	DefaultHost         = "127.0.0.1"
	DefaultSOCKSPort    = 10808
	DefaultHTTPPort     = 10809
	maxEndpointRunes    = 512
	maxProxyPasswordLen = 256
)

// schemeAliases 接受用户从代理客户端里直接复制的写法。
var schemeAliases = map[string]string{
	"auto":    SchemeAuto,
	"mixed":   SchemeAuto,
	"http":    SchemeHTTP,
	"https":   SchemeHTTP,
	"socks":   SchemeSOCKS5,
	"socks5":  SchemeSOCKS5,
	"socks5h": SchemeSOCKS5,
}

// ParseEndpoint 解析用户输入的代理端点，支持：
//
//	10808                       → auto://127.0.0.1:10808
//	mixed:10808 / socks:10808   → auto / socks5，主机取 loopback
//	127.0.0.1:10809             → 保留原主机
//	http://u:p@127.0.0.1:10809  → http + 凭证
//
// 非 URL 形式必须带端口：代理端口没有可靠默认值，猜错会让用户以为功能坏了。
func ParseEndpoint(text string) (Endpoint, error) {
	raw := strings.TrimSpace(text)
	if raw == "" || len(raw) > maxEndpointRunes || strings.ContainsAny(raw, "\r\n\t ") {
		return Endpoint{}, fmt.Errorf("%w: empty or malformed", ErrInvalidEndpoint)
	}
	endpoint := Endpoint{Scheme: SchemeAuto}
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Hostname() == "" || parsed.Opaque != "" || parsed.Fragment != "" || parsed.RawQuery != "" || parsed.ForceQuery || (parsed.Path != "" && parsed.Path != "/") {
			return Endpoint{}, fmt.Errorf("%w: malformed URL", ErrInvalidEndpoint)
		}
		scheme, ok := schemeAliases[strings.ToLower(parsed.Scheme)]
		if !ok {
			return Endpoint{}, fmt.Errorf("%w: unsupported scheme", ErrInvalidEndpoint)
		}
		endpoint.Scheme, endpoint.Host = scheme, parsed.Hostname()
		if parsed.User != nil {
			endpoint.Username = parsed.User.Username()
			endpoint.Password, _ = parsed.User.Password()
		}
		endpoint.Port = DefaultPort(scheme)
		if port := parsed.Port(); port != "" {
			value, err := parsePort(port)
			if err != nil {
				return Endpoint{}, err
			}
			endpoint.Port = value
		}
	} else {
		rest := raw
		if alias, tail, ok := strings.Cut(raw, ":"); ok && isSchemeAlias(alias) {
			endpoint.Scheme = schemeAliases[strings.ToLower(strings.TrimSpace(alias))]
			rest = strings.TrimSpace(tail)
		}
		host, port, err := splitHostPort(rest)
		if err != nil {
			return Endpoint{}, err
		}
		endpoint.Host, endpoint.Port = host, port
	}
	if endpoint.Host == "" {
		endpoint.Host = DefaultHost
	}
	if err := validateEndpointHost(endpoint.Host); err != nil {
		return Endpoint{}, err
	}
	if len(endpoint.Username) > maxEndpointRunes || len(endpoint.Password) > maxProxyPasswordLen {
		return Endpoint{}, fmt.Errorf("%w: credentials too long", ErrInvalidEndpoint)
	}
	for _, value := range []string{endpoint.Username, endpoint.Password} {
		for _, char := range value {
			if char < 33 || char > 126 {
				return Endpoint{}, fmt.Errorf("%w: credentials contain control characters", ErrInvalidEndpoint)
			}
		}
	}
	if endpoint.Password != "" && endpoint.Username == "" {
		return Endpoint{}, fmt.Errorf("%w: password without username", ErrInvalidEndpoint)
	}
	return endpoint, nil
}

func isSchemeAlias(value string) bool {
	_, ok := schemeAliases[strings.ToLower(strings.TrimSpace(value))]
	return ok
}

// splitHostPort 只接受显式端口；纯数字被视为 loopback 端口。
func splitHostPort(value string) (string, int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", 0, fmt.Errorf("%w: missing address", ErrInvalidEndpoint)
	}
	if allDigits(value) {
		port, err := parsePort(value)
		if err != nil {
			return "", 0, err
		}
		return DefaultHost, port, nil
	}
	host, portText, err := net.SplitHostPort(value)
	if err != nil {
		return "", 0, fmt.Errorf("%w: address must include a port", ErrInvalidEndpoint)
	}
	if host == "" {
		return "", 0, fmt.Errorf("%w: missing host", ErrInvalidEndpoint)
	}
	port, err := parsePort(portText)
	if err != nil {
		return "", 0, err
	}
	return host, port, nil
}

func parsePort(value string) (int, error) {
	port, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || port < 1 || port > 65535 {
		return 0, fmt.Errorf("%w: port out of range", ErrInvalidEndpoint)
	}
	return port, nil
}

func validateEndpointHost(host string) error {
	if host == "" || len(host) > 255 {
		return fmt.Errorf("%w: invalid host", ErrInvalidEndpoint)
	}
	if allDigits(host) {
		// 纯数字当作主机名说明端口写错了（例如 10808:80）。
		return fmt.Errorf("%w: invalid host", ErrInvalidEndpoint)
	}
	for _, char := range host {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z', char >= '0' && char <= '9':
		case char == '-' || char == '.' || char == ':' || char == '_' || char == '%':
		default:
			return fmt.Errorf("%w: invalid host", ErrInvalidEndpoint)
		}
	}
	if strings.Contains(host, ":") && net.ParseIP(host) == nil {
		return fmt.Errorf("%w: invalid host", ErrInvalidEndpoint)
	}
	return nil
}

// DefaultPort 返回给定协议下的默认端口，用于输入框提示。
func DefaultPort(scheme string) int {
	if scheme == SchemeHTTP {
		return DefaultHTTPPort
	}
	return DefaultSOCKSPort
}

// joinHostPort 为 IPv6 主机补方括号。
func joinHostPort(host string, port int) string {
	return net.JoinHostPort(host, strconv.Itoa(port))
}

func allDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

// ValidMode 校验提供商代理策略取值。
func ValidMode(mode string) bool {
	return mode == ModeInherit || mode == ModeDirect || mode == ModeProxy
}

// ValidScheme 校验持久化的协议取值。
func ValidScheme(scheme string) bool {
	return scheme == SchemeAuto || scheme == SchemeHTTP || scheme == SchemeSOCKS5
}
