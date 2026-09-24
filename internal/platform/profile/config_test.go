package profile

import (
	"os"
	"path/filepath"
	"testing"
)

// useTempConfigDir points the persisted configuration at a temp dir so tests
// never touch the real user config directory.
func useTempConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(envConfigDir, dir)
	return dir
}

func TestConfigFilePathUsesConfigDirOverride(t *testing.T) {
	dir := useTempConfigDir(t)
	if got, want := ConfigFilePath(Development), filepath.Join(dir, "app.json"); got != want {
		t.Fatalf("ConfigFilePath() = %q, want %q", got, want)
	}
}

func TestLoadConfigMissingFileIsEmpty(t *testing.T) {
	useTempConfigDir(t)
	cfg, err := LoadConfig(Test)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.DataDir != "" || cfg.PreviousDataDir != "" {
		t.Fatalf("expected empty config, got %+v", cfg)
	}
}

func TestSaveAndLoadConfigRoundTrip(t *testing.T) {
	useTempConfigDir(t)
	want := Config{DataDir: `D:\omniproxy-data`, PreviousDataDir: `C:\old`}
	if err := SaveConfig(Test, want); err != nil {
		t.Fatalf("SaveConfig error: %v", err)
	}
	got, err := LoadConfig(Test)
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if got != want {
		t.Fatalf("LoadConfig() = %+v, want %+v", got, want)
	}
}

func TestLoadConfigRejectsCorruptFile(t *testing.T) {
	dir := useTempConfigDir(t)
	if err := os.WriteFile(filepath.Join(dir, "app.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(Test); err == nil {
		t.Fatal("expected error for corrupt config")
	}
}

func TestSetDataDirValidatesAndPersists(t *testing.T) {
	useTempConfigDir(t)
	current := t.TempDir()
	target := filepath.Join(t.TempDir(), "new-location")

	cfg, err := SetDataDir(Test, current, target)
	if err != nil {
		t.Fatalf("SetDataDir error: %v", err)
	}
	if cfg.DataDir != target {
		t.Errorf("DataDir = %q, want %q", cfg.DataDir, target)
	}
	if cfg.PreviousDataDir != current {
		t.Errorf("PreviousDataDir = %q, want %q", cfg.PreviousDataDir, current)
	}
	if _, err := os.Stat(target); err != nil {
		t.Errorf("target directory was not created: %v", err)
	}

	stored, err := LoadConfig(Test)
	if err != nil {
		t.Fatal(err)
	}
	if stored != cfg {
		t.Errorf("stored config = %+v, want %+v", stored, cfg)
	}
}

func TestSetDataDirRejectsEmptyAndSameDir(t *testing.T) {
	useTempConfigDir(t)
	current := t.TempDir()

	if _, err := SetDataDir(Test, current, "   "); err == nil {
		t.Error("expected error for empty path")
	}
	if _, err := SetDataDir(Test, current, current); err != ErrSameDataDir {
		t.Errorf("error = %v, want ErrSameDataDir", err)
	}
}

func TestSetDataDirRejectsNestedPaths(t *testing.T) {
	useTempConfigDir(t)
	current := t.TempDir()
	nested := filepath.Join(current, "inside")

	if _, err := SetDataDir(Test, current, nested); err == nil {
		t.Error("expected error when new dir is inside the current dir")
	}
	if _, err := SetDataDir(Test, nested, current); err == nil {
		t.Error("expected error when current dir is inside the new dir")
	}
}

func TestSetDataDirRejectsUnwritableTarget(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses filesystem permissions")
	}
	useTempConfigDir(t)
	// A regular file cannot serve as a directory.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(blocker, "sub")
	if _, err := SetDataDir(Test, t.TempDir(), target); err == nil {
		t.Error("expected error for unwritable target")
	}
}

func TestEffectiveDataDirGuardsInterruptedMigration(t *testing.T) {
	oldDir := t.TempDir()
	newDir := t.TempDir()
	dbFile := filepath.Join(oldDir, databaseFileName)
	if err := os.WriteFile(dbFile, []byte("db"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := Config{DataDir: newDir, PreviousDataDir: oldDir}
	if got := cfg.effectiveDataDir(); got != oldDir {
		t.Fatalf("effectiveDataDir() = %q, want revert to %q", got, oldDir)
	}

	// Once the database exists at the new location, the new location wins.
	if err := os.WriteFile(filepath.Join(newDir, databaseFileName), []byte("db"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := cfg.effectiveDataDir(); got != newDir {
		t.Fatalf("effectiveDataDir() = %q, want %q", got, newDir)
	}
}

func TestEffectiveDataDirWithoutPrevious(t *testing.T) {
	dir := t.TempDir()
	cfg := Config{DataDir: dir}
	if got := cfg.effectiveDataDir(); got != dir {
		t.Fatalf("effectiveDataDir() = %q, want %q", got, dir)
	}
	if got := (Config{}).effectiveDataDir(); got != "" {
		t.Fatalf("empty config effectiveDataDir() = %q, want empty", got)
	}
}

func TestRevertDataDirRestoresPrevious(t *testing.T) {
	useTempConfigDir(t)
	if err := SaveConfig(Test, Config{DataDir: `D:\new`, PreviousDataDir: `C:\old`}); err != nil {
		t.Fatal(err)
	}
	if err := RevertDataDir(Test); err != nil {
		t.Fatalf("RevertDataDir error: %v", err)
	}
	cfg, err := LoadConfig(Test)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataDir != `C:\old` || cfg.PreviousDataDir != "" {
		t.Fatalf("after revert: %+v", cfg)
	}
}

func TestRevertDataDirWithoutPreviousIsNoop(t *testing.T) {
	useTempConfigDir(t)
	if err := SaveConfig(Test, Config{DataDir: `D:\new`}); err != nil {
		t.Fatal(err)
	}
	if err := RevertDataDir(Test); err != nil {
		t.Fatalf("RevertDataDir error: %v", err)
	}
	cfg, err := LoadConfig(Test)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataDir != `D:\new` {
		t.Fatalf("DataDir = %q, want unchanged", cfg.DataDir)
	}
}

func TestDetectAppliesStoredDataDir(t *testing.T) {
	useTempConfigDir(t)
	stored := t.TempDir()
	if err := SaveConfig(Test, Config{DataDir: stored}); err != nil {
		t.Fatal(err)
	}

	t.Setenv(envProfile, "test")
	p, err := Detect()
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}
	if p.DataDir != stored {
		t.Fatalf("DataDir = %q, want stored %q", p.DataDir, stored)
	}
	if want := filepath.Join(stored, "logs"); p.LogDir != want {
		t.Errorf("LogDir = %q, want %q", p.LogDir, want)
	}
}

func TestDetectEnvDataDirWinsOverStored(t *testing.T) {
	useTempConfigDir(t)
	stored := t.TempDir()
	envDir := t.TempDir()
	if err := SaveConfig(Test, Config{DataDir: stored}); err != nil {
		t.Fatal(err)
	}

	t.Setenv(envProfile, "test")
	t.Setenv(envDataDir, envDir)
	p, err := Detect()
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}
	if p.DataDir != envDir {
		t.Fatalf("DataDir = %q, want env %q", p.DataDir, envDir)
	}
}

func TestDetectRevertsInterruptedMigration(t *testing.T) {
	useTempConfigDir(t)
	oldDir := t.TempDir()
	newDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(oldDir, databaseFileName), []byte("db"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SaveConfig(Test, Config{DataDir: newDir, PreviousDataDir: oldDir}); err != nil {
		t.Fatal(err)
	}

	t.Setenv(envProfile, "test")
	p, err := Detect()
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}
	if p.DataDir != oldDir {
		t.Fatalf("DataDir = %q, want reverted %q", p.DataDir, oldDir)
	}

	// The correction must be persisted so the next start agrees.
	cfg, err := LoadConfig(Test)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataDir != oldDir || cfg.PreviousDataDir != "" {
		t.Fatalf("persisted config after revert: %+v", cfg)
	}
}
