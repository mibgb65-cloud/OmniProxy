package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"omniproxy/internal/logs"
	"omniproxy/internal/proxy"
	"omniproxy/internal/token"
)

const (
	codexLocalAPIKey = "sk-omniproxy-local-sub2api"
	maxCodexModels   = 4
)

var defaultCodexModels = []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna"}

type codexConfigureResult struct {
	ConfigPath       string   `json:"configPath"`
	AuthPath         string   `json:"authPath"`
	BackupPath       string   `json:"backupPath"`
	BaseURL          string   `json:"baseUrl"`
	Model            string   `json:"model,omitempty"`
	Models           []string `json:"models,omitempty"`
	ProfilePaths     []string `json:"profilePaths,omitempty"`
	ImportedAuth     bool     `json:"importedAuth"`
	AuthAlreadyAdded bool     `json:"authAlreadyAdded"`
	AuthUpdated      bool     `json:"authUpdated"`
	Message          string   `json:"message"`
}

type codexConfigureRequest struct {
	Model  string   `json:"model"`
	Models []string `json:"models"`
}

func (a *appServer) configureCodex(requests ...codexConfigureRequest) (codexConfigureResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return codexConfigureResult{}, err
	}

	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()

	codexDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		return codexConfigureResult{}, err
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d/codex/v1", cfg.ProxyPort)
	var request codexConfigureRequest
	if len(requests) > 0 {
		request = requests[0]
	}
	models := normalizeCodexConfigureModels(request)
	model := models[0]
	configPath := filepath.Join(codexDir, "config.toml")
	result := codexConfigureResult{
		ConfigPath: configPath,
		AuthPath:   filepath.Join(codexDir, "auth.json"),
		BackupPath: configPath + ".omniproxy.bak",
		BaseURL:    baseURL,
		Model:      model,
		Models:     models,
	}

	preferredAuthID := ""
	authBytes, err := os.ReadFile(result.AuthPath)
	authValue := strings.TrimSpace(string(authBytes))
	switch {
	case err == nil:
		fields, ok := token.ExtractCodexAuthFields(authValue)
		if !ok || strings.TrimSpace(fields.Email) == "" || !fields.HasSupportedToken() {
			a.logs.Add(logs.Entry{Level: logs.LevelWarn, Message: "codex auth skipped: auth.json is not importable Codex auth"})
			break
		}
		authValue, err = a.refreshCodexAuthValueIfNeeded(context.Background(), authValue)
		if err != nil {
			a.logs.Add(logs.Entry{Level: logs.LevelWarn, Message: fmt.Sprintf("codex auth skipped: auth.json is expired or cannot be refreshed: %v", err)})
			break
		}
		fields, ok = token.ExtractCodexAuthFields(authValue)
		if !ok || strings.TrimSpace(fields.Email) == "" || !fields.HasSupportedToken() {
			a.logs.Add(logs.Entry{Level: logs.LevelWarn, Message: "codex auth skipped: refreshed auth.json is not importable Codex auth"})
			break
		}
		req := token.UpsertRequest{
			Provider:       token.ProviderOpenAI,
			CredentialType: token.CredentialTypeCodexAuthJSON,
			TokenValue:     authValue,
		}
		item, addErr := a.tokens.Add(req)
		if addErr == nil {
			result.ImportedAuth = true
			preferredAuthID = item.ID
			a.logs.Add(logs.Entry{Level: logs.LevelInfo, TokenName: a.tokenDisplayName(item), Message: "codex auth imported"})
		} else if errors.Is(addErr, token.ErrDuplicateName) {
			existing, findErr := a.tokens.FindCodexAuth(token.ProviderOpenAI, fields)
			if findErr != nil {
				return codexConfigureResult{}, addErr
			}
			if strings.TrimSpace(existing.TokenValue) == authValue {
				result.AuthAlreadyAdded = true
				preferredAuthID = existing.ID
				a.logs.Add(logs.Entry{Level: logs.LevelInfo, TokenName: a.tokenDisplayName(existing), Message: "codex auth already imported"})
				break
			}
			updated, updateErr := a.tokens.Update(existing.ID, req)
			if updateErr != nil {
				return codexConfigureResult{}, updateErr
			}
			result.AuthUpdated = true
			preferredAuthID = updated.ID
			a.logs.Add(logs.Entry{Level: logs.LevelInfo, TokenName: a.tokenDisplayName(updated), Message: "codex auth synced"})
		} else {
			return codexConfigureResult{}, addErr
		}
	case errors.Is(err, os.ErrNotExist):
	default:
		return codexConfigureResult{}, err
	}

	clientAuth, hasClientAuth := a.selectFreshCodexClientAuth(context.Background(), preferredAuthID)
	loginMethod := "chatgpt"
	if hasClientAuth {
		authState, writeErr := writeCodexAccountAuth(result.AuthPath, clientAuth.TokenValue)
		if writeErr != nil {
			return codexConfigureResult{}, writeErr
		}
		if authState != "existing" && !result.ImportedAuth && !result.AuthAlreadyAdded {
			result.AuthUpdated = true
		}
	} else {
		loginMethod = "api"
		authState, writeErr := ensureCodexAPIKeyAuth(result.AuthPath)
		if writeErr != nil {
			return codexConfigureResult{}, writeErr
		}
		if authState != "existing" {
			result.AuthUpdated = true
		}
	}

	if err := writeCodexOpenAIResponsesConfig(configPath, baseURL, models, loginMethod); err != nil {
		return codexConfigureResult{}, err
	}
	profilePaths, err := writeCodexModelProfiles(codexDir, models)
	if err != nil {
		return codexConfigureResult{}, err
	}
	result.ProfilePaths = profilePaths

	parts := []string{fmt.Sprintf("Codex 已配置 %d 个模型：%s，默认模型 %s", len(models), strings.Join(models, "、"), model)}
	if !hasClientAuth {
		parts = append(parts, "未找到可用 auth.json 账号，已自动切换为 API 登录模式")
	} else if result.ImportedAuth {
		parts = append(parts, "已导入现有 auth.json，并使用 ChatGPT 账号登录")
	} else if result.AuthUpdated {
		parts = append(parts, "已从 auth 池写入一个 ChatGPT 登录账号")
	} else if result.AuthAlreadyAdded {
		parts = append(parts, "auth.json 账号已存在，并继续用于 ChatGPT 登录")
	} else {
		parts = append(parts, "已使用 auth 池账号接入 ChatGPT")
	}
	parts = append(parts, "实际请求仍由 OmniProxy 后端 auth 池调度")
	result.Message = strings.Join(parts, "；")
	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "codex configured for omniproxy"})
	return result, nil
}

func normalizeCodexConfigureModel(model string) string {
	model = strings.TrimSpace(model)
	switch strings.ToLower(model) {
	case "deepseek-v4-pro[1m]":
		return "deepseek-v4-pro"
	default:
		return model
	}
}

func normalizeCodexConfigureModels(req codexConfigureRequest) []string {
	rawModels := req.Models
	if len(rawModels) == 0 && strings.TrimSpace(req.Model) != "" {
		rawModels = []string{req.Model}
	}
	if len(rawModels) == 0 {
		rawModels = defaultCodexModels
	}
	models := make([]string, 0, maxCodexModels)
	seen := map[string]bool{}
	for _, raw := range rawModels {
		model := normalizeCodexConfigureModel(raw)
		key := strings.ToLower(model)
		if model == "" || seen[key] {
			continue
		}
		models = append(models, model)
		seen[key] = true
		if len(models) >= maxCodexModels {
			break
		}
	}
	if len(models) == 0 {
		return append([]string(nil), defaultCodexModels...)
	}
	return models
}

func writeCodexModelProfiles(codexDir string, models []string) ([]string, error) {
	if err := cleanupCodexModelProfiles(codexDir); err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(models))
	for _, model := range models {
		path := filepath.Join(codexDir, codexProfileName(model)+".config.toml")
		lines := []string{
			fmt.Sprintf(`model = "%s"`, tomlEscape(model)),
			fmt.Sprintf(`review_model = "%s"`, tomlEscape(model)),
		}
		if contextWindow, compactLimit, ok := codexModelContext(model); ok {
			lines = append(lines,
				fmt.Sprintf("model_context_window = %d", contextWindow),
				fmt.Sprintf("model_auto_compact_token_limit = %d", compactLimit),
			)
		}
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
			return nil, err
		}
		paths = append(paths, path)
	}
	return paths, nil
}

func cleanupCodexModelProfiles(codexDir string) error {
	matches, err := filepath.Glob(filepath.Join(codexDir, "omniproxy-*.config.toml"))
	if err != nil {
		return err
	}
	for _, path := range matches {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func codexProfileName(model string) string {
	model = strings.ToLower(strings.TrimSpace(model))
	var builder strings.Builder
	builder.WriteString("omniproxy-")
	lastDash := false
	for _, r := range model {
		allowed := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-'
		if allowed {
			builder.WriteRune(r)
			lastDash = r == '-'
			continue
		}
		if !lastDash {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	name := strings.Trim(builder.String(), "-")
	if name == "omniproxy" {
		return "omniproxy-model"
	}
	return name
}

func codexModelContext(model string) (int, int, bool) {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "gpt-5.6", "gpt-5.6-sol", "gpt-5.6-terra":
		return 1_050_000, 900_000, true
	case "gpt-5.6-luna", "gpt-5.4-mini":
		return 400_000, 360_000, true
	case "gpt-5.5", "gpt-5.4":
		return 1_000_000, 900_000, true
	default:
		return 0, 0, false
	}
}

func (a *appServer) restoreCodexConfig() (codexConfigureResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return codexConfigureResult{}, err
	}

	codexDir := filepath.Join(home, ".codex")
	configPath := filepath.Join(codexDir, "config.toml")
	backupPath := configPath + ".omniproxy.bak"
	backup, err := os.ReadFile(backupPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return codexConfigureResult{}, errors.New("未找到 Codex 原始配置备份")
		}
		return codexConfigureResult{}, err
	}
	if err := os.WriteFile(configPath, backup, 0o600); err != nil {
		return codexConfigureResult{}, err
	}
	if err := cleanupCodexModelProfiles(codexDir); err != nil {
		return codexConfigureResult{}, err
	}

	a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "codex config restored"})
	return codexConfigureResult{
		ConfigPath: configPath,
		AuthPath:   filepath.Join(codexDir, "auth.json"),
		BackupPath: backupPath,
		Message:    "Codex 配置已恢复到一键配置前的原始配置",
	}, nil
}

func writeCodexOmniProxyConfig(path string, baseURL string) error {
	content, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	lines := []string{}
	if text != "" {
		lines = strings.Split(strings.TrimRight(text, "\n"), "\n")
	}

	lines = setRootStringKey(lines, "model_provider", "openai")
	lines = setRootStringKey(lines, "openai_base_url", baseURL)
	lines = removeRootKey(lines, "preferred_auth_method")
	lines = removeRootKey(lines, "chatgpt_base_url")
	lines = removeRootKey(lines, "disable_response_storage")
	lines = removeTomlSection(lines, "[model_providers.openai]")
	lines = removeTomlSection(lines, "[model_providers.omniproxy]")
	lines = removeTomlSection(lines, "[model_providers.anyrouter]")
	lines = removeTomlSection(lines, "[model_providers.prem]")

	next := strings.Join(lines, "\n") + "\n"
	if len(content) > 0 {
		backupPath := path + ".omniproxy.bak"
		if _, err := os.Stat(backupPath); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(backupPath, content, 0o600); err != nil {
				return err
			}
		}
	}
	return os.WriteFile(path, []byte(next), 0o600)
}

func writeCodexOpenAIResponsesConfig(path string, baseURL string, models []string, loginMethod string) error {
	content, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	models = normalizeCodexConfigureModels(codexConfigureRequest{Models: models})
	model := models[0]

	text := strings.ReplaceAll(string(content), "\r\n", "\n")
	lines := []string{}
	if text != "" {
		lines = strings.Split(strings.TrimRight(text, "\n"), "\n")
	}

	lines = setRootStringKey(lines, "model_provider", "openai")
	lines = setRootStringKey(lines, "model", model)
	lines = setRootStringKey(lines, "review_model", model)
	lines = setRootStringKey(lines, "model_reasoning_effort", "xhigh")
	lines = setRootStringKey(lines, "network_access", "enabled")
	lines = setRootStringKey(lines, "forced_login_method", loginMethod)
	lines = setRootStringKey(lines, "cli_auth_credentials_store", "file")
	lines = setRootRawKey(lines, "windows_wsl_setup_acknowledged", "true")
	contextWindow, compactLimit, ok := codexModelContext(model)
	if !ok {
		contextWindow, compactLimit = 1_000_000, 900_000
	}
	lines = setRootRawKey(lines, "model_context_window", fmt.Sprintf("%d", contextWindow))
	lines = setRootRawKey(lines, "model_auto_compact_token_limit", fmt.Sprintf("%d", compactLimit))
	lines = removeRootKey(lines, "preferred_auth_method")
	lines = setRootStringKey(lines, "openai_base_url", baseURL)
	lines = removeRootKey(lines, "chatgpt_base_url")
	lines = removeRootKey(lines, "disable_response_storage")
	lines = removeRootKey(lines, "profile")
	lines = removeTomlSection(lines, "[model_providers.openai]")
	lines = removeTomlSection(lines, "[model_providers.OpenAI]")
	lines = removeTomlSection(lines, "[model_providers.omniproxy]")
	lines = removeTomlSection(lines, "[model_providers.sub2api]")
	lines = removeTomlSection(lines, "[model_providers.newapi]")
	lines = removeTomlSection(lines, "[model_providers.zo]")
	lines = removeTomlSection(lines, "[model_providers.anyrouter]")
	lines = removeTomlSection(lines, "[model_providers.prem]")

	next := strings.Join(lines, "\n") + "\n"
	if len(content) > 0 {
		backupPath := path + ".omniproxy.bak"
		if _, err := os.Stat(backupPath); errors.Is(err, os.ErrNotExist) {
			if err := os.WriteFile(backupPath, content, 0o600); err != nil {
				return err
			}
		}
	}
	return os.WriteFile(path, []byte(next), 0o600)
}

func selectCodexClientAuth(items []token.Token, preferredID string) (token.Token, bool) {
	for _, status := range []token.Status{token.StatusActive, token.StatusLow} {
		for _, item := range items {
			if preferredID != "" && item.ID == preferredID && codexClientAuthUsable(item, status) {
				return item, true
			}
		}
		for _, selectedOnly := range []bool{true, false} {
			for _, item := range items {
				if item.Selected != selectedOnly || !codexClientAuthUsable(item, status) {
					continue
				}
				return item, true
			}
		}
	}
	return token.Token{}, false
}

func (a *appServer) selectFreshCodexClientAuth(ctx context.Context, preferredID string) (token.Token, bool) {
	items := a.tokens.List()
	for attempts := 0; attempts < len(items); attempts++ {
		selected, ok := selectCodexClientAuth(a.tokens.List(), preferredID)
		if !ok {
			return token.Token{}, false
		}

		refreshed, _, err := a.refreshAuthTokenIfNeeded(ctx, selected, false)
		if err == nil {
			return refreshed, true
		}
		_ = a.tokens.MarkInvalid(selected.ID, fmt.Sprintf("OAuth token refresh failed: %v", err))
		if a.logs != nil {
			a.logs.Add(logs.Entry{Level: logs.LevelWarn, TokenName: a.tokenDisplayName(selected), Message: fmt.Sprintf("codex client auth skipped: expired or cannot be refreshed: %v", err)})
		}
		if selected.ID == preferredID {
			preferredID = ""
		}
	}
	return token.Token{}, false
}

func (a *appServer) refreshCodexAuthValueIfNeeded(ctx context.Context, authValue string) (string, error) {
	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()

	selected := token.Token{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     authValue,
	}
	client, err := proxy.NewTokenHTTPClient(cfg, selected, healthRequestTimeout)
	if err != nil {
		return "", err
	}
	updated, _, err := proxy.RefreshCodexAuthJSON(ctx, client, authValue, false, time.Now())
	if err != nil {
		return "", err
	}
	return updated, nil
}

func codexClientAuthUsable(item token.Token, status token.Status) bool {
	if item.Disabled || item.Status != status || token.NormalizeProvider(item.Provider) != token.ProviderOpenAI || item.CredentialType != token.CredentialTypeCodexAuthJSON {
		return false
	}
	fields, ok := token.ExtractCodexAuthFields(item.TokenValue)
	return ok && strings.TrimSpace(fields.Email) != "" && strings.TrimSpace(fields.AccessToken) != ""
}

func writeCodexAccountAuth(path string, authValue string) (string, error) {
	var payload map[string]any
	if err := json.Unmarshal([]byte(authValue), &payload); err != nil || payload == nil {
		return "", errors.New("Codex App 登录账号 auth.json 无效")
	}
	delete(payload, "OPENAI_API_KEY")
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	data = append(data, '\n')

	state := "updated"
	content, readErr := os.ReadFile(path)
	if errors.Is(readErr, os.ErrNotExist) {
		state = "created"
	} else if readErr != nil {
		return "", readErr
	} else if strings.TrimSpace(string(content)) == strings.TrimSpace(string(data)) {
		return "existing", nil
	}
	return state, os.WriteFile(path, data, 0o600)
}

func ensureCodexAPIKeyAuth(path string) (string, error) {
	payload := map[string]any{}
	state := "updated"
	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		state = "created"
	} else if err != nil {
		return "", err
	} else if json.Unmarshal(content, &payload) == nil && payload != nil {
		if value, ok := payload["OPENAI_API_KEY"].(string); ok && strings.TrimSpace(value) != "" {
			return "existing", nil
		}
	} else {
		payload = map[string]any{}
	}

	payload["OPENAI_API_KEY"] = codexLocalAPIKey
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return state, os.WriteFile(path, append(data, '\n'), 0o600)
}

func setRootStringKey(lines []string, key string, value string) []string {
	replacement := fmt.Sprintf(`%s = "%s"`, key, tomlEscape(value))
	return setRootLine(lines, key, replacement)
}

func setRootRawKey(lines []string, key string, value string) []string {
	return setRootLine(lines, key, fmt.Sprintf("%s = %s", key, value))
}

func setRootLine(lines []string, key string, replacement string) []string {
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			break
		}
		if tomlKey(trimmed) == key {
			lines[i] = replacement
			return lines
		}
	}

	insertAt := len(lines)
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "[") {
			insertAt = i
			break
		}
	}

	next := make([]string, 0, len(lines)+2)
	next = append(next, lines[:insertAt]...)
	next = append(next, replacement)
	if insertAt < len(lines) && strings.TrimSpace(lines[insertAt]) != "" {
		next = append(next, "")
	}
	next = append(next, lines[insertAt:]...)
	return next
}

func removeTomlSection(lines []string, section string) []string {
	next := make([]string, 0, len(lines))
	skip := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.EqualFold(trimmed, section) {
			skip = true
			continue
		}
		if skip && strings.HasPrefix(trimmed, "[") {
			skip = false
		}
		if !skip {
			next = append(next, line)
		}
	}
	return next
}

func tomlKey(line string) string {
	if line == "" || strings.HasPrefix(line, "#") {
		return ""
	}
	beforeValue, _, ok := strings.Cut(line, "=")
	if !ok {
		return ""
	}
	return strings.TrimSpace(beforeValue)
}

func tomlEscape(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	return strings.ReplaceAll(value, `"`, `\"`)
}

func removeRootKey(lines []string, key string) []string {
	next := make([]string, 0, len(lines))
	inRoot := true
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			inRoot = false
		}
		if inRoot && tomlKey(trimmed) == key {
			continue
		}
		next = append(next, line)
	}
	return next
}
