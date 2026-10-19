package subscription

// InProgress 同时用于共享登录界面和 Google 会话生命周期。
func InProgress(state string) bool {
	switch state {
	case "starting", "waiting", "exchanging", "verifying", "saving":
		return true
	default:
		return false
	}
}

func (s *Service) progress(job *session, state string) {
	s.mu.Lock()
	if s.current != job {
		s.mu.Unlock()
		return
	}
	job.view.State = state
	// 官方已确认授权后，不再向 UI 暴露已消费的登录链接与设备码。
	job.view.URL, job.view.UserCode = "", ""
	s.mu.Unlock()
	s.changed()
}
