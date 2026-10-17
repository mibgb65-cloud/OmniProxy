package agentinstall

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHarnessDesktopVersionedNSISRegistration(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "DeepSeek Harness.exe")
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"DeepSeek Harness", "DeepSeek Harness 0.2.0-rc.2"} {
		for _, icon := range []string{path + ",0", "\"" + path + "\",0"} {
			location, ok := codingDesktopLocation("harness-desktop", name, "", "0.2.0-rc.2", root, icon)
			if !ok || location != root {
				t.Fatal("NSIS registration not detected", name)
			}
		}
	}
	if location, ok := codingDesktopLocation("harness-desktop", "DeepSeek Harness", "DeepSeek AI", "0.2.0-rc.2", root, ""); !ok || location != root {
		t.Fatal("existing publisher registration no longer detected")
	}
}

func TestHarnessDesktopRejectsWrongProductAndUnsafeMetadata(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "DeepSeek Harness.exe")
	if err := os.WriteFile(path, []byte("fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ name, publisher, version, icon string }{
		{"DeepSeek Harness Plugin 0.2.0-rc.2", "", "0.2.0-rc.2", path + ",0"},
		{"DeepSeek Harness 0.2.0-rc.3", "", "0.2.0-rc.2", path + ",0"},
		{"DeepSeek Harness", "Unknown", "0.2.0-rc.2", path + ",0"},
		{"DeepSeek Harness", "", "bad version", path + ",0"},
		{"DeepSeek Harness", "", "0.2.0-rc.2", path + " --command bad"},
		{"DeepSeek Harness", "", "0.2.0-rc.2", path + ",1"},
		{"DeepSeek Harness", "", "0.2.0-rc.2", "DeepSeek Harness.exe,0"},
		{"DeepSeek Harness", "", "0.2.0-rc.2", "https://example.invalid/DeepSeek Harness.exe"},
		{"DeepSeek Harness", "", "0.2.0-rc.2", filepath.Join(root, "missing", "DeepSeek Harness.exe")},
		{"DeepSeek Harness", "", "0.2.0-rc.2", filepath.Join(root, "Uninstall DeepSeek Harness.exe")},
	} {
		if _, ok := codingDesktopLocation("harness-desktop", item.name, item.publisher, item.version, root, item.icon); ok {
			t.Fatal("unsafe or unrelated registration accepted")
		}
	}
}
