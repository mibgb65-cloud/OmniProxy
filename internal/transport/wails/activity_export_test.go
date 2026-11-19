package wails

import (
	"archive/zip"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/application/activity"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/platform/profile"
)

func requestExportService(t *testing.T) *AppService {
	t.Helper()
	prof, err := profile.New(profile.Test)
	if err != nil {
		t.Fatal(err)
	}
	prof.DataDir = t.TempDir()
	db, err := database.Open(database.Config{Path: prof.DatabasePath()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	service := NewAppService(prof, nil, db, nil, nil)
	service.activity = activity.New(db)
	return service
}

func seedExportRequest(t *testing.T, service *AppService) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	event := proxy.RequestEvent{
		ID: strings.Repeat("1", 32), StartedAt: now, FinishedAt: now, Protocol: "responses", Transport: "http",
		Model: "gpt-export", InputModel: "alias", State: "success", StatusCode: 200,
		Target: proxy.TargetSnapshot{ProviderID: "builtin_river", ProviderName: "River", AccountID: "acct", AccountName: "Member", AuthMethod: "api_key"},
	}
	if err := service.activity.Begin(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := service.activity.Update(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := service.activity.Finish(ctx, event); err != nil {
		t.Fatal(err)
	}
	if err := service.activity.RecordRequestContent(ctx, proxy.RequestContentEvent{
		RequestID: event.ID, Content: `{"messages":[{"role":"user","content":"hello"}]}`, OriginalBytes: 45,
	}); err != nil {
		t.Fatal(err)
	}
}

func sheetText(t *testing.T, path string) string {
	t.Helper()
	reader, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		if file.Name != "xl/worksheets/sheet1.xml" {
			continue
		}
		handle, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(handle)
		handle.Close()
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	t.Fatal("worksheet part missing")
	return ""
}

func TestRequestsExportWritesXlsx(t *testing.T) {
	service := requestExportService(t)
	seedExportRequest(t, service)
	path := filepath.Join(t.TempDir(), "requests.xlsx")
	service.selectRequestsExport = func(string) (string, error) { return path, nil }
	result, err := service.RequestsExport(activity.RequestQuery{}, "zh")
	if err != nil || result.Count != 1 || result.Limited || result.Cancelled {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if info, err := os.Stat(path); err != nil || info.Size() == 0 {
		t.Fatal("export file missing")
	}
	sheet := sheetText(t, path)
	for _, want := range []string{"请求时间", "gpt-export", "hello", "请求正文"} {
		if !strings.Contains(sheet, want) {
			t.Fatalf("export sheet missing %q", want)
		}
	}
}

func TestRequestsExportCancelAndErrors(t *testing.T) {
	service := requestExportService(t)
	seedExportRequest(t, service)
	service.selectRequestsExport = func(string) (string, error) { return "", nil }
	result, err := service.RequestsExport(activity.RequestQuery{}, "en")
	if err != nil || !result.Cancelled {
		t.Fatal("cancel was not preserved")
	}
	for _, language := range []string{"", "xx"} {
		if _, err = service.RequestsExport(activity.RequestQuery{}, language); err == nil {
			t.Fatal("invalid language accepted")
		}
	}
	service.selectRequestsExport = func(string) (string, error) { return "", errors.New("private dialog error") }
	if _, err = service.RequestsExport(activity.RequestQuery{}, "zh"); err == nil || strings.Contains(err.Error(), "private") {
		t.Fatal("dialog error leaked")
	}
	service.selectRequestsExport = func(string) (string, error) {
		return filepath.Join(t.TempDir(), "bad.txt"), nil
	}
	if _, err = service.RequestsExport(activity.RequestQuery{}, "zh"); err == nil {
		t.Fatal("unsafe extension accepted")
	}
	service.requestsExportMu.Lock()
	defer service.requestsExportMu.Unlock()
	if _, err = service.RequestsExport(activity.RequestQuery{}, "zh"); err == nil {
		t.Fatal("concurrent export accepted")
	}
}

func TestRequestContentBindings(t *testing.T) {
	service := requestExportService(t)
	ctx := context.Background()
	seedExportRequest(t, service)
	content, err := service.RequestContentRead(strings.Repeat("1", 32))
	if err != nil || !content.Available || !strings.Contains(content.Content, "hello") {
		t.Fatalf("content=%+v err=%v", content, err)
	}
	retention, err := service.RequestContentSet(true)
	if err != nil || !retention.ContentEnabled {
		t.Fatalf("retention=%+v err=%v", retention, err)
	}
	removed, err := service.RequestContentClear()
	if err != nil || removed != 1 {
		t.Fatalf("removed=%d err=%v", removed, err)
	}
	if content, err = service.RequestContentRead(strings.Repeat("1", 32)); err != nil || content.Available {
		t.Fatal("cleared content still available", content, err)
	}
	if enabled, err := service.activity.ContentCaptureEnabled(ctx); err != nil || !enabled {
		t.Fatal("content setting not persisted")
	}
	empty := &AppService{}
	if _, err := empty.RequestContentRead(strings.Repeat("1", 32)); !errors.Is(err, activity.ErrUnavailable) {
		t.Fatal("missing service hidden")
	}
}
