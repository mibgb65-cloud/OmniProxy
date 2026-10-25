// Package netproxy 负责出站代理：端点解析、mixed 协议识别与按提供商分流的上游 Transport。
//
// 它属于 Core：不依赖数据库、Wails 或 React。真实的设置读取由应用层实现 Resolver 后
// 通过 Install 注入；未安装时 Wrap 原样返回底层 RoundTripper，旧行为保持不变。
package netproxy

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

// 代理协议取值（与 proxy_settings.scheme 一致）。auto 表示 mixed 端口，由探测决定。
const (
	SchemeAuto   = "auto"
	SchemeHTTP   = "http"
	SchemeSOCKS5 = "socks5"
)

// 每个提供商的路由策略（与 providers.proxy_mode 一致）。
const (
	ModeInherit = "inherit"
	ModeDirect  = "direct"
	ModeProxy   = "proxy"
)

// 决策与失败原因。原因只用于诊断文案，不能当作可展示给用户的原文。
const (
	ReasonDisabled      = "disabled"
	ReasonSelected      = "selected"
	ReasonInherited     = "inherited"
	ReasonDirect        = "direct"
	ReasonNoProxy       = "no_proxy"
	ReasonNotSelected   = "not_selected"
	ReasonNotConfigured = "proxy_not_configured"
)

var (
	// ErrUnavailable 表示代理端口本身不可用（未监听、连接被拒）。
	ErrUnavailable = errors.New("netproxy: proxy unavailable")
	// ErrProtocolMismatch 表示端口不是预期的 HTTP/SOCKS5 代理。
	ErrProtocolMismatch = errors.New("netproxy: proxy protocol mismatch")
	// ErrInvalidEndpoint 表示端点文本无法解析成合法代理地址。
	ErrInvalidEndpoint = errors.New("netproxy: invalid proxy endpoint")
)

// Error 携带稳定错误码，正文不包含凭证。
type Error struct {
	Code string
	Err  error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return "netproxy: " + e.Code
	}
	return "netproxy: " + e.Code + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error { return e.Err }

// CodeOf 返回错误链上最外层 netproxy 错误码；没有时返回空串。
func CodeOf(err error) string {
	var target *Error
	if errors.As(err, &target) {
		return target.Code
	}
	if errors.Is(err, ErrUnavailable) {
		return "proxy_unavailable"
	}
	if errors.Is(err, ErrProtocolMismatch) {
		return "proxy_protocol_mismatch"
	}
	return ""
}

// Scope 标识发起请求的提供商，由应用层在调用上游前写入 ctx。
type Scope struct {
	ProviderID   string
	ProviderKind string
}

type scopeKey struct{}

// WithProvider 把提供商身份写入 ctx；已存在时覆盖，避免嵌套调用沿用旧身份。
func WithProvider(ctx context.Context, providerID, kind string) context.Context {
	if providerID == "" && kind == "" {
		return ctx
	}
	return context.WithValue(ctx, scopeKey{}, Scope{ProviderID: providerID, ProviderKind: kind})
}

// ScopeFrom 读取 ctx 中的提供商身份。
func ScopeFrom(ctx context.Context) (Scope, bool) {
	if ctx == nil {
		return Scope{}, false
	}
	scope, ok := ctx.Value(scopeKey{}).(Scope)
	return scope, ok
}

// Endpoint 是规范化后的代理监听地址。
type Endpoint struct {
	Scheme   string
	Host     string
	Port     int
	Username string
	Password string
}

// Address 返回 host:port，不含凭证。
func (e Endpoint) Address() string { return joinHostPort(e.Host, e.Port) }

// String 返回脱敏描述，日志与 UI 只能使用它。
func (e Endpoint) String() string {
	scheme := e.Scheme
	if scheme == "" {
		scheme = SchemeAuto
	}
	return scheme + "://" + e.Address()
}

// URL 生成 http.Transport 使用的代理地址；用户名为空时不带凭证。
func (e Endpoint) URL(scheme string) *url.URL {
	u := &url.URL{Scheme: scheme, Host: e.Address()}
	if e.Username != "" {
		u.User = url.UserPassword(e.Username, e.Password)
	}
	return u
}

// Redact 从任意文本中抹掉可能存在的凭证，用于错误与日志。
func Redact(text string) string {
	if !strings.Contains(text, "@") {
		return text
	}
	at := strings.LastIndex(text, "@")
	head := text[:at]
	if idx := strings.Index(head, "://"); idx >= 0 {
		return head[:idx+3] + "***@" + text[at+1:]
	}
	return "***@" + text[at+1:]
}
