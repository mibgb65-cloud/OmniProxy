package logger

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type unavailableConsoleWriter struct{}

func (unavailableConsoleWriter) Write([]byte) (int, error) {
	return 0, errors.New("console unavailable")
}

func TestUnavailableConsoleDoesNotSuppressFileAndUIDiagnostics(t *testing.T) {
	dir := t.TempDir()
	log, files, err := Setup(Options{LogDir: dir, Console: unavailableConsoleWriter{}})
	if err != nil {
		t.Fatal(err)
	}
	defer files.Close()
	log.Warn("subscription authorization request failed", "module", "subscription", "code", "account_unavailable", "authorization", "private-auth-marker")
	snapshot, err := files.Buffer().Read(Query{Limit: 10})
	if err != nil || len(snapshot.Entries) != 1 || snapshot.Entries[0].Fields["code"] != "account_unavailable" {
		t.Fatal("UI lost diagnostics after console failed", snapshot, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "omniproxy.log"))
	if err != nil || !strings.Contains(string(data), "account_unavailable") || strings.Contains(string(data), "private-auth-marker") {
		t.Fatal("file diagnostics unavailable or unredacted")
	}
}
