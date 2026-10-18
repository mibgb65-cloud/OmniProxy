package agentconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"sort"
	"strings"
	"unicode"

	"github.com/pelletier/go-toml/v2"
	catalog "omniproxy/internal/core/modelcatalog"
)

// ModelSummary 仅包含客户端目录中的公开标识，不暴露配置和凭据。
type ModelSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ConfiguredCatalog 是实际受管模型目录，不表示客户端已重启或模型调用已验证。
type ConfiguredCatalog struct {
	Models       []ModelSummary `json:"models"`
	DefaultModel string         `json:"defaultModel"`
}

// ConfiguredModels 只读取已接管且版本匹配的文件，未接管时返回空列表。
func (s *Service) ConfiguredModels(ctx context.Context, client string) (ConfiguredCatalog, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := ConfiguredCatalog{Models: []ModelSummary{}}
	path, err := s.path(client)
	if err != nil {
		return result, err
	}
	var expected string
	err = s.db.QueryRowContext(ctx, `SELECT applied_hash FROM agent_backups WHERE client=?`, client).Scan(&expected)
	if err == sql.ErrNoRows {
		return result, nil
	}
	if err != nil {
		return result, failure("backup_unavailable")
	}
	before, err := s.snapshotClient(client, path)
	if err != nil {
		return result, err
	}
	defer clear(before.body)
	if before.revision != expected {
		return result, failure("config_changed")
	}
	entries, err := s.clientEntries(ctx, client)
	if err != nil {
		return result, err
	}
	result.DefaultModel = upgradeClaudeDefault(entries, existingModel(client, before.body))
	switch {
	case isGatewayAgent(client):
		result.Models, err = configuredGatewayAgent(client, before.body)
	case client == "codex":
		result.Models, err = s.configuredCodex(ctx, before.body, entries)
	case isHarness(client):
		result.Models, err = configuredHarness(before.body)
	case isChatClient(client):
		result.Models, err = configuredChat(client, before.body)
	default:
		result.Models, err = configuredClaude(client, before.body, entries)
	}
	if err != nil {
		return ConfiguredCatalog{}, err
	}
	if len(result.Models) > maxSelectedModels {
		return ConfiguredCatalog{}, failure("catalog_too_large")
	}
	seen := map[string]bool{}
	for _, model := range result.Models {
		if !catalog.ValidClaudeModel(model.ID) || len(model.Name) > 200 || strings.ContainsFunc(model.Name, unicode.IsControl) || seen[model.ID] {
			return ConfiguredCatalog{}, failure("invalid_config")
		}
		seen[model.ID] = true
	}
	sort.SliceStable(result.Models, func(i, j int) bool { return result.Models[i].ID < result.Models[j].ID })
	return result, nil
}

func (s *Service) configuredCodex(ctx context.Context, body []byte, entries []catalog.Entry) ([]ModelSummary, error) {
	var config map[string]any
	if toml.Unmarshal(body, &config) != nil {
		return nil, failure("invalid_config")
	}
	if config["model_catalog_json"] != s.catalogPath() {
		features, _ := config["features"].(map[string]any)
		if features["api_key_model_discovery"] != true {
			return []ModelSummary{}, nil
		}
		result := make([]ModelSummary, 0, len(entries))
		for _, entry := range entries {
			result = append(result, ModelSummary{entry.ID, entry.Name})
		}
		return result, nil
	}
	if err := s.checkCatalogFile(ctx); err != nil {
		return nil, err
	}
	catalogBody, _, err := read(s.catalogPath())
	if err != nil {
		return nil, err
	}
	defer clear(catalogBody)
	var stored struct {
		Models []struct {
			ID   string `json:"slug"`
			Name string `json:"display_name"`
		} `json:"models"`
	}
	if json.Unmarshal(catalogBody, &stored) != nil {
		return nil, failure("invalid_config")
	}
	result := make([]ModelSummary, 0, len(stored.Models))
	for _, model := range stored.Models {
		result = append(result, ModelSummary{model.ID, model.Name})
	}
	return result, nil
}

func configuredClaude(client string, body []byte, entries []catalog.Entry) ([]ModelSummary, error) {
	var stored struct {
		Picker struct {
			Options []struct {
				ID   string `json:"model"`
				Name string `json:"label"`
			} `json:"options"`
		} `json:"modelPicker"`
		Models []catalog.DesktopModel `json:"inferenceModels"`
	}
	if json.Unmarshal(body, &stored) != nil {
		return nil, failure("invalid_config")
	}
	result := []ModelSummary{}
	if client == "claude" {
		for _, model := range stored.Picker.Options {
			result = append(result, ModelSummary{model.ID, model.Name})
		}
		return result, nil
	}
	aliases := map[string]string{}
	for _, entry := range entries {
		aliases[catalog.DesktopAlias(entry.ID)] = entry.ID
	}
	for _, model := range stored.Models {
		id := model.Name
		if resolved, ok := aliases[id]; ok {
			id = resolved
		}
		result = append(result, ModelSummary{id, model.LabelOverride})
	}
	return result, nil
}
