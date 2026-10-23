package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMeasureUsageGroupsAllDataFiles(t *testing.T) {
	dataDir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		path := filepath.Join(dataDir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("omniproxy.db", "1234")
	write("omniproxy.db-wal", "123")
	write("omniproxy.db-shm", "12")
	write("omniproxy.db-journal", "1")
	write(filepath.Join("logs", "old", "app.log"), "12345")
	write(filepath.Join("cache", "nested", "entry"), "123456")
	write(filepath.Join("updates", "package-0123456789abcdef0123456789abcdef.exe"), "123456789")
	write(filepath.Join("credentials", "production", "ref.bin"), "1234567890")
	write("app.json", "1234567")
	write(filepath.Join("logs-old", "archive"), "12345678")

	usage, err := MeasureUsage(dataDir, filepath.Join(dataDir, "omniproxy.db"), filepath.Join(dataDir, "logs"))
	if err != nil {
		t.Fatal(err)
	}
	if usage.Database != 10 || usage.Logs != 5 || usage.Cache != 6 ||
		usage.Updates != 9 || usage.Credentials != 10 || usage.Other != 15 {
		t.Fatalf(
			"usage = %+v, want database=10 logs=5 cache=6 updates=9 credentials=10 other=15",
			usage,
		)
	}
	if usage.Total() != 55 {
		t.Fatalf("total = %d, want 55", usage.Total())
	}
}

func TestMeasureUsageMissingAndInvalidRoot(t *testing.T) {
	dataDir := t.TempDir()
	missing := filepath.Join(dataDir, "missing")
	usage, err := MeasureUsage(missing, filepath.Join(missing, "omniproxy.db"), filepath.Join(missing, "logs"))
	if err != nil || usage.Total() != 0 {
		t.Fatalf("missing directory: usage=%+v, err=%v", usage, err)
	}
	file := filepath.Join(dataDir, "file")
	if err := os.WriteFile(file, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := MeasureUsage(file, filepath.Join(file, "omniproxy.db"), filepath.Join(file, "logs")); err == nil {
		t.Fatal("regular file used as data directory should fail")
	}
}
