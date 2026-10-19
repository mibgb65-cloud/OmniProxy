package subscription

import "strings"

// Diagnostic 只包含有限状态和稳定错误码，不携带身份、会话 ID 或授权信息。
type Diagnostic struct {
	State string
	Code  string
}

// DiagnosticState 返回当前或最近一次授权结果，供脱敏运行诊断使用。
func (s *Service) DiagnosticState() Diagnostic {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.current != nil {
		return Diagnostic{State: s.current.view.State, Code: s.current.view.ErrorCode}
	}
	var latest State
	for _, item := range s.history {
		if item.ExpiresAt > latest.ExpiresAt {
			latest = item
		}
	}
	return Diagnostic{State: latest.State, Code: latest.ErrorCode}
}

// ErrorCode 只提取本模块完整的稳定错误码，原始消息和其他提供商正文不进入诊断。
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}
	value := strings.TrimPrefix(err.Error(), "subscription: ")
	switch value {
	case "quota_unavailable", "quota_timeout", "quota_refresh_failed", "quota_not_authenticated", "identity_unavailable", "codex_unavailable", "login_expired", "operation_cancelled", "login_failed", "request_failed", "protocol_failed", "identity_changed", "cleanup_failed", "busy",
		"account_unavailable", "initialization_failed", "invalid_input", "invalid_provider", "invalid_account":
		return value
	}
	return "operation_failed"
}
