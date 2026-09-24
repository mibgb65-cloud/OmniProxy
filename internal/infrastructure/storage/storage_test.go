package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestDirSizeSumsNestedFiles(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "12345")             // 5
	writeFile(t, filepath.Join(dir, "sub", "b.txt"), "1234567890") // 10
	writeFile(t, filepath.Join(dir, "sub", "deep", "c.txt"), "12") // 2

	size, err := DirSize(dir)
	if err != nil {
		t.Fatalf("DirSize error: %v", err)
	}
	if size != 17 {
		t.Fatalf("DirSize() = %d, want 17", size)
	}
}

func TestDirSizeMissingDirIsZero(t *testing.T) {
	size, err := DirSize(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("DirSize error: %v", err)
	}
	if size != 0 {
		t.Fatalf("DirSize() = %d, want 0", size)
	}
}

func TestFileSize(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "db.sqlite")
	writeFile(t, path, "1234567")

	size, err := FileSize(path)
	if err != nil {
		t.Fatalf("FileSize error: %v", err)
	}
	if size != 7 {
		t.Fatalf("FileSize() = %d, want 7", size)
	}

	missing, err := FileSize(filepath.Join(dir, "nope"))
	if err != nil {
		t.Fatalf("FileSize(missing) error: %v", err)
	}
	if missing != 0 {
		t.Fatalf("FileSize(missing) = %d, want 0", missing)
	}
}

func TestMoveDirMovesAllEntries(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "relocated")
	writeFile(t, filepath.Join(src, "omniproxy.db"), "db-content")
	writeFile(t, filepath.Join(src, "logs", "omniproxy.log"), "log-content")

	if err := MoveDir(src, dst); err != nil {
		t.Fatalf("MoveDir error: %v", err)
	}

	for _, rel := range []string{"omniproxy.db", filepath.Join("logs", "omniproxy.log")} {
		if _, err := os.Stat(filepath.Join(dst, rel)); err != nil {
			t.Errorf("expected %s in destination: %v", rel, err)
		}
		if _, err := os.Stat(filepath.Join(src, rel)); !os.IsNotExist(err) {
			t.Errorf("expected %s removed from source", rel)
		}
	}
}

func TestMoveDirOverwritesExistingFiles(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, filepath.Join(src, "omniproxy.db"), "new")
	writeFile(t, filepath.Join(dst, "omniproxy.db"), "old")

	if err := MoveDir(src, dst); err != nil {
		t.Fatalf("MoveDir error: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dst, "omniproxy.db"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("destination content = %q, want %q", data, "new")
	}
}

func TestMoveDirMissingSourceIsNoop(t *testing.T) {
	dst := filepath.Join(t.TempDir(), "dst")
	if err := MoveDir(filepath.Join(t.TempDir(), "missing"), dst); err != nil {
		t.Fatalf("MoveDir error: %v", err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Errorf("destination should not be created for a missing source")
	}
}
