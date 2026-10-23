package updater

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCleanFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCleanCacheRemovesInstallLeftoversOnly(t *testing.T) {
	dir := t.TempDir()
	helper := writeCleanFixture(t, dir, "runner-0123456789abcdef0123456789abcdef.exe", "1234567890")
	helperSecond := writeCleanFixture(t, dir, "runner-fedcba9876543210fedcba9876543210.exe", "12345")
	writeCleanFixture(t, dir, "status.json.tmp", "1234")
	kept := []string{
		writeCleanFixture(t, dir, "package-0123456789abcdef0123456789abcdef.exe", "123456789"),
		writeCleanFixture(t, dir, "status.json", "{\"file\":\"package-0123456789abcdef0123456789abcdef.exe\"}"),
		writeCleanFixture(t, dir, "install-result.json", "{\"success\":false}"),
		writeCleanFixture(t, dir, "runner-not-a-nonce.exe", "123"),
	}

	freed, err := CleanCache(dir)
	if err != nil {
		t.Fatal(err)
	}
	if freed != 19 {
		t.Fatalf("freed = %d, want 19", freed)
	}
	for _, path := range []string{helper, helperSecond, filepath.Join(dir, "status.json.tmp")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s should have been removed", filepath.Base(path))
		}
	}
	for _, path := range kept {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("%s must be preserved: %v", filepath.Base(path), err)
		}
	}
}

func TestCleanCacheKeepsDirectoriesAndMissingRoot(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "runner-0123456789abcdef0123456789abcdef.exe")
	if err := os.Mkdir(nested, 0o750); err != nil {
		t.Fatal(err)
	}
	freed, err := CleanCache(dir)
	if err != nil || freed != 0 {
		t.Fatalf("freed=%d err=%v, want 0/nil", freed, err)
	}
	if _, err := os.Stat(nested); err != nil {
		t.Fatalf("directory must be preserved: %v", err)
	}
	if freed, err := CleanCache(filepath.Join(dir, "missing")); err != nil || freed != 0 {
		t.Fatalf("missing directory: freed=%d err=%v, want 0/nil", freed, err)
	}
}
