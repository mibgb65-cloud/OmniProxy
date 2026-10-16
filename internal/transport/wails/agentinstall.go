package wails

import (
	"context"
	"errors"
	"time"

	"omniproxy/internal/application/agentinstall"
	runtime "omniproxy/internal/infrastructure/agentinstall"
)

func (s *AppService) initAgentInstaller() {
	if s.prof == nil {
		return
	}
	s.installations = agentinstall.New(runtime.New())
	s.installations.OnInstalled(func(id string, detected runtime.Detection) {
		if id == "codex-cli" && s.agents != nil && detected.Version != "" {
			s.agents.SetCodexVersion("codex-cli " + detected.Version)
		}
	})
}

// AgentInstallations 返回本机安装元数据和后台任务快照，不执行安装。
func (s *AppService) AgentInstallations() (agentinstall.Snapshot, error) {
	if s.installations == nil {
		return agentinstall.Snapshot{}, errors.New("agentinstall: unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.installations.List(ctx), nil
}

// AgentInstallationsRefresh 明确刷新检测结果，不读取客户端登录和会话。
func (s *AppService) AgentInstallationsRefresh() (agentinstall.Snapshot, error) {
	if s.installations == nil {
		return agentinstall.Snapshot{}, errors.New("agentinstall: unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.installations.Refresh(ctx), nil
}

// AgentInstall 只接受固定客户端 ID，返回后台任务状态。
func (s *AppService) AgentInstall(id string) (agentinstall.Snapshot, error) {
	if s.installations == nil {
		return agentinstall.Snapshot{}, errors.New("agentinstall: unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.installations.Start(ctx, id)
}

// AgentInstallationOpenOfficial 只打开固定客户端的官方说明，不接受任意 URL。
func (s *AppService) AgentInstallationOpenOfficial(id string) error {
	product, err := runtime.Find(id)
	if err != nil {
		return err
	}
	if s.openAgentURL == nil {
		return errors.New("agentinstall: unavailable")
	}
	if err := s.openAgentURL(product.OfficialURL); err != nil {
		return errors.New("agentinstall: official_unavailable")
	}
	return nil
}

// AgentOpenClaudeCode 打开官方 Code 编程入口，不附带任务、项目路径或凭据。
func (s *AppService) AgentOpenClaudeCode() error {
	if s.openAgentURL == nil || s.agents == nil {
		return errors.New("agentinstall: unavailable")
	}
	active, err := s.agents.DesktopGatewayActive()
	if err != nil || !active {
		return errors.New("agentinstall: activation_required")
	}
	if s.openAgentURL("claude://code/new") != nil {
		return errors.New("agentinstall: launch_failed")
	}
	return nil
}

// AgentOpenClient 打开固定的 OpenCode 或 Harness 本机入口，不接受路径或命令参数。
func (s *AppService) AgentOpenClient(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return runtime.Launch(ctx, id)
}
