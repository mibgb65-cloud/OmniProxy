package wails

import (
	"testing"

	"omniproxy/internal/application/modelcatalog"
)

// TestMergeSyncStatusCombinesProviderInstances 断言多个服务商的后台同步状态合并成一份：
// 任一运行即运行、待同步数量求和、错误码取第一个、时间取最近一次。
func TestMergeSyncStatusCombinesProviderInstances(t *testing.T) {
	merged := mergeSyncStatus(
		modelcatalog.SyncStatus{Stale: 2, UpdatedAt: "2026-11-16T00:00:01Z"},
		modelcatalog.SyncStatus{Running: true, Stale: 1, LastError: "howtofix_auto_sync_failed", UpdatedAt: "2026-11-16T00:00:03Z"},
		modelcatalog.SyncStatus{Stale: 1, LastError: "ignored_second_error", UpdatedAt: "2026-11-16T00:00:02Z"},
	)
	if !merged.Running || merged.Stale != 4 {
		t.Fatalf("merged = %+v", merged)
	}
	if merged.LastError != "howtofix_auto_sync_failed" {
		t.Fatalf("last error = %q, want the first non-empty one", merged.LastError)
	}
	if merged.UpdatedAt != "2026-11-16T00:00:03Z" {
		t.Fatalf("updated at = %q, want the latest", merged.UpdatedAt)
	}
}

func TestMergeSyncStatusKeepsEmptyStatusZero(t *testing.T) {
	if merged := mergeSyncStatus(); merged.Running || merged.Stale != 0 || merged.LastError != "" || merged.UpdatedAt != "" {
		t.Fatalf("empty merge = %+v", merged)
	}
	if merged := mergeSyncStatus(modelcatalog.SyncStatus{}, modelcatalog.SyncStatus{}); merged.Running || merged.Stale != 0 {
		t.Fatalf("empty statuses = %+v", merged)
	}
}
