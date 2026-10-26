package profile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// isolateInstallerConfig keeps installer tests away from the real user config
// and from any ambient data-directory override.
func isolateInstallerConfig(t *testing.T) InstallerLocation {
	t.Helper()
	config := useTempConfigDir(t)
	t.Setenv(envDataDir, "")
	return InstallerLocation{
		ConfigDir: config, DefaultDataDir: filepath.Join(t.TempDir(), AppName),
		InstallDir: t.TempDir(),
	}
}

func TestInstallerDataDirDefaultsWithoutConfig(t *testing.T) {
	location := isolateInstallerConfig(t)

	got, err := location.DataDir()
	if err != nil {
		t.Fatalf("InstallerDataDir error: %v", err)
	}
	if want := location.DefaultDataDir; got != want {
		t.Fatalf("InstallerDataDir() = %q, want %q", got, want)
	}
}

func TestInstallerDataDirReportsStoredChoice(t *testing.T) {
	location := isolateInstallerConfig(t)
	stored := t.TempDir()
	if err := SaveConfig(Production, Config{DataDir: stored}); err != nil {
		t.Fatal(err)
	}

	got, err := location.DataDir()
	if err != nil {
		t.Fatalf("InstallerDataDir error: %v", err)
	}
	if got != stored {
		t.Fatalf("InstallerDataDir() = %q, want %q", got, stored)
	}
}

func TestInstallerSetDataDirPersistsChoice(t *testing.T) {
	location := isolateInstallerConfig(t)
	current := t.TempDir()
	if err := SaveConfig(Production, Config{DataDir: current}); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "OmniProxyData")

	if err := location.SetDataDir(target); err != nil {
		t.Fatalf("InstallerSetDataDir error: %v", err)
	}
	cfg, err := LoadConfig(Production)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataDir != target || cfg.PreviousDataDir != "" {
		t.Fatalf("config after installer write: %+v", cfg)
	}
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		t.Fatalf("target directory was not created: %v", err)
	}
	if got, err := location.DataDir(); err != nil || got != target {
		t.Fatalf("InstallerDataDir() = %q (err %v), want %q", got, err, target)
	}
}

func TestInstallerSetDataDirRefusesToAbandonExistingData(t *testing.T) {
	for _, name := range []string{"database", "isolated v2 database"} {
		t.Run(name, func(t *testing.T) {
			location := isolateInstallerConfig(t)
			current := t.TempDir()
			if err := SaveConfig(Production, Config{DataDir: current}); err != nil {
				t.Fatal(err)
			}
			dbPath := filepath.Join(current, databaseFileName)
			if name == "isolated v2 database" {
				dbPath = filepath.Join(current, "v2", databaseFileName)
				if err := os.MkdirAll(filepath.Dir(dbPath), 0o750); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(dbPath, []byte("db"), 0o600); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(t.TempDir(), "elsewhere")

			if err := location.SetDataDir(target); !errors.Is(err, ErrDataDirInUse) {
				t.Fatalf("expected ErrDataDirInUse, got %v", err)
			}
			cfg, err := LoadConfig(Production)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.DataDir != current {
				t.Fatalf("config changed to %q, want %q", cfg.DataDir, current)
			}
			if _, err := os.Stat(target); !os.IsNotExist(err) {
				t.Fatalf("target directory must not be created on refusal: %v", err)
			}
		})
	}
}

func TestInstallerSetDataDirAcceptsRepeatedChoice(t *testing.T) {
	location := isolateInstallerConfig(t)
	stored := t.TempDir()
	if err := SaveConfig(Production, Config{DataDir: stored, PreviousDataDir: `C:\old`}); err != nil {
		t.Fatal(err)
	}

	if err := location.SetDataDir(stored); err != nil {
		t.Fatalf("InstallerSetDataDir error: %v", err)
	}
	cfg, err := LoadConfig(Production)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DataDir != stored || cfg.PreviousDataDir != `C:\old` {
		t.Fatalf("unchanged choice must keep the config untouched: %+v", cfg)
	}
}

func TestInstallerSetDataDirRejectsInvalidInput(t *testing.T) {
	location := isolateInstallerConfig(t)
	current := t.TempDir()
	if err := SaveConfig(Production, Config{DataDir: current}); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"", "   ", "relative\\dir", filepath.Join(current, "nested")} {
		if err := location.SetDataDir(target); err == nil {
			t.Fatalf("InstallerSetDataDir(%q) accepted an invalid target", target)
		}
	}
}
