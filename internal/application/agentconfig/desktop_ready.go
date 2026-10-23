package agentconfig

import (
	"encoding/json"

	"omniproxy/internal/core/agentendpoint"
)

// DesktopGatewayActive 确认启动模式、选中配置和网关配置均已就绪，避免按钮打开到官方登录页。
func (s *Service) DesktopGatewayActive() (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	change, err := s.desktopFiles()
	if err != nil {
		return false, err
	}
	defer change.clear()
	state := change.current
	if state.Normal.Value == nil || *state.Normal.Value != "3p" || state.ThirdParty.Value == nil || *state.ThirdParty.Value != "3p" || state.Applied.Value == nil || *state.Applied.Value != DesktopProfileID {
		return false, nil
	}
	body, exists, err := read(s.paths.ClaudeDesktop)
	if err != nil {
		return false, err
	}
	defer clear(body)
	var profile struct {
		Provider string `json:"inferenceProvider"`
		BaseURL  string `json:"inferenceGatewayBaseUrl"`
		Key      string `json:"inferenceGatewayApiKey"`
	}
	if !exists || json.Unmarshal(body, &profile) != nil {
		return false, nil
	}
	baseURL, ok := agentendpoint.BaseURL("claude-desktop", s.port)
	if !ok {
		return false, nil
	}
	return profile.Provider == "gateway" && profile.BaseURL == baseURL && len(profile.Key) > 0, nil
}
