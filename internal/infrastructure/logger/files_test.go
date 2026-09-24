package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestFiles(t *testing.T) (*Files, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "omniproxy.log")
	rot, err := NewRotatingFile(path)
	if err != nil {
		t.Fatalf("NewRotatingFile error: %v", err)
	}
	t.Cleanup(func() { _ = rot.Close() })
	return &Files{dir: dir, path: path, active: rot}, dir
}

func TestFilesCleanRemovesBackupsAndTruncatesActive(t *testing.T) {
	files, dir := newTestFiles(t)

	// Active log content: 100 bytes plus the newline. Backups: 50 + 30.
	if _, err := files.active.Write([]byte(strings.Repeat("x", 100) + "\n")); err != nil {
		t.Fatal(err)
	}
	for i, size := range []int{50, 30} {
		backup := filepath.Join(dir, "omniproxy.log."+string(rune('1'+i)))
		if err := os.WriteFile(backup, []byte(strings.Repeat("b", size)), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// A non-numbered file must be left alone.
	if err := os.WriteFile(filepath.Join(dir, "omniproxy.log.old"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	freed, err := files.Clean()
	if err != nil {
		t.Fatalf("Clean error: %v", err)
	}
	if want := int64(101 + 50 + 30); freed != want {
		t.Errorf("freed = %d, want %d", freed, want)
	}

	info, err := os.Stat(filepath.Join(dir, "omniproxy.log"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() != 0 {
		t.Errorf("active log size = %d, want 0", info.Size())
	}
	for _, name := range []string{"omniproxy.log.1", "omniproxy.log.2"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("expected %s to be removed", name)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "omniproxy.log.old")); err != nil {
		t.Errorf("non-numbered file should be kept: %v", err)
	}
}

func TestFilesCleanConsoleOnlyIsNoop(t *testing.T) {
	var files Files
	freed, err := files.Clean()
	if err != nil || freed != 0 {
		t.Fatalf("Clean() = %d, %v; want 0, nil", freed, err)
	}
	if err := files.Close(); err != nil {
		t.Fatalf("Close() = %v, want nil", err)
	}
}

func TestSetupWithoutLogDirReturnsEmptyFiles(t *testing.T) {
	_, files, err := Setup(Options{Profile: "test", Console: os.Stderr})
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}
	if files == nil {
		t.Fatal("Files handle must not be nil even in console-only mode")
	}
	freed, err := files.Clean()
	if err != nil || freed != 0 {
		t.Fatalf("Clean() = %d, %v; want 0, nil", freed, err)
	}
}
