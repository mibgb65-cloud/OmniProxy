package legacy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSourceBoundsSnapshotAndReadOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "tokens.json")
	initial := `[{"id":"old","provider":"deepseek","tokenValue":"fake-only-test","unknown":{"ignored":true}}]`
	if err := os.WriteFile(path, []byte(initial), 0600); err != nil {
		t.Fatal(err)
	}
	first, err := Read(dir)
	if err != nil || len(first.Tokens) != 1 {
		t.Fatalf("read: %+v %v", first, err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"deepseekBaseUrl":"https://api.deepseek.com"}`), 0600); err != nil {
		t.Fatal(err)
	}
	second, err := Read(dir)
	if err != nil || first.Snapshot == second.Snapshot {
		t.Fatal("configuration change not detected")
	}
	data, _ := os.ReadFile(path)
	if string(data) != initial {
		t.Fatal("source was modified")
	}
	for _, bad := range []string{`null`, `{}`, `[1]`, `[{"id":4}]`, "[" + strings.Repeat(`{},`, 1000) + "{}]"} {
		if err := os.WriteFile(path, []byte(bad), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := Read(dir); err == nil {
			t.Errorf("accepted invalid source %.30s", bad)
		}
	}
	if _, err := Read("relative/path"); err == nil {
		t.Fatal("relative path accepted")
	}
}
