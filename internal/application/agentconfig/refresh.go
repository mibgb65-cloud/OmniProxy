package agentconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/pelletier/go-toml/v2"
	catalog "omniproxy/internal/core/modelcatalog"
)

// RefreshCatalog 只更新已由本应用接管且没有外部改动的目录；不新接管任何客户端。
func (s *Service) RefreshCatalog(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result error
	for _, client := range Clients() {
		var expected string
		err := s.db.QueryRowContext(ctx, `SELECT applied_hash FROM agent_backups WHERE client=?`, client).Scan(&expected)
		if err == nil {
			entries, catalogError := s.clientEntries(ctx, client)
			if catalogError != nil {
				result = errors.Join(result, catalogError)
			} else {
				result = errors.Join(result, s.refreshClient(ctx, client, expected, entries))
			}
		}
		// 没有受管记录时不读取或修改该客户端；数据库错误仍需报告。
		if err != nil && err != sql.ErrNoRows {
			result = errors.Join(result, failure("backup_unavailable"))
		}
	}
	return result
}

func (s *Service) refreshClient(ctx context.Context, client, expected string, entries []catalog.Entry) error {
	path, err := s.path(client)
	if err != nil {
		return err
	}
	before, err := s.snapshotClient(client, path)
	if err != nil {
		return err
	}
	defer clear(before.body)
	if before.revision != expected {
		return failure("config_changed")
	}
	if client == "opencode" || isHarness(client) {
		return s.refreshGatewayClient(ctx, client, path, before, entries)
	}
	if client == "codex" {
		return s.refreshCodex(ctx, entries, before)
	}
	var config map[string]any
	if json.Unmarshal(before.body, &config) != nil {
		return failure("invalid_config")
	}
	if client == "claude" {
		claudePickerSettings(config, entries)
		if env, ok := config["env"].(map[string]any); ok {
			for _, key := range []string{"ANTHROPIC_MODEL", "ANTHROPIC_DEFAULT_OPUS_MODEL", "ANTHROPIC_DEFAULT_SONNET_MODEL", "ANTHROPIC_DEFAULT_HAIKU_MODEL"} {
				if selected, ok := env[key].(string); ok {
					env[key] = upgradeClaudeDefault(entries, selected)
				}
			}
		}
		if err := subscriptionCompactSettings(config, entries); err != nil {
			return err
		}
	} else {
		selected := upgradeClaudeDefault(entries, existingModel(client, before.body))
		config["inferenceModels"] = catalog.DesktopModels(entries, selected)
	}
	body, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return failure("invalid_config")
	}
	body = append(body, '\n')
	defer clear(body)
	if digest(body, true) == before.revision {
		return nil
	}
	if err := writeAtomic(path, body, before.revision); err != nil {
		return err
	}
	if err := s.commitFileVersions(ctx, client, digest(body, true), false, ""); err != nil {
		return errors.Join(err, rollbackFile(path, digest(body, true), before))
	}
	return nil
}

func (s *Service) refreshCodex(ctx context.Context, entries []catalog.Entry, config fileSnapshot) error {
	var values map[string]any
	if toml.Unmarshal(config.body, &values) != nil {
		return failure("invalid_config")
	}
	if values["model_catalog_json"] != s.catalogPath() {
		return nil
	}
	if err := s.checkCatalogFile(ctx); err != nil {
		return err
	}
	before, err := snapshotFile(s.catalogPath())
	if err != nil {
		return err
	}
	defer clear(before.body)
	body, err := catalog.CodexJSON(entries)
	if err != nil {
		return failure("models_unavailable")
	}
	if digest(body, true) == before.revision {
		return nil
	}
	if err := writeAtomic(s.catalogPath(), body, before.revision); err != nil {
		return err
	}
	configBody := config.body
	original, _ := values["model"].(string)
	selected := upgradeClaudeDefault(entries, original)
	present := false
	for _, entry := range entries {
		if entry.ID == selected {
			present = true
		}
	}
	if !present && len(entries) > 0 {
		selected = entries[0].ID
	}
	configChanged := selected != original
	if configChanged {
		provider, _ := values["model_provider"].(string)
		if provider == "" {
			provider = "openai"
		}
		fields := map[string]string{"model": quote(selected)}
		if !present {
			fields["model_reasoning_effort"] = ""
		}
		configBody, err = patchTOML(config.body, provider, fields)
		if err == nil {
			err = writeAtomic(s.paths.Codex, configBody, config.revision)
		}
		if err != nil {
			return errors.Join(err, rollbackFile(s.catalogPath(), digest(body, true), before))
		}
		defer clear(configBody)
	}
	if err := s.commitFileVersions(ctx, "codex", digest(configBody, true), true, digest(body, true)); err != nil {
		var configError error
		if configChanged {
			configError = rollbackFile(s.paths.Codex, digest(configBody, true), config)
		}
		return errors.Join(err, configError, rollbackFile(s.catalogPath(), digest(body, true), before))
	}
	return nil
}
