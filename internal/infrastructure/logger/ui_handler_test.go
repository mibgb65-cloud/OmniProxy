package logger

import (
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"omniproxy/internal/platform/profile"
)

func TestUILogStructuredAndNestedRedaction(t *testing.T) {
	dir := t.TempDir()
	log, files, err := Setup(Options{Profile: profile.Development, LogDir: dir, Console: io.Discard, UIBufferCapacity: 8})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = files.Close() }()
	secret := "sk-0123456789abcdefghijkl"
	log.With("module", "gateway", "name", "short-key-prefix").WithGroup("http").Info("request failed",
		slog.String("model", "deepseek-flash"), slog.Int("status_code", 401),
		slog.Any("error", errors.New("Bearer "+secret)),
		slog.Any("data", map[string]any{"nested": map[string]any{"apiKey": secret, "prompt": "private prompt", "safe": "ok"}}))
	log.WithGroup("response").Info("reply boundary", "text", "private reply")
	snapshot, err := files.Buffer().Read(Query{Limit: 8})
	if err != nil || len(snapshot.Entries) != 2 {
		t.Fatalf("snapshot=%+v,%v", snapshot, err)
	}
	entry := snapshot.Entries[0]
	if entry.Level != "INFO" || entry.Module != "gateway" || entry.Time == "" || entry.Fields["http.status_code"] != "401" || !strings.Contains(entry.Fields["http.error"], "***") {
		t.Fatalf("entry=%+v", entry)
	}
	if _, exists := entry.Fields["http.data"]; exists {
		t.Fatal("unapproved fields leaked into UI")
	}
	data, err := os.ReadFile(filepath.Join(dir, "omniproxy.log"))
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{secret, "private prompt", "private reply", "short-key-prefix"} {
		if strings.Contains(string(data), private) {
			t.Fatalf("private data leaked: %s", private)
		}
	}
}

func TestUILogFiltersLevelAndBoundsText(t *testing.T) {
	log, files, err := Setup(Options{LogDir: t.TempDir(), Level: "warn", Console: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = files.Close() }()
	log.Info("filtered")
	log.Warn(strings.Repeat("界", 2000), "error", strings.Repeat("x", 5000))
	snapshot, err := files.Buffer().Read(Query{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Entries) != 1 || len(snapshot.Entries[0].Message) > maxLogText+3 || len(snapshot.Entries[0].Fields["error"]) > maxLogText+3 || files.RecordedLevel() != "WARN" {
		t.Fatalf("snapshot=%+v", snapshot)
	}
}

func TestRecursiveGroupAndCyclicAnyAreSafe(t *testing.T) {
	log, files, err := Setup(Options{LogDir: t.TempDir(), Console: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = files.Close() }()
	cycle := map[string]any{}
	cycle["self"] = cycle
	attrs := make([]slog.Attr, 1)
	value := slog.GroupValue(attrs...)
	attrs[0] = slog.Attr{Key: "child", Value: value}
	log.Info("safe boundaries", slog.Any("error", cycle), slog.Attr{Key: "group", Value: value})
	if files.Buffer().Len() != 1 {
		t.Fatal("log write did not finish")
	}
}
