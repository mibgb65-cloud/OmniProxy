package subscription

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
