package main

import (
	"errors"
	"omniproxy/internal/claudedesktop"
	"omniproxy/internal/clientconfig"
	"omniproxy/internal/config"
	"omniproxy/internal/logs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestWriteSelectedClaudeSettingsUsesSelectedModels(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	initial := `{"env":{"ANTHROPIC_MODEL":"mimo-v2.5-pro[1m]","ANTHROPIC_DEFAULT_HAIKU_MODEL":"mimo-v2.5","ANTHROPIC_CUSTOM_MODEL_OPTION":"mimo-v2.5-pro","OTHER":"keep"},"availableModels":["custom-existing-model","mimo-v2.5-pro[1m]"],"modelOverrides":{"claude-opus-4-7":"mimo-v2.5-pro"}}` + "\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	targets, err := normalizeClaudeModelTargets([]string{"deepseek-v4-pro", "mimo-2.5-pro"})
	if err != nil {
		t.Fatal(err)
	}

	if err := writeSelectedClaudeSettings(path, "http://127.0.0.1:3000/anthropic-router", targets); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, expected := range []string{
		`"ANTHROPIC_BASE_URL": "http://127.0.0.1:3000/anthropic-router"`,
		`"ANTHROPIC_AUTH_TOKEN": "omniproxy"`,
		`"ANTHROPIC_MODEL": "deepseek-v4-pro"`,
		`"ANTHROPIC_DEFAULT_OPUS_MODEL": "deepseek-v4-pro"`,
		`"ANTHROPIC_DEFAULT_OPUS_MODEL_NAME": "DeepSeek V4 Pro"`,
		`"ANTHROPIC_DEFAULT_SONNET_MODEL": "mimo-v2.5-pro"`,
		`"ANTHROPIC_DEFAULT_SONNET_MODEL_NAME": "MiMo-V2.5-Pro"`,
		`"ANTHROPIC_DEFAULT_HAIKU_MODEL": "mimo-v2.5-pro"`,
		`"CLAUDE_CODE_SUBAGENT_MODEL": "mimo-v2.5-pro"`,
		`"CLAUDE_CODE_EFFORT_LEVEL": "max"`,
		`"availableModels": [`,
		`"OTHER": "keep"`,
		`"custom-existing-model"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected settings to contain %q, got:\n%s", expected, text)
		}
	}
	for _, unwanted := range []string{
		`"ANTHROPIC_CUSTOM_MODEL_OPTION"`,
		`"claude-opus-4-7"`,
		`"mimo-v2.5-pro[1m]"`,
	} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("expected settings not to contain %q, got:\n%s", unwanted, text)
		}
	}
}

func TestConfigureClaudeDesktopModelsUsesSelectedModels(t *testing.T) {
	localAppData := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", localAppData)

	server := &appServer{
		cfg:  config.Config{ProxyPort: 3000},
		logs: logs.NewRecorder(10),
	}
	result, err := server.configureClaudeDesktopModels(claudeModelsConfigureRequest{
		Models: []string{"deepseek-v4-pro", "mimo-v2.5"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Model != deepSeekProModel {
		t.Fatalf("expected primary model %q, got %q", deepSeekProModel, result.Model)
	}
	if !strings.Contains(result.Message, "Claude Code Desktop 已配置 2 个模型") {
		t.Fatalf("expected desktop-specific result message, got %q", result.Message)
	}

	paths, err := claudedesktop.CurrentPaths()
	if err != nil {
		t.Fatal(err)
	}
	for _, configPath := range []string{paths.NormalConfigPath, paths.ThreePConfigPath} {
		data, err := clientconfig.ReadJSONObject(configPath)
		if err != nil {
			t.Fatal(err)
		}
		if data["deploymentMode"] != "3p" {
			t.Fatalf("expected %s deploymentMode 3p, got %#v", configPath, data)
		}
	}

	profile, err := clientconfig.ReadJSONObject(paths.ProfilePath)
	if err != nil {
		t.Fatal(err)
	}
	if profile["inferenceProvider"] != "gateway" ||
		profile["inferenceGatewayBaseUrl"] != "http://127.0.0.1:3000/claude-desktop" ||
		profile["inferenceGatewayApiKey"] != claudedesktop.GatewayToken {
		t.Fatalf("unexpected desktop profile: %#v", profile)
	}
	models, ok := profile["inferenceModels"].([]any)
	if !ok || len(models) != 2 {
		t.Fatalf("expected 2 desktop models, got %#v", profile["inferenceModels"])
	}
	first := models[0].(map[string]any)
	if first["name"] != "claude-sonnet-4-6" || first["labelOverride"] != "DeepSeek V4 Pro" || first["supports1m"] == true {
		t.Fatalf("unexpected first desktop model: %#v", first)
	}
	second := models[1].(map[string]any)
	if second["name"] != "claude-opus-4-7" || second["labelOverride"] != "MiMo-V2.5" {
		t.Fatalf("unexpected second desktop model: %#v", second)
	}

	routes, err := claudedesktop.LoadRoutes()
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 2 || routes[0].RouteID != "claude-sonnet-4-6" || routes[0].UpstreamModel != deepSeekProModel || routes[1].UpstreamModel != "mimo-v2.5" {
		t.Fatalf("unexpected desktop routes: %#v", routes)
	}
	meta, err := clientconfig.ReadJSONObject(paths.MetaPath)
	if err != nil {
		t.Fatal(err)
	}
	if meta["appliedId"] != claudedesktop.ProfileID {
		t.Fatalf("expected applied profile id, got %#v", meta)
	}

	if _, err := server.restoreClaudeDesktopConfig(); err != nil {
		t.Fatal(err)
	}
	for _, configPath := range []string{paths.NormalConfigPath, paths.ThreePConfigPath} {
		data, err := clientconfig.ReadJSONObject(configPath)
		if err != nil {
			t.Fatal(err)
		}
		if data["deploymentMode"] != "1p" {
			t.Fatalf("expected %s deploymentMode 1p after restore, got %#v", configPath, data)
		}
	}
	if _, err := os.Stat(paths.ProfilePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected profile to be removed, got %v", err)
	}
	if _, err := os.Stat(paths.RoutesPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected routes to be removed, got %v", err)
	}
	meta, err = clientconfig.ReadJSONObject(paths.MetaPath)
	if err != nil {
		t.Fatal(err)
	}
	if meta["appliedId"] == claudedesktop.ProfileID {
		t.Fatalf("expected applied profile id to be cleared, got %#v", meta)
	}
}

func TestWriteSelectedClaudeSettingsPreservesDeepSeek1MModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"env":{"OTHER":"keep"}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	targets, err := normalizeClaudeModelTargets([]string{"deepseek-v4-pro[1m]"})
	if err != nil {
		t.Fatal(err)
	}

	if err := writeSelectedClaudeSettings(path, "http://127.0.0.1:3000/anthropic-router", targets); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, expected := range []string{
		`"ANTHROPIC_MODEL": "deepseek-v4-pro[1m]"`,
		`"ANTHROPIC_DEFAULT_OPUS_MODEL": "deepseek-v4-pro[1m]"`,
		`"ANTHROPIC_DEFAULT_OPUS_MODEL_NAME": "DeepSeek V4 Pro [1m]"`,
		`"CLAUDE_CODE_EFFORT_LEVEL": "max"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected settings to contain %q, got:\n%s", expected, text)
		}
	}
}

func TestConfigureClaudeDesktopModelsMarksOnlyDeepSeek1MAs1M(t *testing.T) {
	localAppData := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", localAppData)

	server := &appServer{
		cfg:  config.Config{ProxyPort: 3000},
		logs: logs.NewRecorder(10),
	}
	result, err := server.configureClaudeDesktopModels(claudeModelsConfigureRequest{
		Models: []string{"deepseek-v4-pro[1m]"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Model != deepSeekProLongModel {
		t.Fatalf("expected primary 1M model %q, got %q", deepSeekProLongModel, result.Model)
	}

	paths, err := claudedesktop.CurrentPaths()
	if err != nil {
		t.Fatal(err)
	}
	profile, err := clientconfig.ReadJSONObject(paths.ProfilePath)
	if err != nil {
		t.Fatal(err)
	}
	models, ok := profile["inferenceModels"].([]any)
	if !ok || len(models) != 1 {
		t.Fatalf("expected 1 desktop model, got %#v", profile["inferenceModels"])
	}
	first := models[0].(map[string]any)
	if first["labelOverride"] != "DeepSeek V4 Pro [1m]" || first["supports1m"] != true {
		t.Fatalf("unexpected 1M desktop model: %#v", first)
	}
	routes, err := claudedesktop.LoadRoutes()
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || routes[0].UpstreamModel != deepSeekProLongModel {
		t.Fatalf("unexpected 1M desktop routes: %#v", routes)
	}
}

func TestWriteSelectedClaudeSettingsSupportsOfficialClaudeAliases(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"env":{"OTHER":"keep"}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	targets, err := normalizeClaudeModelTargets([]string{"default", "sonnet", "opus", "haiku"})
	if err != nil {
		t.Fatal(err)
	}

	if err := writeSelectedClaudeSettings(path, "http://127.0.0.1:3000/anthropic-router", targets); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, expected := range []string{
		`"ANTHROPIC_MODEL": "default"`,
		`"ANTHROPIC_DEFAULT_OPUS_MODEL": "opus"`,
		`"ANTHROPIC_DEFAULT_SONNET_MODEL": "sonnet"`,
		`"ANTHROPIC_DEFAULT_HAIKU_MODEL": "haiku"`,
		`"CLAUDE_CODE_SUBAGENT_MODEL": "sonnet"`,
		`"OTHER": "keep"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected settings to contain %q, got:\n%s", expected, text)
		}
	}
	if strings.Contains(text, `"ANTHROPIC_CUSTOM_MODEL_OPTION"`) {
		t.Fatalf("expected official aliases not to create a custom model option, got:\n%s", text)
	}
}

func TestConfigureClaudeDesktopModelsSupportsZoModels(t *testing.T) {
	localAppData := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("LOCALAPPDATA", localAppData)

	server := &appServer{
		cfg:  config.Config{ProxyPort: 3000},
		logs: logs.NewRecorder(10),
	}
	result, err := server.configureClaudeDesktopModels(claudeModelsConfigureRequest{
		Models: []string{"claude-opus-4-7", "claude-sonnet-4-6"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Model != zoClaudeModel {
		t.Fatalf("expected primary Zo model %q, got %q", zoClaudeModel, result.Model)
	}

	paths, err := claudedesktop.CurrentPaths()
	if err != nil {
		t.Fatal(err)
	}
	profile, err := clientconfig.ReadJSONObject(paths.ProfilePath)
	if err != nil {
		t.Fatal(err)
	}
	models, ok := profile["inferenceModels"].([]any)
	if !ok || len(models) != 2 {
		t.Fatalf("expected 2 desktop models, got %#v", profile["inferenceModels"])
	}
	first := models[0].(map[string]any)
	if first["name"] != "claude-sonnet-4-6" || first["labelOverride"] != "Zo Claude Opus 4.7" {
		t.Fatalf("unexpected first Zo desktop model: %#v", first)
	}
	second := models[1].(map[string]any)
	if second["name"] != "claude-opus-4-7" || second["labelOverride"] != "Zo Claude Sonnet 4.6" {
		t.Fatalf("unexpected second Zo desktop model: %#v", second)
	}

	routes, err := claudedesktop.LoadRoutes()
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 2 || routes[0].UpstreamModel != zoClaudeModel || routes[1].UpstreamModel != zoClaudeSonnetModel {
		t.Fatalf("unexpected Zo desktop routes: %#v", routes)
	}
}

func TestNormalizeClaudeModelTargetsValidatesSelection(t *testing.T) {
	targets, err := normalizeClaudeModelTargets([]string{
		"deepseek-4-pro",
		"mimo-2.5-pro",
		"kimi-for-coding",
		"claude-opus-4-7",
	})
	if err != nil {
		t.Fatal(err)
	}
	models := claudeModelIDs(targets)
	expected := []string{deepSeekProModel, mimoModel, kimiCodingModel, zoClaudeModel}
	if !reflect.DeepEqual(models, expected) {
		t.Fatalf("expected normalized models %#v, got %#v", expected, models)
	}

	deepseekTargets, err := normalizeClaudeModelTargets([]string{"deepseek-v4-pro", "deepseek-v4-pro[1m]"})
	if err != nil {
		t.Fatal(err)
	}
	deepseekModels := claudeModelIDs(deepseekTargets)
	deepseekExpected := []string{deepSeekProModel, deepSeekProLongModel}
	if !reflect.DeepEqual(deepseekModels, deepseekExpected) {
		t.Fatalf("expected DeepSeek models %#v, got %#v", deepseekExpected, deepseekModels)
	}

	officialTargets, err := normalizeClaudeModelTargets([]string{"default", "sonnet", "opus", "haiku"})
	if err != nil {
		t.Fatal(err)
	}
	officialModels := claudeModelIDs(officialTargets)
	officialExpected := []string{claudeDefaultModel, claudeSonnetModel, claudeOpusModel, claudeHaikuModel}
	if !reflect.DeepEqual(officialModels, officialExpected) {
		t.Fatalf("expected official models %#v, got %#v", officialExpected, officialModels)
	}

	if _, err := normalizeClaudeModelTargets(nil); err == nil {
		t.Fatal("expected empty model selection to fail")
	}
	if _, err := normalizeClaudeModelTargets([]string{"unknown-model"}); err == nil {
		t.Fatal("expected unknown model to fail")
	}
	if _, err := normalizeClaudeModelTargets([]string{
		deepSeekProModel,
		deepSeekFastModel,
		mimoLongContextModel,
		mimoModel,
		kimiCodingModel,
	}); err == nil {
		t.Fatal("expected more than four selected models to fail")
	}
}

func TestWriteClaudeRouterSettingsAcceptsUTF8BOM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	raw := append([]byte{0xEF, 0xBB, 0xBF}, []byte(`{"env":{"OTHER":"keep"}}`+"\n")...)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	targets, err := normalizeClaudeModelTargets([]string{"mimo-2.5"})
	if err != nil {
		t.Fatal(err)
	}
	if err := writeSelectedClaudeSettings(path, "http://127.0.0.1:3000/anthropic-router", targets); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(string(content), "\ufeff") {
		t.Fatalf("expected rewritten settings to omit UTF-8 BOM, got:\n%s", string(content))
	}
	if !strings.Contains(string(content), `"ANTHROPIC_BASE_URL": "http://127.0.0.1:3000/anthropic-router"`) {
		t.Fatalf("expected router base url, got:\n%s", string(content))
	}
}

func TestWriteSelectedClaudeSettingsSupportsAtriaModel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"env":{"OTHER":"keep"}}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	targets, err := normalizeClaudeModelTargets([]string{"Atria-Dawn-Preview"})
	if err != nil {
		t.Fatal(err)
	}
	if models := claudeModelIDs(targets); len(models) != 1 || models[0] != atriaClaudeModel {
		t.Fatalf("expected normalized Atria model %q, got %#v", atriaClaudeModel, models)
	}

	if err := writeSelectedClaudeSettings(path, "http://127.0.0.1:3000/anthropic-router", targets); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, expected := range []string{
		`"ANTHROPIC_MODEL": "Atria-Dawn-Preview"`,
		`"ANTHROPIC_DEFAULT_OPUS_MODEL": "Atria-Dawn-Preview"`,
		`"ANTHROPIC_DEFAULT_OPUS_MODEL_NAME": "Atria Dawn Preview"`,
		`"CLAUDE_CODE_SUBAGENT_MODEL": "Atria-Dawn-Preview"`,
		`"OTHER": "keep"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("expected settings to contain %q, got:\n%s", expected, text)
		}
	}
	if strings.Contains(text, `"CLAUDE_CODE_EFFORT_LEVEL"`) {
		t.Fatalf("expected Atria selection not to force max effort, got:\n%s", text)
	}
}
