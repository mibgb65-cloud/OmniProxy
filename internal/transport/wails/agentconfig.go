package wails

import (
	"context"
	"errors"
	"omniproxy/internal/application/agentconfig"
)

func (s *AppService) AgentClients() ([]agentconfig.Item, error) {
	if s.agents == nil {
		return nil, errors.New("agent: unavailable")
	}
	s.refreshAgentCatalog()
	return s.agents.Inspect(context.Background())
}

// AgentConfiguredModels 仅返回实际受管目录的公开模型信息。
func (s *AppService) AgentConfiguredModels(client string) (agentconfig.ConfiguredCatalog, error) {
	if s.agents == nil {
		return agentconfig.ConfiguredCatalog{}, errors.New("agent: unavailable")
	}
	return s.agents.ConfiguredModels(context.Background(), client)
}

func (s *AppService) refreshAgentCatalog() {
	if s.agents == nil {
		return
	}
	if err := s.agents.RefreshCatalog(context.Background()); err != nil && s.log != nil {
		s.log.Warn("managed client catalog refresh failed", "module", "agentconfig", "code", "refresh_failed")
	}
}
func (s *AppService) AgentPreview(input agentconfig.Input) (string, error) {
	if s.agents == nil {
		return "", errors.New("agent: unavailable")
	}
	return s.agents.Preview(context.Background(), input)
}

// AgentPreviewPlan 返回受校验目录快照，供一键配置预览和应用使用。
func (s *AppService) AgentPreviewPlan(input agentconfig.Input) (agentconfig.Plan, error) {
	if s.agents == nil {
		return agentconfig.Plan{}, errors.New("agent: unavailable")
	}
	return s.agents.PreviewPlan(context.Background(), input)
}
func (s *AppService) AgentApply(input agentconfig.Input) ([]agentconfig.Item, error) {
	if s.agents == nil {
		return nil, errors.New("agent: unavailable")
	}
	return s.agents.Apply(context.Background(), input)
}
func (s *AppService) AgentRestore(client string) ([]agentconfig.Item, error) {
	if s.agents == nil {
		return nil, errors.New("agent: unavailable")
	}
	return s.agents.Restore(context.Background(), client)
}
func (s *AppService) AgentSetEnabled(client string, enabled bool) error {
	if s.agents == nil {
		return errors.New("agent: unavailable")
	}
	return s.agents.SetEnabled(context.Background(), client, enabled)
}

// AgentCopyKey 在桌面端复制本地接入 Key，不通过绑定返回完整凭据。
func (s *AppService) AgentCopyKey(client string) error {
	if s.agents == nil || s.copyText == nil {
		return errors.New("agent: unavailable")
	}
	valid := false
	for _, id := range agentconfig.Clients() {
		valid = valid || client == id
	}
	if !valid {
		return errors.New("agent: invalid_client")
	}
	key, err := s.agents.Key(context.Background(), client)
	if err != nil {
		return err
	}
	defer clear(key)
	s.clipboardMu.Lock()
	defer s.clipboardMu.Unlock()
	if !s.copyText(string(key)) {
		return errors.New("agent: clipboard_failed")
	}
	return nil
}
