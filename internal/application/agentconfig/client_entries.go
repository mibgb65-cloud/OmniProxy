package agentconfig

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pelletier/go-toml/v2"
	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/provider/river"
)

// clientEntries 仅排除已经确认不适合当前客户端的模型，未知能力保持未知。
func (s *Service) clientEntries(ctx context.Context, client string) ([]catalog.Entry, error) {
	entries, err := s.entries(ctx)
	if err != nil {
		return nil, err
	}
	if isChatClient(client) {
		return gatewayEntries(entries), nil
	}
	protocol := "messages"
	if client == "codex" {
		protocol = "responses"
	}
	entries = catalog.ForProtocol(entries, protocol)
	if protocol == "messages" {
		return catalog.ClaudeEntries(entries)
	}
	return entries, nil
}

func upgradeClaudeDefault(entries []catalog.Entry, selected string) string {
	// 升级已确认目标的旧默认，不因普通 ID 消失而切到其他模型；Codex 目录不生成该变体。
	for _, entry := range entries {
		if entry.ID == selected {
			return selected
		}
	}
	// 旧 River 原始 ID 和 Desktop 摘要继续指向归一后的同款型号，不能因菜单去重而切换默认模型。
	for _, original := range river.Models() {
		for _, suffix := range []string{"", catalog.ClaudeLongContextSuffix} {
			old := original + suffix
			alias := catalog.DesktopAlias(old)
			if selected != old && selected != alias && selected != strings.TrimSuffix(alias, catalog.ClaudeLongContextSuffix) {
				continue
			}
			for _, entry := range entries {
				if entry.ID == river.PublicModel(original)+suffix {
					return entry.ID
				}
			}
		}
	}
	for _, entry := range entries {
		alias := catalog.DesktopAlias(entry.ID)
		if selected == alias || selected == strings.TrimSuffix(alias, catalog.ClaudeLongContextSuffix) {
			return entry.ID
		}
		if entry.Metadata.ContextVerified && entry.ID == selected+catalog.ClaudeLongContextSuffix {
			return entry.ID
		}
		if entry.Metadata.ContextVerified && strings.HasSuffix(entry.ID, catalog.ClaudeLongContextSuffix) && selected == catalog.DesktopAlias(strings.TrimSuffix(entry.ID, catalog.ClaudeLongContextSuffix)) {
			return entry.ID
		}
	}
	return selected
}

func existingModel(client string, body []byte) string {
	if isGatewayAgent(client) {
		return gatewayAgentModel(client, body)
	}
	if isCompatibleClient(client) {
		return openCodeModel(body)
	}
	if client == "qwen-code" {
		return qwenCodeModel(body)
	}
	var config map[string]any
	if client == "codex" {
		if toml.Unmarshal(body, &config) == nil {
			model, _ := config["model"].(string)
			return model
		}
	} else if client == "claude" {
		if json.Unmarshal(body, &config) == nil {
			if env, ok := config["env"].(map[string]any); ok {
				model, _ := env["ANTHROPIC_MODEL"].(string)
				return model
			}
		}
	} else if client == "claude-desktop" {
		var desktop struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"inferenceModels"`
		}
		if json.Unmarshal(body, &desktop) == nil && len(desktop.Models) > 0 {
			return desktop.Models[0].Name
		}
	}
	return ""
}
