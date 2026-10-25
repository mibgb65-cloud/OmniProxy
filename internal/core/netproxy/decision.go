package netproxy

import (
	"net"
	"strings"
)

// Snapshot 是应用层读取到的一次代理配置快照，Resolver 负责缓存与失效。
type Snapshot struct {
	// Version 变化时 Transport 丢弃协议缓存与连接池（保存设置后立即生效）。
	Version   string
	Enabled   bool
	Endpoint  Endpoint
	Detected  string
	NoProxy   []string
	Modes     map[string]string
	HostModes map[string]string
}

// Decision 是单个请求的路由结论。
type Decision struct {
	UseProxy   bool
	Reason     string
	ProviderID string
	Mode       string
}

// ModeFor 解析该请求所属提供商的路由策略：作用域优先，其次目标主机名。
func (s Snapshot) ModeFor(scope Scope, target string) (string, string) {
	if scope.ProviderID != "" {
		if mode, ok := s.Modes[scope.ProviderID]; ok && ValidMode(mode) {
			return mode, scope.ProviderID
		}
	}
	host := strings.ToLower(target)
	if mode, ok := s.HostModes[host]; ok && ValidMode(mode) {
		return mode, ""
	}
	if hostOnly, _, err := net.SplitHostPort(host); err == nil {
		if mode, ok := s.HostModes[hostOnly]; ok && ValidMode(mode) {
			return mode, ""
		}
	}
	return ModeInherit, scope.ProviderID
}

// Decide 按固定顺序给出路由：绕行/回环 → direct → proxy（未配置则报错）→ inherit → 直连。
func Decide(snapshot Snapshot, scope Scope, target string) Decision {
	mode, providerID := snapshot.ModeFor(scope, target)
	decision := Decision{ProviderID: providerID, Mode: mode}
	switch {
	case mode == ModeDirect:
		decision.Reason = ReasonDirect
		return decision
	case bypassed(snapshot.NoProxy, target) && mode != ModeProxy:
		decision.Reason = ReasonNoProxy
		return decision
	case isLoopbackTarget(target) && mode != ModeProxy:
		decision.Reason = ReasonNoProxy
		return decision
	case mode == ModeProxy:
		if !snapshot.Configured() {
			decision.Reason = ReasonNotConfigured
			return decision
		}
		decision.UseProxy, decision.Reason = true, ReasonSelected
		return decision
	case snapshot.Enabled:
		if !snapshot.Configured() {
			decision.Reason = ReasonNotConfigured
			return decision
		}
		decision.UseProxy, decision.Reason = true, ReasonInherited
		return decision
	default:
		decision.Reason = ReasonDisabled
		return decision
	}
}

// Configured 表示代理端点有可用的主机与端口。
func (s Snapshot) Configured() bool {
	return s.Endpoint.Host != "" && s.Endpoint.Port > 0 && s.Endpoint.Port <= 65535
}

// bypassed 匹配 no_proxy 列表：精确主机、主机:端口或 ".suffix"/"*.suffix" 后缀。
func bypassed(list []string, target string) bool {
	host := strings.ToLower(target)
	hostOnly, port, err := net.SplitHostPort(host)
	if err != nil {
		hostOnly, port = host, ""
	}
	for _, entry := range list {
		item := strings.ToLower(strings.TrimSpace(entry))
		if item == "" {
			continue
		}
		if item == host || item == hostOnly {
			return true
		}
		if port != "" && item == net.JoinHostPort(hostOnly, port) {
			return true
		}
		if strings.HasPrefix(item, "*.") || strings.HasPrefix(item, ".") {
			suffix := strings.TrimPrefix(strings.TrimPrefix(item, "*"), ".")
			if hostOnly == suffix || strings.HasSuffix(hostOnly, "."+suffix) {
				return true
			}
		}
	}
	return false
}

func isLoopbackTarget(target string) bool {
	host := target
	if hostOnly, _, err := net.SplitHostPort(target); err == nil {
		host = hostOnly
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}
