package agentconfig

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	catalog "omniproxy/internal/core/modelcatalog"
)

type preparation struct {
	plan   Plan
	path   string
	body   []byte
	exists bool
}

func (s *Service) entries(ctx context.Context) ([]catalog.Entry, error) {
	if s.catalog == nil {
		return nil, failure("models_unavailable")
	}
	if source, ok := s.catalog.(interface {
		Catalog(context.Context) ([]catalog.Entry, error)
	}); ok {
		entries, err := source.Catalog(ctx)
		if err != nil {
			return nil, failure("models_unavailable")
		}
		return catalog.Normalize(entries)
	}
	ids, err := s.catalog.AvailableModels(ctx)
	if err != nil {
		return nil, failure("models_unavailable")
	}
	entries := make([]catalog.Entry, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, catalog.Entry{ID: id, Name: id})
	}
	return catalog.Normalize(entries)
}

func supportsRemoteCatalog(version string) bool {
	parts := strings.Split(strings.TrimPrefix(strings.TrimSpace(version), "codex-cli "), ".")
	if len(parts) < 2 {
		return false
	}
	major, e1 := strconv.Atoi(parts[0])
	minor, e2 := strconv.Atoi(parts[1])
	return e1 == nil && e2 == nil && (major > 0 || minor >= 156)
}

func (s *Service) catalogPath() string {
	return filepath.Join(filepath.Dir(s.paths.Codex), "omniproxy-models.json")
}

func (s *Service) prepare(ctx context.Context, input Input) (preparation, error) {
	if len(input.Revision) != 64 {
		return preparation{}, failure("config_changed")
	}
	if input.CatalogRevision != "" && len(input.CatalogRevision) != 64 {
		return preparation{}, failure("catalog_changed")
	}
	if len(input.Model) > 128 {
		return preparation{}, failure("invalid_model")
	}
	path, err := s.path(input.Client)
	if err != nil {
		return preparation{}, err
	}
	body, exists, err := s.readClient(input.Client, path)
	if err != nil {
		return preparation{}, err
	}
	failed := func(err error) (preparation, error) { clear(body); return preparation{}, err }
	if input.Revision != digest(body, exists) {
		return failed(failure("config_changed"))
	}
	if s.port < 1 || s.port > 65535 {
		return failed(failure("invalid_config"))
	}
	entries, err := s.clientEntries(ctx, input.Client)
	if err != nil {
		return failed(err)
	}
	if len(entries) == 0 {
		return failed(failure("no_models"))
	}
	// 版本绑定完整源目录；用户可在该快照中调整所选子集和默认模型。
	serialized, err := json.Marshal(struct {
		Models          []catalog.Entry
		Client, Version string
		Port            int
		Config          string `json:",omitempty"`
	}{entries, input.Client, s.codexVersion, s.port, s.gatewayAgentIdentity(input.Client, path)})
	if err != nil {
		return failed(failure("models_unavailable"))
	}
	revision := digest(serialized, true)
	if input.CatalogRevision != "" && input.CatalogRevision != revision {
		return failed(failure("catalog_changed"))
	}
	entries, err = filterSelectedModels(entries, input.ModelIDs, true)
	if err != nil {
		return failed(err)
	}
	selected := input.Model
	if selected == "" {
		selected = existingModel(input.Client, body)
	}
	selected = upgradeClaudeDefault(entries, selected)
	valid := false
	for _, entry := range entries {
		if entry.ID == selected {
			valid = true
		}
	}
	if !valid {
		if input.Model != "" {
			return failed(failure("invalid_model"))
		}
		selected = entries[0].ID
	}
	mode := "client-default"
	if isGatewayAgent(input.Client) {
		mode = input.Client + "-model-list"
	}
	if input.Client == "opencode" {
		mode = "opencode-model-list"
	}
	if input.Client == "kilocode" {
		mode = "kilo-model-list"
	}
	if input.Client == "qwen-code" {
		mode = "qwen-model-list"
	}
	if input.Client == "mimo-code" {
		mode = "mimo-code-model-list"
	}
	if isHarness(input.Client) {
		mode = "harness-model-list"
	}
	provider := ""
	if input.Client == "codex" {
		var config map[string]any
		if len(body) > 0 && toml.Unmarshal(body, &config) != nil {
			return failed(failure("invalid_config"))
		}
		provider, _ = config["model_provider"].(string)
		if provider == "" {
			provider = "openai"
		}
		mode = "local-catalog"
		if input.ModelIDs == nil && provider != "openai" && supportsRemoteCatalog(s.codexVersion) {
			mode = "remote-catalog"
		}
		if err := s.checkCatalogFile(ctx); err != nil {
			return failed(err)
		}
	} else if input.Client == "claude" {
		mode = "cli-model-list"
	} else if input.Client == "claude-desktop" {
		mode = "desktop-model-list"
		desktop, err := s.prepareDesktopChange(ctx, true, false)
		if err != nil {
			return failed(err)
		}
		desktop.clear()
	}
	if input.Client == "codex" {
		if _, err := catalog.CodexJSON(entries); err != nil {
			return failed(failure("catalog_too_large"))
		}
	}
	plan := Plan{Mode: mode, DefaultModel: selected, Models: entries, CatalogRevision: revision}
	if effortClient(input.Client) {
		if err := validateEfforts(entries, input.Efforts); err != nil {
			return failed(err)
		}
		efforts, err := s.effortPlan(ctx, entries, input.Efforts)
		if err != nil {
			return failed(err)
		}
		plan.Efforts = efforts
	}
	// 摘要仅包含本次受管信息；绝不能回显原配置的授权或命令。
	plan.Summary = fmt.Sprintf("配置文件：%s\n可用模型／路由：%d 个\n默认模型：%s\n目录方式：%s\n接入：本地网关 API Key\n原提供商标识：%s（保留）\n保留官方登录及全部会话；应用前保存加密备份。", path, len(entries), selected, mode, provider)
	if input.Client == "claude-desktop" {
		plan.Summary += "\n启用第三方网关模式并选中 OmniProxy 本地配置；完全退出并重新启动 Claude 后进入 Code。"
	}
	if input.Client == "opencode" {
		plan.Summary = fmt.Sprintf("配置文件：%s\n接入：本地网关 Chat API\n可用模型／路由：%d 个\n默认模型：%s\nCLI 与 Desktop 共用全局配置；保留其他提供商及设置，应用前保存加密备份。", path, len(entries), selected)
	}
	if isHarness(input.Client) {
		plan.Summary = fmt.Sprintf("配置文件：%s\n接入：本地网关 Chat API\n可用模型／路由：%d 个\n凭据文件：%s\n只更新本 profile 的 OmniProxy 提供商及独立凭据引用；应用前保存加密备份。\n保存后在 Harness 中选择 OmniProxy 模型；其他 profile 与启动覆盖保持原样。", path, len(entries), s.paths.HarnessCredentials)
	}
	if input.Client == "kilocode" || input.Client == "mimo-code" || input.Client == "qwen-code" {
		plan.Summary = fmt.Sprintf("配置文件：%s\n接入：本地网关 Chat API\n可用模型／路由：%d 个\n默认模型：%s\n保留其他提供商、权限和会话；应用前保存加密备份。\n配置当前本机引擎；桌面宿主的独立数据目录、内联设置或启动覆盖可能优先。", path, len(entries), selected)
	}
	key := []byte("<本地接入 API Key>")
	if isGatewayAgent(input.Client) {
		plan.Summary = gatewayAgentSummary(input.Client, path, selected, len(entries))
	}
	patched, _, err := s.buildPlan(input.Client, body, key, plan)
	clear(patched)
	if err != nil {
		return failed(err)
	}
	return preparation{plan, path, body, exists}, nil
}

// PreviewPlan 验证配置和模型快照，应用时使用同一目录版本。
func (s *Service) PreviewPlan(ctx context.Context, input Input) (Plan, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	prepared, err := s.prepare(ctx, input)
	if err != nil {
		return Plan{}, err
	}
	defer clear(prepared.body)
	return prepared.plan, nil
}
