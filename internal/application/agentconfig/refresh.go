package agentconfig

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
	catalog "omniproxy/internal/core/modelcatalog"
)

// RefreshCatalog 只更新已由本应用接管且没有外部改动的目录；不新接管任何客户端。
//
// 每个客户端的失败原因按稳定码记录在内存里，由 Inspect 交给界面：只有日志的话，
// 用户只会看到「模型没更新」，不知道该点哪里。
func (s *Service) RefreshCatalog(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result error
	issues := map[string]string{}
	for _, client := range Clients() {
		// Antigravity 的模型目录由远端 listSource 实时提供，配置文件只存默认模型；
		// 它还会持续改写 userSettings，不能按整文件摘要刷新。
		if client == "antigravity" {
			continue
		}
		var expected string
		err := s.db.QueryRowContext(ctx, `SELECT applied_hash FROM agent_backups WHERE client=?`, client).Scan(&expected)
		if err == nil {
			entries, catalogError := s.clientEntries(ctx, client)
			if catalogError == nil {
				entries, catalogError = s.selectedCatalog(ctx, client, entries)
			}
			var clientError error
			if catalogError != nil {
				clientError = catalogError
			} else {
				clientError = s.refreshClient(ctx, client, expected, entries)
			}
			if clientError != nil {
				result = errors.Join(result, clientError)
				if code := failureCode(clientError); code != "" {
					issues[client] = code
				}
			}
		}
		// 没有受管记录时不读取或修改该客户端；数据库错误仍需报告。
		if err != nil && err != sql.ErrNoRows {
			result = errors.Join(result, failure("backup_unavailable"))
			issues[client] = "backup_unavailable"
		}
	}
	s.setIssues(issues)
	return result
}

// failureCode 取本包稳定错误码（错误文本固定为 "agent: <code>"）；识别不了返回空串。
// errors.Join 的聚合错误按顺序取第一个可识别的码。
func failureCode(err error) string {
	if err == nil {
		return ""
	}
	var joined interface{ Unwrap() []error }
	if errors.As(err, &joined) {
		for _, inner := range joined.Unwrap() {
			if code := failureCode(inner); code != "" {
				return code
			}
		}
		return ""
	}
	code, found := strings.CutPrefix(err.Error(), "agent: ")
	if !found || code == "" || len(code) > 64 {
		return ""
	}
	for _, char := range code {
		if char >= 'a' && char <= 'z' || char == '_' {
			continue
		}
		return ""
	}
	return code
}

func (s *Service) setIssues(issues map[string]string) {
	s.issueMu.Lock()
	s.issues = issues
	s.issueMu.Unlock()
}

// issue 返回该客户端最近一次后台刷新的失败码；没有失败返回空串。
func (s *Service) issue(client string) string {
	s.issueMu.Lock()
	defer s.issueMu.Unlock()
	return s.issues[client]
}

// clearIssue 在客户端重新应用或恢复成功后清除失败记录。
func (s *Service) clearIssue(client string) {
	s.issueMu.Lock()
	delete(s.issues, client)
	s.issueMu.Unlock()
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
	if isChatClient(client) {
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
					model := upgradeClaudeDefault(entries, selected)
					if len(entries) > 0 && !slices.ContainsFunc(entries, func(entry catalog.Entry) bool { return entry.ID == model }) {
						model = entries[0].ID
					}
					env[key] = model
				}
			}
		}
		if err := subscriptionCompactSettings(config, entries); err != nil {
			return err
		}
		if err := claudeEffortSettings(config, entries); err != nil {
			return err
		}
	} else {
		selected := upgradeClaudeDefault(entries, existingModel(client, before.body))
		config["inferenceModels"] = catalog.DesktopModels(entries, selected)
		desktopDefaultEffortSettings(config, entries, selected)
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
	configBody, err := s.cleanCodexTransport(ctx, config.body)
	if err != nil {
		return err
	}
	defer clear(configBody)
	if values["model_catalog_json"] != s.catalogPath() {
		if digest(configBody, true) == config.revision {
			return nil
		}
		if err := writeAtomic(s.paths.Codex, configBody, config.revision); err != nil {
			return err
		}
		if err := s.commitFileVersions(ctx, "codex", digest(configBody, true), false, ""); err != nil {
			return errors.Join(err, rollbackFile(s.paths.Codex, digest(configBody, true), config))
		}
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
	catalogChanged := digest(body, true) != before.revision
	if catalogChanged {
		if err := writeAtomic(s.catalogPath(), body, before.revision); err != nil {
			return err
		}
	}
	rollbackCatalog := func() error {
		if catalogChanged {
			return rollbackFile(s.catalogPath(), digest(body, true), before)
		}
		return nil
	}
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
	if selected != original {
		provider, _ := values["model_provider"].(string)
		if provider == "" {
			provider = "openai"
		}
		fields := map[string]string{"model": quote(selected)}
		if !present {
			fields["model_reasoning_effort"] = ""
		}
		updated, patchErr := patchTOML(configBody, provider, fields)
		err = patchErr
		if err != nil {
			return errors.Join(err, rollbackCatalog())
		}
		clear(configBody)
		configBody = updated
		defer clear(updated)
	}
	configChanged := digest(configBody, true) != config.revision
	if !configChanged && !catalogChanged {
		return nil
	}
	if configChanged {
		if err := writeAtomic(s.paths.Codex, configBody, config.revision); err != nil {
			return errors.Join(err, rollbackCatalog())
		}
	}
	if err := s.commitFileVersions(ctx, "codex", digest(configBody, true), catalogChanged, digest(body, true)); err != nil {
		var configError error
		if configChanged {
			configError = rollbackFile(s.paths.Codex, digest(configBody, true), config)
		}
		return errors.Join(err, configError, rollbackCatalog())
	}
	return nil
}
