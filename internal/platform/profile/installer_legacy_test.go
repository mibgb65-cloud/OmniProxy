package profile

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestInstallerDataDirReadsLegacyInstallation(t *testing.T) {
	location := isolateInstallerConfig(t)
	legacy := t.TempDir()
	location.InstallDir = useInstallDir(t, legacy+"\r\n")
	got, err := location.DataDir()
	if err != nil || got != legacy {
		t.Fatalf("InstallerDataDir() = %q, %v; want legacy %q", got, err, legacy)
	}
}

func TestInstallerLegacyChoicePersistsWhenProgramDirectoryChanges(t *testing.T) {
	location := isolateInstallerConfig(t)
	legacy := t.TempDir()
	location.InstallDir = useInstallDir(t, legacy)
	if err := os.WriteFile(filepath.Join(legacy, databaseFileName), []byte("synthetic database"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := location.SetDataDir(legacy); err != nil {
		t.Fatal(err)
	}
	location.InstallDir = t.TempDir()
	got, err := location.DataDir()
	if err != nil || got != legacy {
		t.Fatalf("data location was lost after moving the program: %q %v", got, err)
	}
}

func TestInstallerStoredChoiceWinsOverOldMarkerAndEnvironment(t *testing.T) {
	location := isolateInstallerConfig(t)
	stored := t.TempDir()
	if err := SaveConfig(Production, Config{DataDir: stored, UpdateChannel: "beta"}); err != nil {
		t.Fatal(err)
	}
	location.InstallDir = useInstallDir(t, t.TempDir())
	t.Setenv(envConfigDir, t.TempDir())
	t.Setenv(envDataDir, t.TempDir())
	got, err := location.DataDir()
	if err != nil || got != stored {
		t.Fatalf("wrong precedence or inherited environment used: %q %v", got, err)
	}
	target := t.TempDir()
	if err := location.SetDataDir(target); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadConfigFile(filepath.Join(location.ConfigDir, configFileName))
	if err != nil || cfg.DataDir != target || cfg.UpdateChannel != "beta" {
		t.Fatalf("wrong config was saved or unrelated preferences lost: %+v %v", cfg, err)
	}
}

func TestInstallerFailsClosedForInvalidLegacyMarker(t *testing.T) {
	for _, marker := range []string{"", "relative/data", strings.Repeat("x", 16385), `C:\`} {
		t.Run("invalid marker", func(t *testing.T) {
			location := isolateInstallerConfig(t)
			if err := os.WriteFile(filepath.Join(location.InstallDir, installDataDirFileName), []byte(marker), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := location.SetDataDir(t.TempDir()); err == nil {
				t.Fatalf("unresolved legacy marker allowed changing the pointer: %q", marker[:min(len(marker), 20)])
			}
			if _, err := os.Stat(filepath.Join(location.ConfigDir, configFileName)); !os.IsNotExist(err) {
				t.Fatalf("unresolved legacy marker changed config: %v", err)
			}
		})
	}
}

func TestInstallerFailsClosedForUnreadableMetadata(t *testing.T) {
	location := isolateInstallerConfig(t)
	if err := os.Mkdir(filepath.Join(location.InstallDir, installDataDirFileName), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := location.DataDir(); err == nil {
		t.Fatal("unreadable marker silently fell back to default")
	}
}

func TestInstallerWriteProbePreservesExistingFile(t *testing.T) {
	location := isolateInstallerConfig(t)
	target := t.TempDir()
	probe := filepath.Join(target, ".omniproxy-write-test")
	want := "user-owned file"
	if err := os.WriteFile(probe, []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := location.SetDataDir(target); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(probe)
	if err != nil || string(got) != want {
		t.Fatalf("existing probe name was modified: %q %v", got, err)
	}
}

func TestInstallerReadsUnicodeLegacyMarker(t *testing.T) {
	location := isolateInstallerConfig(t)
	target := filepath.Join(t.TempDir(), "历史 数据")
	data := []byte{0xff, 0xfe}
	for _, unit := range utf16.Encode([]rune(target + "\r\n")) {
		data = append(data, byte(unit), byte(unit>>8))
	}
	if err := os.WriteFile(filepath.Join(location.InstallDir, installDataDirFileName), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := location.DataDir(); err != nil || got != target {
		t.Fatalf("Unicode legacy location was not retained: %q %v", got, err)
	}
}

func TestInstallerRejectsCorruptConfigAndNonRegularDatabase(t *testing.T) {
	location := isolateInstallerConfig(t)
	path := filepath.Join(location.ConfigDir, configFileName)
	for _, content := range []string{"{broken", strings.Repeat("x", (1<<20)+1)} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := location.SetDataDir(t.TempDir()); err == nil {
			t.Fatal("corrupt configuration was silently replaced")
		}
	}
	current := t.TempDir()
	if err := SaveConfig(Production, Config{DataDir: current}); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(current, databaseFileName), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := location.SetDataDir(t.TempDir()); err == nil {
		t.Fatal("an invalid database entry was silently ignored")
	}
}

func TestInstallerRefusesToAbandonLegacyDatabase(t *testing.T) {
	for _, layout := range []string{"omniproxy.db", filepath.Join("v2", "omniproxy.db")} {
		t.Run(layout, func(t *testing.T) {
			location := isolateInstallerConfig(t)
			legacy := t.TempDir()
			location.InstallDir = useInstallDir(t, legacy)
			database := filepath.Join(legacy, layout)
			if err := os.MkdirAll(filepath.Dir(database), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(database, []byte("synthetic database"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := location.SetDataDir(t.TempDir()); !errors.Is(err, ErrDataDirInUse) {
				t.Fatalf("legacy database was abandoned: %v", err)
			}
		})
	}
}
