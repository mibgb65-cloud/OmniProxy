package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotatingFileRotatesAtMaxSize(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	w, err := newRotatingFile(path, 64, 2)
	if err != nil {
		t.Fatalf("newRotatingFile error: %v", err)
	}
	defer func() { _ = w.Close() }()

	line := strings.Repeat("x", 30) + "\n" // 31 bytes
	for i := 0; i < 6; i++ {
		if _, err := w.Write([]byte(line)); err != nil {
			t.Fatalf("write %d error: %v", i, err)
		}
	}

	for _, name := range []string{path, path + ".1", path + ".2"} {
		info, err := os.Stat(name)
		if err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
		if info.Size() > 64 {
			t.Errorf("%s size = %d, want <= 64", name, info.Size())
		}
	}
	if _, err := os.Stat(path + ".3"); !os.IsNotExist(err) {
		t.Errorf("expected %s.3 to be pruned, stat err = %v", path, err)
	}
}

func TestRotatingFileKeepsBackupsWithinLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	w, err := newRotatingFile(path, 32, 1)
	if err != nil {
		t.Fatalf("newRotatingFile error: %v", err)
	}
	defer func() { _ = w.Close() }()

	for i := 0; i < 5; i++ {
		if _, err := w.Write([]byte(strings.Repeat("y", 20) + "\n")); err != nil {
			t.Fatalf("write %d error: %v", i, err)
		}
	}
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Fatalf("expected backup .1: %v", err)
	}
	if _, err := os.Stat(path + ".2"); !os.IsNotExist(err) {
		t.Errorf("expected backup .2 to be pruned, stat err = %v", err)
	}
}

func TestRotatingFileRejectsInvalidMaxSize(t *testing.T) {
	if _, err := newRotatingFile(filepath.Join(t.TempDir(), "a.log"), 0, 1); err == nil {
		t.Fatal("expected error for non-positive maxSize")
	}
}

func TestRotatingFileCloseIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	w, err := NewRotatingFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}
