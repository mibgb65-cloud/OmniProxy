package wails

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/infrastructure/logger"
	"omniproxy/internal/platform/profile"
)

func logTestService(t *testing.T) *AppService {
	t.Helper()
	dir := t.TempDir()
	log, files, err := logger.Setup(logger.Options{LogDir: dir, Console: io.Discard, UIBufferCapacity: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = files.Close() })
	return &AppService{log: log, logFiles: files, prof: &profile.Profile{LogDir: dir}}
}

func TestLogReadAndExport(t *testing.T) {
	service := logTestService(t)
	service.log.Info("first")
	service.log.Warn("second")
	service.log.Error("third")
	snapshot, err := service.LogsRead(logger.Query{Limit: 2})
	if err != nil || len(snapshot.Entries) != 2 || !snapshot.Dropped {
		t.Fatalf("snapshot=%+v,%v", snapshot, err)
	}
	path := filepath.Join(t.TempDir(), "view.jsonl")
	service.selectLogExport = func(language string) (string, error) { return path, nil }
	result, err := service.LogsExport([]uint64{1, 2}, "zh")
	if err != nil || result.Count != 1 || result.Missing != 1 {
		t.Fatalf("result=%+v,%v", result, err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "second") || strings.Contains(string(data), "third") {
		t.Fatalf("export=%s,%v", data, err)
	}
	if head, err := service.LogsHead(); err != nil || head != 4 {
		t.Fatal("export completion was not logged")
	}
}

func TestLogExportCancelAndErrors(t *testing.T) {
	service := logTestService(t)
	service.log.Info("first")
	service.selectLogExport = func(string) (string, error) { return "", nil }
	result, err := service.LogsExport([]uint64{1}, "en")
	if err != nil || !result.Cancelled {
		t.Fatal("cancel was not preserved")
	}
	service.selectLogExport = func(string) (string, error) { return "", errors.New("private path error") }
	if _, err = service.LogsExport([]uint64{1}, "en"); err == nil || strings.Contains(err.Error(), "private") {
		t.Fatal("unsafe error")
	}
	for _, language := range []string{"", "xx"} {
		if _, err = service.LogsExport([]uint64{1}, language); err == nil {
			t.Fatal("invalid locale accepted")
		}
	}
	if _, err = service.LogsExport([]uint64{1, 1}, "zh"); err == nil {
		t.Fatal("duplicate IDs accepted")
	}
	service.selectLogExport = func(string) (string, error) { return filepath.Join(t.TempDir(), "bad.log"), nil }
	if _, err = service.LogsExport([]uint64{1}, "zh"); err == nil {
		t.Fatal("unsafe extension accepted")
	}
}

func TestLogDirectoryAndUnavailable(t *testing.T) {
	service := logTestService(t)
	service.openLogDirectory = func(path string) error {
		if path != service.prof.LogDir {
			t.Fatal("unexpected path")
		}
		return nil
	}
	if err := service.LogsOpenDirectory(); err != nil {
		t.Fatal(err)
	}
	service.openLogDirectory = func(string) error { return errors.New("private") }
	if err := service.LogsOpenDirectory(); err == nil || strings.Contains(err.Error(), "private") {
		t.Fatal("unsafe error")
	}
	empty := &AppService{}
	if _, err := empty.LogsHead(); err == nil {
		t.Fatal("missing buffer accepted")
	}
	if _, err := empty.LogsRead(logger.Query{Limit: 1}); err == nil {
		t.Fatal("missing buffer accepted")
	}
}

func TestLogNotificationsCoalesceAndStop(t *testing.T) {
	service := logTestService(t)
	notices := make(chan LogNotice, 8)
	stop := forwardLogNotices(service.logFiles.Buffer(), func(notice LogNotice) { notices <- notice })
	defer stop()
	for i := 0; i < 100; i++ {
		service.log.Info("burst")
	}
	select {
	case notice := <-notices:
		if notice.LatestID != 100 {
			t.Fatalf("notice=%+v", notice)
		}
	case <-time.After(time.Second):
		t.Fatal("notification missing")
	}
	stop()
	stop()
	service.log.Info("after stop")
	select {
	case <-notices:
		t.Fatal("notification after stop")
	case <-time.After(2 * logNotifyInterval):
	}
}
