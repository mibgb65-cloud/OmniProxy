package wails

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/infrastructure/logger"
	"omniproxy/internal/platform/profile"
	"omniproxy/internal/transport/httpapi"
)

func newTestService(t *testing.T) (*AppService, *profile.Profile) {
	t.Helper()
	t.Setenv("OMNIPROXY_CONFIG_DIR", t.TempDir())

	prof, err := profile.New(profile.Test)
	if err != nil {
		t.Fatalf("profile.New error: %v", err)
	}
	prof.DataDir = t.TempDir()
	prof.LogDir = filepath.Join(prof.DataDir, "logs")

	db, err := database.Open(database.Config{Path: prof.DatabasePath()})
	if err != nil {
		t.Fatalf("database.Open error: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	migrations := fstest.MapFS{
		"001_initial.sql": &fstest.MapFile{Data: []byte(`CREATE TABLE providers (id TEXT PRIMARY KEY);`)},
	}
	if err := database.Migrate(context.Background(), db, migrations); err != nil {
		t.Fatalf("database.Migrate error: %v", err)
	}

	log, files, err := logger.Setup(logger.Options{Profile: profile.Test, LogDir: prof.LogDir, Console: io.Discard})
	if err != nil {
		t.Fatalf("logger.Setup error: %v", err)
	}
	t.Cleanup(func() { _ = files.Close() })

	svc := NewAppService(prof, httpapi.New(prof), db, files, log)
	return svc, prof
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestStorageInfoReportsLocationsAndSizes(t *testing.T) {
	svc, prof := newTestService(t)
	svc.log.Info("a log line that occupies space")
	writeFile(t, filepath.Join(prof.DataDir, "cache", "response.bin"), "cached")
	writeFile(t, filepath.Join(prof.DataDir, "updates", "package-0123456789abcdef0123456789abcdef.exe"), "installer")
	writeFile(t, filepath.Join(prof.DataDir, "credentials", "test", "ref.bin"), "secret")
	writeFile(t, filepath.Join(prof.DataDir, "notes.txt"), "other")

	info := svc.StorageInfo()
	if info.DataDir != prof.DataDir {
		t.Errorf("DataDir = %q, want %q", info.DataDir, prof.DataDir)
	}
	if info.DatabasePath != prof.DatabasePath() {
		t.Errorf("DatabasePath = %q, want %q", info.DatabasePath, prof.DatabasePath())
	}
	if len(info.Items) != 6 {
		t.Fatalf("Items = %d, want 6", len(info.Items))
	}
	for i, name := range []string{"database", "logs", "cache", "updates", "credentials", "other"} {
		if info.Items[i].Name != name {
			t.Errorf("item %d name = %q, want %q", i, info.Items[i].Name, name)
		}
	}
	if info.Items[0].Bytes <= 0 {
		t.Errorf("database size = %d, want > 0", info.Items[0].Bytes)
	}
	sizes := map[string]int64{}
	var sum int64
	for _, item := range info.Items {
		sizes[item.Name] = item.Bytes
		sum += item.Bytes
	}
	if sizes["cache"] != 6 || sizes["updates"] != 9 || sizes["credentials"] != 6 || sizes["other"] != 5 {
		t.Errorf("cache/updates/credentials/other = %d/%d/%d/%d, want 6/9/6/5",
			sizes["cache"], sizes["updates"], sizes["credentials"], sizes["other"])
	}
	if info.TotalBytes != sum {
		t.Errorf("TotalBytes = %d, want sum of items %d", info.TotalBytes, sum)
	}
	if info.PendingDataDir != "" {
		t.Errorf("PendingDataDir = %q, want empty", info.PendingDataDir)
	}
}

func TestCleanStorageUpdatesRemovesInstallLeftovers(t *testing.T) {
	svc, prof := newTestService(t)
	helper := filepath.Join(prof.DataDir, "updates", "runner-0123456789abcdef0123456789abcdef.exe")
	writeFile(t, helper, "helper-copy")
	packageFile := filepath.Join(prof.DataDir, "updates", "package-0123456789abcdef0123456789abcdef.exe")
	writeFile(t, packageFile, "installer")

	result, err := svc.CleanStorage("updates")
	if err != nil {
		t.Fatalf("CleanStorage error: %v", err)
	}
	if result.Target != "updates" || result.FreedBytes != int64(len("helper-copy")) {
		t.Fatalf("result = %+v, want freed helper copy", result)
	}
	if _, err := os.Stat(helper); !os.IsNotExist(err) {
		t.Error("helper copy should be removed")
	}
	if _, err := os.Stat(packageFile); err != nil {
		t.Errorf("downloaded installer must be kept: %v", err)
	}
}

func TestStorageInfoReportsPendingDataDir(t *testing.T) {
	svc, prof := newTestService(t)
	target := filepath.Join(t.TempDir(), "pending")

	if _, err := svc.SetDataDir(target); err != nil {
		t.Fatalf("SetDataDir error: %v", err)
	}
	if info := svc.StorageInfo(); info.PendingDataDir != target {
		t.Fatalf("PendingDataDir = %q, want %q", info.PendingDataDir, target)
	}
	if prof.DataDir == target {
		t.Fatal("active data dir must not change until restart")
	}
}

func TestSetDataDirPersistsAndReportsRestart(t *testing.T) {
	svc, prof := newTestService(t)
	target := filepath.Join(t.TempDir(), "new-home")

	result, err := svc.SetDataDir(target)
	if err != nil {
		t.Fatalf("SetDataDir error: %v", err)
	}
	if result.PreviousDir != prof.DataDir || result.NewDir != target {
		t.Errorf("result = %+v", result)
	}
	if !result.RequiresRestart {
		t.Error("RequiresRestart = false, want true")
	}

	cfg, err := profile.LoadConfig(profile.Test)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataDir != target || cfg.PreviousDataDir != prof.DataDir {
		t.Errorf("persisted config = %+v", cfg)
	}
}

func TestSetDataDirRejectsInvalidInput(t *testing.T) {
	svc, prof := newTestService(t)

	if _, err := svc.SetDataDir(""); err == nil {
		t.Error("expected error for empty path")
	}
	if _, err := svc.SetDataDir(prof.DataDir); err == nil {
		t.Error("expected error for unchanged path")
	}
	if _, err := svc.SetDataDir(filepath.Join(prof.DataDir, "nested")); err == nil {
		t.Error("expected error for nested path")
	}
}

func TestCleanStorageLogs(t *testing.T) {
	svc, prof := newTestService(t)
	writeFile(t, filepath.Join(prof.LogDir, "omniproxy.log.1"), "backup-content")
	svc.log.Info("occupying the active log")

	result, err := svc.CleanStorage("logs")
	if err != nil {
		t.Fatalf("CleanStorage error: %v", err)
	}
	if result.Target != "logs" || result.FreedBytes <= 0 {
		t.Fatalf("result = %+v, want positive freed bytes", result)
	}
	if _, err := os.Stat(filepath.Join(prof.LogDir, "omniproxy.log.1")); !os.IsNotExist(err) {
		t.Error("backup should be removed")
	}
}

func TestCleanStorageDatabase(t *testing.T) {
	svc, prof := newTestService(t)

	// Grow the database with deletable rows so VACUUM has work to do.
	db, err := database.Open(database.Config{Path: prof.DatabasePath()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE tmp (id INTEGER PRIMARY KEY, blob TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO tmp (blob) VALUES (?)`, string(make([]byte, 200000))); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM tmp`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	result, err := svc.CleanStorage("database")
	if err != nil {
		t.Fatalf("CleanStorage error: %v", err)
	}
	if result.Target != "database" {
		t.Errorf("target = %q, want database", result.Target)
	}
	if result.FreedBytes < 0 {
		t.Errorf("freed bytes = %d, want >= 0", result.FreedBytes)
	}
}

func TestCleanStorageRejectsUnknownTarget(t *testing.T) {
	svc, _ := newTestService(t)
	if _, err := svc.CleanStorage("everything"); err == nil {
		t.Error("expected error for unknown target")
	}
}

func TestRelocateIfPendingMovesData(t *testing.T) {
	svc, prof := newTestService(t)
	target := filepath.Join(t.TempDir(), "relocated")
	writeFile(t, prof.DatabasePath(), "db-content")
	writeFile(t, filepath.Join(prof.LogDir, "omniproxy.log"), "log-content")

	if _, err := svc.SetDataDir(target); err != nil {
		t.Fatalf("SetDataDir error: %v", err)
	}
	svc.relocateIfPending()

	for _, rel := range []string{"omniproxy.db", filepath.Join("logs", "omniproxy.log")} {
		if _, err := os.Stat(filepath.Join(target, rel)); err != nil {
			t.Errorf("expected %s in the new location: %v", rel, err)
		}
	}
	if _, err := os.Stat(prof.DatabasePath()); !os.IsNotExist(err) {
		t.Error("expected the database to be gone from the old location")
	}
}

func TestRelocateIfPendingNoopWithoutChange(t *testing.T) {
	svc, prof := newTestService(t)
	writeFile(t, prof.DatabasePath(), "db-content")

	svc.relocateIfPending() // no persisted change: must not move anything

	if _, err := os.Stat(prof.DatabasePath()); err != nil {
		t.Errorf("database should stay in place: %v", err)
	}
}

func TestRelocateIfPendingRevertsOnFailure(t *testing.T) {
	svc, prof := newTestService(t)
	writeFile(t, prof.DatabasePath(), "db-content")

	// Persist a target that cannot be created: a regular file used as a directory.
	// (profile.SetDataDir would reject it, so write the config directly.)
	blocker := filepath.Join(t.TempDir(), "blocker")
	writeFile(t, blocker, "x")
	badTarget := filepath.Join(blocker, "sub")
	if err := profile.SaveConfig(profile.Test, profile.Config{DataDir: badTarget, PreviousDataDir: prof.DataDir}); err != nil {
		t.Fatalf("SaveConfig error: %v", err)
	}

	svc.relocateIfPending()

	after, err := profile.LoadConfig(profile.Test)
	if err != nil {
		t.Fatal(err)
	}
	if after.DataDir != prof.DataDir || after.PreviousDataDir != "" {
		t.Fatalf("config should be reverted to the active dir, got %+v", after)
	}
	if _, err := os.Stat(prof.DatabasePath()); err != nil {
		t.Errorf("database should stay in place after a failed move: %v", err)
	}
}
