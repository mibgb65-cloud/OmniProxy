package logger

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"omniproxy/internal/platform/profile"
)

func readLogFile(t *testing.T, logDir string) []map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(logDir, "omniproxy.log"))
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	var records []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("log line is not JSON: %q: %v", line, err)
		}
		records = append(records, rec)
	}
	return records
}

func TestSetupWritesRedactedStructuredLogs(t *testing.T) {
	logDir := filepath.Join(t.TempDir(), "logs")
	log, closer, err := Setup(Options{Profile: profile.Development, LogDir: logDir, Console: io.Discard})
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	const secret = "Bearer sk-1234567890abcdef"
	log.Info("calling upstream provider", "authorization", secret, "provider", "openai", "model", "gpt-4")
	if err := closer.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	records := readLogFile(t, logDir)
	if len(records) != 1 {
		t.Fatalf("got %d log records, want 1", len(records))
	}
	rec := records[0]
	if rec["msg"] != "calling upstream provider" {
		t.Errorf("msg = %v, want %q", rec["msg"], "calling upstream provider")
	}
	if rec["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", rec["level"])
	}
	if rec["authorization"] != masked {
		t.Errorf("authorization = %v, want %q", rec["authorization"], masked)
	}
	if rec["provider"] != "openai" {
		t.Errorf("provider = %v, want openai", rec["provider"])
	}

	data, err := os.ReadFile(filepath.Join(logDir, "omniproxy.log"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), secret) || strings.Contains(string(data), "sk-1234567890abcdef") {
		t.Fatal("log file contains the raw secret")
	}
}

func TestSetupRedactsSecretsInMessage(t *testing.T) {
	logDir := filepath.Join(t.TempDir(), "logs")
	log, closer, err := Setup(Options{Profile: profile.Production, LogDir: logDir, Console: io.Discard})
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}
	log.Error("upstream rejected request: Authorization: Bearer sk-ant-api03-999888777")
	if err := closer.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(logDir, "omniproxy.log"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "sk-ant-api03-999888777") {
		t.Fatal("log file contains the raw api key from the message")
	}
}

func TestSetupHonoursLevel(t *testing.T) {
	logDir := filepath.Join(t.TempDir(), "logs")
	log, closer, err := Setup(Options{Profile: profile.Development, LogDir: logDir, Console: io.Discard, Level: "warn"})
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}
	log.Info("should be filtered")
	log.Warn("should be kept")
	if err := closer.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	records := readLogFile(t, logDir)
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0]["msg"] != "should be kept" {
		t.Errorf("msg = %v, want %q", records[0]["msg"], "should be kept")
	}
}

func TestSetupWithoutLogDirIsConsoleOnly(t *testing.T) {
	log, closer, err := Setup(Options{Profile: profile.Test, Console: io.Discard})
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}
	log.Info("console only")
	if err := closer.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestResolveLevel(t *testing.T) {
	tests := []struct {
		level string
		kind  profile.Kind
		want  string
	}{
		{"debug", profile.Production, "DEBUG"},
		{"warn", profile.Production, "WARN"},
		{"error", profile.Production, "ERROR"},
		{"", profile.Production, "INFO"},
		{"", profile.Development, "DEBUG"},
		{"", profile.Test, "INFO"},
		{"bogus", profile.Production, "INFO"},
	}
	for _, tt := range tests {
		got := resolveLevel(tt.level, tt.kind).String()
		if got != tt.want {
			t.Errorf("resolveLevel(%q, %q) = %s, want %s", tt.level, tt.kind, got, tt.want)
		}
	}
}
