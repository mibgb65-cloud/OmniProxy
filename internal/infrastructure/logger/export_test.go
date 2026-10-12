package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteExportPreservesExistingFileOnInvalidSelection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.jsonl")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := WriteExport(path, nil); err == nil {
		t.Fatal("empty export accepted")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "original" {
		t.Fatal("existing file changed")
	}
	if err := WriteExport(filepath.Join(dir, "unsafe.log"), []Entry{{Message: "safe"}}); err == nil {
		t.Fatal("unsafe extension accepted")
	}
	if err := WriteExport(path, []Entry{{ID: 1, Message: "safe"}}); err != nil {
		t.Fatal(err)
	}
	matches, err := filepath.Glob(filepath.Join(dir, ".omniproxy-log-export-*"))
	if err != nil || len(matches) != 0 {
		t.Fatal("temporary export files leaked")
	}
}

func TestSensitiveTokenKeysAndCounters(t *testing.T) {
	r := NewRedactor()
	for _, key := range []string{"token", "accessToken", "refreshToken", "authToken", "clientSecret", "accountName"} {
		if !r.SensitiveKey(key) {
			t.Fatalf("key %s was not sensitive", key)
		}
	}
	for _, key := range []string{"token_count", "input_tokens", "request_id"} {
		if r.SensitiveKey(key) {
			t.Fatalf("safe key %s masked", key)
		}
	}
}
