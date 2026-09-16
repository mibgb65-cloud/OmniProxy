package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"omniproxy/internal/claudedesktop"
	"omniproxy/internal/clientconfig"
	"omniproxy/internal/logs"
)

const (
	mimoModel            = "mimo-v2.5-pro"
	mimoLongContextModel = "mimo-v2.5-pro[1m]"
	mimoStandardModel    = "mimo-v2.5"
	deepSeekProModel     = "deepseek-v4-pro"
	deepSeekProLongModel = "deepseek-v4-pro[1m]"
	deepSeekProLegacy    = deepSeekProLongModel
	deepSeekFastModel    = "deepseek-v4-flash"
	kimiCodingModel      = "kimi-for-coding"
	zhipuGLMModel        = "glm-5.1"
	claudeDefaultModel   = "default"
	claudeSonnetModel    = "sonnet"
	claudeOpusModel      = "opus"
	claudeHaikuModel     = "haiku"
	anyRouterClaudeModel = "claude-opus-4-5-20251101"
	zoClaudeModel        = "claude-opus-4-7"
	zoClaudeSonnetModel  = "claude-sonnet-4-6"
	premClaudeModel      = "deepseek-v4-pro"
	atriaClaudeModel     = "Atria-Dawn-Preview"
	omniProxyMimoAuth    = "omniproxy"
	maxClaudeModels      = 4
)

type mimoConfigureResult struct {
	ConfigPath    string   `json:"configPath,omitempty"`
	SettingsPath  string   `json:"settingsPath,omitempty"`
	ClaudePath    string   `json:"claudePath,omitempty"`
	BackupPath    string   `json:"backupPath,omitempty"`
	BaseURL       string   `json:"baseUrl,omitempty"`
	Model         string   `json:"model,omitempty"`
	Models        []string `json:"models,omitempty"`
	EnvConfigured bool     `json:"envConfigured,omitempty"`
	Message       string   `json:"message"`
}

type claudeModelsConfigureRequest struct {
	Models []string `json:"models"`
}

type claudeModelTarget struct {
	Model       string
	Name        string
	Description string
	LogMessage  string
	Message     string
}

var (
	claudeMimoTarget = claudeModelTarget{
		Model:       mimoLongContextModel,
		Name:        "MiMo-V2.5-Pro [1m]",
		Description: "Xiaomi MiMo-V2.5-Pro 1M context routed through OmniProxy",
		LogMessage:  "mimo claude configured",
		Message:     "Claude Code 已配置为通过 OmniProxy 使用 Xiaomi MiMo",
	}
	claudeDefaultTarget = claudeModelTarget{
		Model:       claudeDefaultModel,
		Name:        "Claude Default",
		Description: "Claude Code official default model routed through OmniProxy",
		LogMessage:  "official claude default configured",
		Message:     "Claude Code 已配置为通过 OmniProxy 使用 Claude 官方默认模型",
	}
	claudeSonnetTarget = claudeModelTarget{
		Model:       claudeSonnetModel,
		Name:        "Claude Sonnet",
		Description: "Claude Code official Sonnet alias routed through OmniProxy",
	}
	claudeOpusTarget = claudeModelTarget{
		Model:       claudeOpusModel,
		Name:        "Claude Opus",
		Description: "Claude Code official Opus alias routed through OmniProxy",
	}
	claudeHaikuTarget = claudeModelTarget{
		Model:       claudeHaikuModel,
		Name:        "Claude Haiku",
		Description: "Claude Code official Haiku alias routed through OmniProxy",
	}
	claudeDeepSeekTarget = claudeModelTarget{
		Model:       deepSeekProModel,
		Name:        "DeepSeek V4 Pro",
		Description: "DeepSeek V4 Pro routed through OmniProxy",
		LogMessage:  "deepseek claude configured",
		Message:     "Claude Code 已配置为通过 OmniProxy 使用 DeepSeek",
	}
	claudeDeepSeekLongTarget = claudeModelTarget{
		Model:       deepSeekProLongModel,
		Name:        "DeepSeek V4 Pro [1m]",
		Description: "DeepSeek V4 Pro 1M context routed through OmniProxy",
		LogMessage:  "deepseek claude 1m configured",
		Message:     "Claude Code 已配置为通过 OmniProxy 使用 DeepSeek 1M 上下文模型",
	}
	claudeKimiTarget = claudeModelTarget{
		Model:       kimiCodingModel,
		Name:        "Kimi for Coding",
		Description: "Kimi Code routed through OmniProxy",
		LogMessage:  "kimi claude configured",
		Message:     "Claude Code 已配置为通过 OmniProxy 使用 Kimi",
	}
	claudeZhipuTarget = claudeModelTarget{
		Model:       zhipuGLMModel,
		Name:        "GLM-5.1",
		Description: "Zhipu GLM-5.1 routed through OmniProxy",
		LogMessage:  "zhipu claude configured",
		Message:     "Claude Code 已配置为通过 OmniProxy 使用 Zhipu GLM",
	}
	claudeZoTarget = claudeModelTarget{
		Model:       zoClaudeModel,
		Name:        "Zo Claude Opus 4.7",
		Description: "Claude Opus 4.7 routed through OmniProxy Zo Computer",
		LogMessage:  "zo claude configured",
		Message:     "Claude Code 已配置为通过 OmniProxy 使用 Zo Computer",
	}
	claudeZoSonnetTarget = claudeModelTarget{
		Model:       zoClaudeSonnetModel,
		Name:        "Zo Claude Sonnet 4.6",
		Description: "Claude Sonnet 4.6 routed through OmniProxy Zo Computer",
	}
	claudeAtriaTarget = claudeModelTarget{
		Model:       atriaClaudeModel,
		Name:        "Atria Dawn Preview",
		Description: "Atria Atria-Dawn-Preview routed through OmniProxy",
		LogMessage:  "atria claude configured",
		Message:     "Claude Code 已配置为通过 OmniProxy 使用 Atria",
	}
)

type claudeModelSelectionError struct {
	message string
}

func (e *claudeModelSelectionError) Error() string {
	return e.message
}

func (a *appServer) configureClaudeModels(req claudeModelsConfigureRequest) (mimoConfigureResult, error) {
	return a.configureSelectedClaudeModels(req, "selected claude models configured", func(targets []claudeModelTarget) string {
		return fmt.Sprintf("Claude Code 已配置 %d 个模型：%s", len(targets), strings.Join(claudeModelNames(targets), "、"))
	})
}

func (a *appServer) configureClaudeDesktopModels(req claudeModelsConfigureRequest) (mimoConfigureResult, error) {
	targets, err := normalizeClaudeModelTargets(req.Models)
	if err != nil {
		return mimoConfigureResult{}, err
	}
	a.mu.Lock()
	baseURL := claudedesktop.GatewayBaseURL(a.cfg.ProxyPort)
	a.mu.Unlock()

	paths, err := claudedesktop.CurrentPaths()
	if err != nil {
		return mimoConfigureResult{}, err
	}
	routes := claudeDesktopRoutesForTargets(targets)
	if err := writeClaudeDesktopProfile(paths, baseURL, routes); err != nil {
		return mimoConfigureResult{}, err
	}

	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "claude desktop models configured"})
	return mimoConfigureResult{
		ConfigPath: paths.ProfilePath,
		BackupPath: paths.MetaPath,
		BaseURL:    baseURL,
		Model:      targets[0].Model,
		Models:     claudeModelIDs(targets),
		Message:    fmt.Sprintf("Claude Code Desktop 已配置 %d 个模型：%s，请完全退出并重启 Claude Desktop", len(targets), strings.Join(claudeModelNames(targets), "、")),
	}, nil
}

func (a *appServer) configureSelectedClaudeModels(req claudeModelsConfigureRequest, logMessage string, message func([]claudeModelTarget) string) (mimoConfigureResult, error) {
	targets, err := normalizeClaudeModelTargets(req.Models)
	if err != nil {
		return mimoConfigureResult{}, err
	}

	models := claudeModelIDs(targets)
	target := claudeModelTarget{
		Model:      targets[0].Model,
		LogMessage: logMessage,
		Message:    message(targets),
	}
	result, err := a.configureClaudeWithWriter(target, func(path string, baseURL string) error {
		return writeSelectedClaudeSettings(path, baseURL, targets)
	})
	if err != nil {
		return mimoConfigureResult{}, err
	}
	result.Models = models
	return result, nil
}

func (a *appServer) configureClaudeWithWriter(target claudeModelTarget, writeSettings func(string, string) error) (mimoConfigureResult, error) {
	a.mu.Lock()
	baseURL := fmt.Sprintf("http://127.0.0.1:%d/anthropic-router", a.cfg.ProxyPort)
	a.mu.Unlock()

	return a.configureClaudeWithBaseURL(target, baseURL, writeSettings)
}

func (a *appServer) configureClaudeWithBaseURL(target claudeModelTarget, baseURL string, writeSettings func(string, string) error) (mimoConfigureResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return mimoConfigureResult{}, err
	}

	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		return mimoConfigureResult{}, err
	}

	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := writeSettings(settingsPath, baseURL); err != nil {
		return mimoConfigureResult{}, err
	}

	claudePath := filepath.Join(home, ".claude.json")
	if err := writeMimoClaudeOnboarding(claudePath); err != nil {
		return mimoConfigureResult{}, err
	}

	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: target.LogMessage})
	return mimoConfigureResult{
		SettingsPath: settingsPath,
		ClaudePath:   claudePath,
		BackupPath:   settingsPath + ".omniproxy.bak",
		BaseURL:      baseURL,
		Model:        target.Model,
		Message:      target.Message,
	}, nil
}

func (a *appServer) restoreClaudeConfig() (mimoConfigureResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return mimoConfigureResult{}, err
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := clientconfig.RestoreBackup(settingsPath, settingsPath+".omniproxy.bak"); err != nil {
		return mimoConfigureResult{}, err
	}
	claudePath := filepath.Join(home, ".claude.json")
	if err := clientconfig.RestoreBackup(claudePath, claudePath+".omniproxy.bak"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return mimoConfigureResult{}, err
	}

	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "claude config restored"})
	return mimoConfigureResult{
		SettingsPath: settingsPath,
		ClaudePath:   claudePath,
		BackupPath:   settingsPath + ".omniproxy.bak",
		Message:      "Claude Code 配置已恢复",
	}, nil
}

func (a *appServer) restoreClaudeDesktopConfig() (mimoConfigureResult, error) {
	paths, err := claudedesktop.CurrentPaths()
	if err != nil {
		return mimoConfigureResult{}, err
	}
	if err := writeClaudeDesktopDeploymentMode(paths.NormalConfigPath, "1p"); err != nil {
		return mimoConfigureResult{}, err
	}
	if err := writeClaudeDesktopDeploymentMode(paths.ThreePConfigPath, "1p"); err != nil {
		return mimoConfigureResult{}, err
	}
	if err := removeFileIfExists(paths.ProfilePath); err != nil {
		return mimoConfigureResult{}, err
	}
	if err := removeFileIfExists(paths.RoutesPath); err != nil {
		return mimoConfigureResult{}, err
	}
	if err := writeClaudeDesktopMeta(paths.MetaPath, false); err != nil {
		return mimoConfigureResult{}, err
	}

	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "claude desktop config restored"})
	return mimoConfigureResult{
		ConfigPath: paths.ProfilePath,
		BackupPath: paths.MetaPath,
		Message:    "Claude Desktop 配置已恢复为官方模式",
	}, nil
}

func writeSelectedClaudeSettings(path string, baseURL string, targets []claudeModelTarget) error {
	if len(targets) == 0 {
		return &claudeModelSelectionError{message: "至少选择一个 Claude Code 模型"}
	}

	data, err := clientconfig.ReadJSONObject(path)
	if err != nil {
		return err
	}
	if err := clientconfig.BackupFile(path, path+".omniproxy.bak", []byte("{}\n")); err != nil {
		return err
	}

	env := cleanClaudeEnv(data)
	env["ANTHROPIC_BASE_URL"] = baseURL
	env["ANTHROPIC_AUTH_TOKEN"] = omniProxyMimoAuth
	env["ANTHROPIC_MODEL"] = targets[0].Model
	opusTarget := claudeTargetForAlias(targets, claudeOpusModel, 0)
	sonnetTarget := claudeTargetForAlias(targets, claudeSonnetModel, 1)
	haikuTarget := claudeTargetForAlias(targets, claudeHaikuModel, 2)
	setClaudeModelGroup(env, "ANTHROPIC_DEFAULT_OPUS_MODEL", opusTarget)
	setClaudeModelGroup(env, "ANTHROPIC_DEFAULT_SONNET_MODEL", sonnetTarget)
	setClaudeModelGroup(env, "ANTHROPIC_DEFAULT_HAIKU_MODEL", haikuTarget)
	env["CLAUDE_CODE_SUBAGENT_MODEL"] = sonnetTarget.Model
	if len(targets) == maxClaudeModels && !isClaudeOfficialAlias(targets[maxClaudeModels-1].Model) {
		setClaudeModelGroup(env, "ANTHROPIC_CUSTOM_MODEL_OPTION", targets[maxClaudeModels-1])
	}
	if claudeTargetsIncludeDeepSeekPro(targets) {
		env["CLAUDE_CODE_EFFORT_LEVEL"] = "max"
	}
	data["env"] = env
	return clientconfig.WriteJSONObject(path, data)
}

func claudeTargetForAlias(targets []claudeModelTarget, alias string, fallbackIndex int) claudeModelTarget {
	for _, target := range targets {
		if strings.EqualFold(target.Model, alias) {
			return target
		}
	}
	return claudeTargetAt(targets, fallbackIndex)
}

func isClaudeOfficialAlias(model string) bool {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case claudeDefaultModel, claudeSonnetModel, claudeOpusModel, claudeHaikuModel:
		return true
	default:
		return false
	}
}

func setClaudeModelGroup(env map[string]any, key string, target claudeModelTarget) {
	env[key] = target.Model
	env[key+"_NAME"] = target.Name
	env[key+"_DESCRIPTION"] = target.Description
}

func claudeTargetAt(targets []claudeModelTarget, index int) claudeModelTarget {
	if index < len(targets) {
		return targets[index]
	}
	return targets[len(targets)-1]
}

func claudeTargetsInclude(targets []claudeModelTarget, model string) bool {
	for _, target := range targets {
		if strings.EqualFold(target.Model, model) {
			return true
		}
	}
	return false
}

func claudeTargetsIncludeDeepSeekPro(targets []claudeModelTarget) bool {
	return claudeTargetsInclude(targets, deepSeekProModel) || claudeTargetsInclude(targets, deepSeekProLongModel)
}

func normalizeClaudeModelTargets(models []string) ([]claudeModelTarget, error) {
	if len(models) == 0 {
		return nil, &claudeModelSelectionError{message: "至少选择一个 Claude Code 模型"}
	}

	targetsByModel := claudeSelectableTargetsByModel()
	seen := map[string]bool{}
	targets := make([]claudeModelTarget, 0, len(models))
	for _, raw := range models {
		model := normalizeClaudeModelID(raw)
		if model == "" {
			continue
		}
		target, ok := targetsByModel[model]
		if !ok {
			return nil, &claudeModelSelectionError{message: fmt.Sprintf("不支持的 Claude Code 模型：%s", strings.TrimSpace(raw))}
		}
		if seen[target.Model] {
			continue
		}
		targets = append(targets, target)
		seen[target.Model] = true
		if len(targets) > maxClaudeModels {
			return nil, &claudeModelSelectionError{message: fmt.Sprintf("Claude Code 最多选择 %d 个模型", maxClaudeModels)}
		}
	}
	if len(targets) == 0 {
		return nil, &claudeModelSelectionError{message: "至少选择一个 Claude Code 模型"}
	}
	return targets, nil
}

func normalizeClaudeModelID(model string) string {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "deepseek-v4-pro", "deepseek-4-pro":
		return deepSeekProModel
	case "deepseek-v4-pro[1m]", "deepseek-v4-pro-1m", "deepseek-4-pro-1m":
		return deepSeekProLongModel
	case "mimo-v2.5-pro-1m", "mimo-2.5-pro-1m":
		return mimoLongContextModel
	case "mimo-2.5-pro":
		return mimoModel
	case "mimo-2.5":
		return mimoStandardModel
	default:
		return strings.ToLower(strings.TrimSpace(model))
	}
}

func claudeSelectableTargetsByModel() map[string]claudeModelTarget {
	targets := map[string]claudeModelTarget{}
	for _, target := range claudeSelectableTargets() {
		targets[strings.ToLower(target.Model)] = target
	}
	return targets
}

func claudeSelectableTargets() []claudeModelTarget {
	return []claudeModelTarget{
		claudeDefaultTarget,
		claudeSonnetTarget,
		claudeOpusTarget,
		claudeHaikuTarget,
		claudeDeepSeekTarget,
		claudeDeepSeekLongTarget,
		{
			Model:       deepSeekFastModel,
			Name:        "DeepSeek V4 Flash",
			Description: "DeepSeek V4 Flash routed through OmniProxy",
		},
		claudeMimoTarget,
		{
			Model:       mimoModel,
			Name:        "MiMo-V2.5-Pro",
			Description: "Xiaomi MiMo-V2.5-Pro routed through OmniProxy",
		},
		{
			Model:       mimoStandardModel,
			Name:        "MiMo-V2.5",
			Description: "Xiaomi MiMo-V2.5 routed through OmniProxy",
		},
		claudeKimiTarget,
		claudeZhipuTarget,
		claudeZoTarget,
		claudeZoSonnetTarget,
		claudeAtriaTarget,
	}
}

func claudeModelIDs(targets []claudeModelTarget) []string {
	models := make([]string, 0, len(targets))
	for _, target := range targets {
		models = append(models, target.Model)
	}
	return models
}

func claudeModelNames(targets []claudeModelTarget) []string {
	names := make([]string, 0, len(targets))
	for _, target := range targets {
		names = append(names, target.Name)
	}
	return names
}

func claudeDesktopRoutesForTargets(targets []claudeModelTarget) []claudedesktop.ModelRoute {
	routeIDs := []string{
		"claude-sonnet-4-6",
		"claude-opus-4-7",
		"claude-haiku-4-5",
		"claude-sonnet-4-6-r2",
	}
	routes := make([]claudedesktop.ModelRoute, 0, len(targets))
	for index, target := range targets {
		if index >= len(routeIDs) {
			break
		}
		label := strings.TrimSpace(target.Name)
		if label == "" {
			label = target.Model
		}
		routes = append(routes, claudedesktop.ModelRoute{
			RouteID:       routeIDs[index],
			UpstreamModel: target.Model,
			LabelOverride: label,
			Supports1M:    claudeTargetSupports1M(target),
		})
	}
	return routes
}

func claudeTargetSupports1M(target claudeModelTarget) bool {
	model := strings.ToLower(strings.TrimSpace(target.Model))
	return strings.Contains(model, "[1m]")
}

func writeClaudeDesktopProfile(paths claudedesktop.Paths, baseURL string, routes []claudedesktop.ModelRoute) error {
	if err := writeClaudeDesktopDeploymentMode(paths.NormalConfigPath, "3p"); err != nil {
		return err
	}
	if err := writeClaudeDesktopDeploymentMode(paths.ThreePConfigPath, "3p"); err != nil {
		return err
	}
	if err := clientconfig.WriteJSONObject(paths.ProfilePath, claudedesktop.BuildGatewayProfile(baseURL, routes)); err != nil {
		return err
	}
	if err := claudedesktop.WriteRoutes(paths.RoutesPath, routes); err != nil {
		return err
	}
	return writeClaudeDesktopMeta(paths.MetaPath, true)
}

func writeClaudeDesktopDeploymentMode(path string, mode string) error {
	data, err := clientconfig.ReadJSONObject(path)
	if err != nil {
		return err
	}
	data["deploymentMode"] = mode
	return clientconfig.WriteJSONObject(path, data)
}

func writeClaudeDesktopMeta(path string, apply bool) error {
	data, err := clientconfig.ReadJSONObject(path)
	if err != nil {
		return err
	}
	entries := make([]any, 0)
	if existing, ok := data["entries"].([]any); ok {
		for _, entry := range existing {
			item, ok := entry.(map[string]any)
			if ok && item["id"] == claudedesktop.ProfileID {
				continue
			}
			entries = append(entries, entry)
		}
	}
	if apply {
		entries = append(entries, map[string]any{
			"id":   claudedesktop.ProfileID,
			"name": claudedesktop.ProfileName,
		})
		data["appliedId"] = claudedesktop.ProfileID
	} else if data["appliedId"] == claudedesktop.ProfileID {
		delete(data, "appliedId")
	}
	data["entries"] = entries
	return clientconfig.WriteJSONObject(path, data)
}
