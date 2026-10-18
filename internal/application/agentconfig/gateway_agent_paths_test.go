package agentconfig

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestGatewayAgentPathsRespectOfficialOverrides(t *testing.T) {
	root := t.TempDir()
	t.Setenv("KIMI_CODE_HOME", "")
	t.Setenv("KIMI_SHARE_DIR", "")
	t.Setenv("PI_CODING_AGENT_DIR", "")
	t.Setenv("GOOSE_PATH_ROOT", "")
	if agentHome(root, "KIMI_CODE_HOME", ".kimi-code") != filepath.Join(root, ".kimi-code") {
		t.Fatal("Kimi Code default incorrectly uses legacy Python client directory")
	}
	if agentHome(root, "PI_CODING_AGENT_DIR", ".pi", "agent") != filepath.Join(root, ".pi", "agent") {
		t.Fatal("Pi default directory incorrect")
	}
	t.Setenv("KIMI_CODE_HOME", filepath.Join(root, "custom-kimi"))
	t.Setenv("PI_CODING_AGENT_DIR", filepath.Join(root, "custom-pi"))
	if agentHome(root, "KIMI_CODE_HOME", ".kimi-code") != filepath.Join(root, "custom-kimi") || agentHome(root, "PI_CODING_AGENT_DIR", ".pi", "agent") != filepath.Join(root, "custom-pi") {
		t.Fatal("explicit client directory ignored")
	}
	if runtime.GOOS == "windows" {
		t.Setenv("APPDATA", root)
		if gooseConfigPath(root) != filepath.Join(root, "Block", "goose", "config", "config.yaml") {
			t.Fatal("Goose Windows configuration directory incorrect")
		}
	}
	t.Setenv("GOOSE_PATH_ROOT", root)
	if gooseConfigPath(root) != filepath.Join(root, "config", "config.yaml") {
		t.Fatal("Goose root override missing config subdirectory")
	}
	t.Setenv("GOOSE_PATH_ROOT", "relative-root")
	if gooseConfigPath(root) == filepath.Join("relative-root", "config", "config.yaml") {
		t.Fatal("relative Goose override accepted")
	}
}

func TestKimiCurrentAndLegacyConfigurationAreNotMixed(t *testing.T) {
	root := t.TempDir()
	t.Setenv("KIMI_CODE_HOME", "")
	t.Setenv("KIMI_SHARE_DIR", "")
	legacyPath := filepath.Join(root, ".kimi", "config.toml")
	writeGatewayFixture(t, legacyPath, "default_thinking=false\n")
	if path, legacy := kimiCodeConfigPath(root); path != legacyPath || !legacy {
		t.Fatal("existing Python Kimi config not detected")
	}
	currentPath := filepath.Join(root, ".kimi-code", "config.toml")
	writeGatewayFixture(t, currentPath, "telemetry=false\n")
	if path, legacy := kimiCodeConfigPath(root); path != currentPath || legacy {
		t.Fatal("current Kimi Code config did not take precedence")
	}
	t.Setenv("KIMI_SHARE_DIR", filepath.Join(root, "legacy-override"))
	if path, legacy := kimiCodeConfigPath(root); path != filepath.Join(root, "legacy-override", "config.toml") || !legacy {
		t.Fatal("legacy explicit directory ignored")
	}
	t.Setenv("KIMI_CODE_HOME", filepath.Join(root, "current-override"))
	if path, legacy := kimiCodeConfigPath(root); path != filepath.Join(root, "current-override", "config.toml") || legacy {
		t.Fatal("current explicit directory did not take precedence")
	}
}
