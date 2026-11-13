package dumate

import (
	"strings"
	"unicode/utf8"
)

const (
	// PastedDomain 是手动粘贴字符串的归属域；console.bce.baidu.com 是它的子域。
	PastedDomain = ".baidu.com"
	// SessionCookie 是控制台要求的登录 Cookie；缺失时无法建立会话。
	SessionCookie = "bce-sessionid"
	// CSRFCookie 的值同时作为 POST 的 csrftoken。
	CSRFCookie = "bce-user-info"

	maxCookies       = 200
	maxCookieName    = 256
	maxCookieValue   = 4096
	maxCookieDomain  = 253
	maxHeaderBytes   = 12 << 10
	maxCookieInput   = 64 << 10
	quoteCharacters  = ":|;= \t"
	quotedCookieTrim = `"'`
)

// Cookie 是控制台需要的浏览器 Cookie 子集；其余字段不进入受保护存储。
type Cookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain,omitempty"`
	Path   string `json:"path,omitempty"`
}

// validCookieName 只接受浏览器允许的 token 字符。
func validCookieName(name string) bool {
	if name == "" || len(name) > maxCookieName {
		return false
	}
	for _, char := range name {
		switch {
		case char >= 'a' && char <= 'z', char >= 'A' && char <= 'Z', char >= '0' && char <= '9':
		case strings.ContainsRune("!#$%&'*+-.^_`|~", char):
		default:
			return false
		}
	}
	return true
}

func validCookieDomain(domain string) bool {
	if domain == "" {
		return true
	}
	if len(domain) > maxCookieDomain {
		return false
	}
	value := strings.TrimPrefix(domain, ".")
	if value == "" || strings.HasPrefix(value, ".") {
		return false
	}
	for _, char := range value {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || char == '-' || char == '.') {
			return false
		}
	}
	return true
}

// validCookieValue 拒绝控制字符与无效 UTF-8，避免头注入和日志伪造。
func validCookieValue(value string) bool {
	if len(value) > maxCookieValue || !utf8.ValidString(value) {
		return false
	}
	for _, char := range value {
		if char < 0x20 || char == 0x7f {
			return false
		}
	}
	return true
}

func validCookie(cookie Cookie) bool {
	return validCookieName(cookie.Name) && validCookieValue(cookie.Value) && validCookieDomain(cookie.Domain)
}

// Validate 校验 Cookie 集合的边界与登录必需字段。
func Validate(cookies []Cookie) error {
	if len(cookies) == 0 || len(cookies) > maxCookies {
		return Failure(CodeCredentialUnavailable)
	}
	found := false
	for _, cookie := range cookies {
		if !validCookie(cookie) {
			return Failure(CodeCredentialUnavailable)
		}
		if cookie.Name == SessionCookie {
			found = true
		}
	}
	if !found {
		return Failure(CodeCredentialUnavailable)
	}
	return nil
}

// ParseCookies 解析粘贴的 "name=value; name2=value2" 字符串。
// 粘贴值不带域名，统一归属 .baidu.com；同名项以首次出现为准。
func ParseCookies(raw string) ([]Cookie, error) {
	if raw == "" || len(raw) > maxCookieInput || !utf8.ValidString(raw) {
		return nil, Failure(CodeCredentialUnavailable)
	}
	seen := make(map[string]bool)
	cookies := make([]Cookie, 0, 16)
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		index := strings.Index(part, "=")
		if index <= 0 {
			continue
		}
		name := strings.TrimSpace(part[:index])
		value := strings.TrimSpace(part[index+1:])
		value = strings.Trim(value, quotedCookieTrim)
		cookie := Cookie{Name: name, Value: value, Domain: PastedDomain}
		if !validCookie(cookie) || seen[name] {
			continue
		}
		seen[name] = true
		cookies = append(cookies, cookie)
	}
	if len(cookies) == 0 || len(cookies) > maxCookies {
		return nil, Failure(CodeCredentialUnavailable)
	}
	return cookies, nil
}

// HostMatches 报告 Cookie 是否应发往该主机；空域名表示不限制。
func HostMatches(cookie Cookie, hostname string) bool {
	if cookie.Domain == "" {
		return true
	}
	domain := strings.TrimPrefix(cookie.Domain, ".")
	return hostname == domain || strings.HasSuffix(hostname, "."+domain)
}

// quoteCookieValue 复刻桌面端序列化规则：含分隔符的值必须加引号，
// 否则 POST 会被上游当成「登录凭证已过期」。
func quoteCookieValue(value string) string {
	if strings.ContainsAny(value, quoteCharacters) {
		return `"` + strings.ReplaceAll(value, `"`, "") + `"`
	}
	return value
}

// HeaderFor 生成发往指定主机的 Cookie 头；超过边界时返回空串。
func HeaderFor(cookies []Cookie, hostname string) string {
	parts := make([]string, 0, len(cookies))
	size := 0
	for _, cookie := range cookies {
		if !validCookie(cookie) || !HostMatches(cookie, hostname) {
			continue
		}
		part := cookie.Name + "=" + quoteCookieValue(strings.Trim(cookie.Value, quotedCookieTrim))
		size += len(part) + 2
		if size > maxHeaderBytes {
			return ""
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, "; ")
}

// CSRF 返回目标主机的 csrftoken；控制台 POST 必须携带。
func CSRF(cookies []Cookie, hostname string) string {
	for _, cookie := range cookies {
		if cookie.Name != CSRFCookie || !validCookie(cookie) || !HostMatches(cookie, hostname) {
			continue
		}
		return strings.Trim(cookie.Value, quotedCookieTrim)
	}
	return ""
}

// CloneCookies 返回独立副本，避免调用方修改受保护内存中的切片。
func CloneCookies(cookies []Cookie) []Cookie {
	result := make([]Cookie, len(cookies))
	copy(result, cookies)
	return result
}
