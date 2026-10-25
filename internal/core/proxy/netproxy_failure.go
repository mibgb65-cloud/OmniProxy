package proxy

import (
	"omniproxy/internal/core/netproxy"
)

// UpstreamFailure 把出站失败换算成边界错误。
//
// 代理层故障必须与上游故障分开：本地代理没开时如果只报 upstream_unavailable，
// 故障切换会把它当成账号失败并给整池账号写冷却。
func UpstreamFailure(err error) error {
	if err == nil {
		return nil
	}
	switch netproxy.CodeOf(err) {
	case "proxy_unavailable":
		return &Error{Code: "proxy_unavailable", StatusCode: 502}
	case "proxy_protocol_mismatch":
		return &Error{Code: "proxy_protocol_mismatch", StatusCode: 502}
	case "proxy_not_configured":
		return &Error{Code: "proxy_not_configured", StatusCode: 503}
	case "proxy_configuration_unavailable":
		return &Error{Code: "configuration_unavailable", StatusCode: 503}
	}
	return &Error{Code: "upstream_unavailable", StatusCode: 502}
}

// ProxyLayerFailure 报告该错误码是否属于代理层（不参与账号冷却）。
func ProxyLayerFailure(code string) bool {
	switch code {
	case "proxy_unavailable", "proxy_protocol_mismatch", "proxy_not_configured":
		return true
	}
	return false
}
