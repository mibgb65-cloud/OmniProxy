package agentconfig

import (
	"path/filepath"
	"testing"
)

func TestCodingClientPathPrecedence(t *testing.T) {
	root := t.TempDir()
	for _, key := range []string{"KILO_CONFIG", "KILO_CONFIG_DIR", "MIMOCODE_CONFIG", "MIMOCODE_CONFIG_DIR", "MIMOCODE_HOME", "QWEN_HOME"} {
		t.Setenv(key, "")
	}
	t.Setenv("XDG_CONFIG_HOME", root)
	writeGatewayFixture(t, filepath.Join(root, "kilo", "kilo.jsonc"), "{}")
	legacy := filepath.Join(root, "kilo", "opencode.jsonc")
	writeGatewayFixture(t, legacy, "{}")
	if kiloConfigPath(root) != legacy {
		t.Fatal("Kilo legacy file overrides the normal file and must be edited instead")
	}
	custom := filepath.Join(root, "custom")
	t.Setenv("KILO_CONFIG_DIR", custom)
	if kiloConfigPath(root) != filepath.Join(custom, "kilo.jsonc") {
		t.Fatal("Kilo custom directory ignored")
	}
	t.Setenv("KILO_CONFIG", filepath.Join(root, "explicit.json"))
	if kiloConfigPath(root) != filepath.Join(root, "explicit.json") {
		t.Fatal("Kilo explicit path ignored")
	}
	t.Setenv("MIMOCODE_HOME", custom)
	if mimoCodeConfigPath(root) != filepath.Join(custom, "config", "mimocode.jsonc") {
		t.Fatal("MiMo isolated home ignored")
	}
	t.Setenv("MIMOCODE_CONFIG_DIR", filepath.Join(root, "override"))
	if mimoCodeConfigPath(root) != filepath.Join(root, "override", "mimocode.jsonc") {
		t.Fatal("MiMo custom directory ignored")
	}
	t.Setenv("MIMOCODE_CONFIG", filepath.Join(root, "explicit-mimo.json"))
	if mimoCodeConfigPath(root) != filepath.Join(root, "explicit-mimo.json") {
		t.Fatal("MiMo explicit path ignored")
	}
	if qwenHome(root) != filepath.Join(root, ".qwen") {
		t.Fatal("Qwen default home incorrect")
	}
	t.Setenv("QWEN_HOME", custom)
	if qwenHome(root) != custom {
		t.Fatal("Qwen explicit home ignored")
	}
}
