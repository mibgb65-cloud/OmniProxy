package database

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenCreatesDatabaseAndParentDirs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "test.db")
	db, err := Open(Config{Path: path})
	if err != nil {
		t.Fatalf("Open error: %v", err)
	}
	defer func() { _ = db.Close() }()

	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("ping after open: %v", err)
	}
}

func TestOpenRejectsEmptyPath(t *testing.T) {
	if _, err := Open(Config{}); err == nil {
		t.Fatal("expected error for empty path")
	}
}

func TestOpenIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	first, err := Open(Config{Path: path})
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := Open(Config{Path: path})
	if err != nil {
		t.Fatalf("reopen error: %v", err)
	}
	defer func() { _ = second.Close() }()
}
