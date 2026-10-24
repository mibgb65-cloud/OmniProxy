package profile

import (
	"os"
	"path/filepath"
	"testing"
	"unicode/utf16"
)

// useInstallDir simulates an installation directory with an optional
// data-dir.txt file and restores the real executable lookup afterwards.
func useInstallDir(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if content != "" {
		if err := os.WriteFile(filepath.Join(dir, installDataDirFileName), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	previous := executablePath
	executablePath = func() (string, error) { return filepath.Join(dir, "omniproxy.exe"), nil }
	t.Cleanup(func() { executablePath = previous })
	return dir
}

func TestParseInstallDataDir(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{"empty", "", ""},
		{"whitespace", "  \r\n\t", ""},
		{"relative", `relative\data`, ""},
		{"plain", filepath.Clean(`D:\OmniProxyData`), filepath.Clean(`D:\OmniProxyData`)},
		{"trailing newline", filepath.Clean(`D:\OmniProxyData`) + "\r\n", filepath.Clean(`D:\OmniProxyData`)},
		{"quoted", `"` + `D:\OmniProxy Data` + `"`, filepath.Clean(`D:\OmniProxy Data`)},
		{"bom", "\ufeff" + filepath.Clean(`D:\OmniProxyData`), filepath.Clean(`D:\OmniProxyData`)},
		{"nested spaces", filepath.Clean(`D:\My Data\OmniProxy`), filepath.Clean(`D:\My Data\OmniProxy`)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := parseInstallDataDir(tc.raw); got != tc.want {
				t.Fatalf("parseInstallDataDir(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}

func TestReadInstallDataDirMissingFile(t *testing.T) {
	dir := useInstallDir(t, "")
	if got := readInstallDataDir(dir); got != "" {
		t.Fatalf("readInstallDataDir() = %q, want empty", got)
	}
}

// NSIS built with `Unicode true` writes UTF-16LE with a BOM.
func TestReadInstallDataDirDecodesUnicodeInstallerFile(t *testing.T) {
	dir := useInstallDir(t, "")
	target := filepath.Clean(`D:\OmniProxy 数据`)
	data := []byte{0xFF, 0xFE}
	for _, unit := range utf16.Encode([]rune(target + "\r\n")) {
		data = append(data, byte(unit), byte(unit>>8))
	}
	if err := os.WriteFile(filepath.Join(dir, installDataDirFileName), data, 0o600); err != nil {
		t.Fatal(err)
	}
	if got := readInstallDataDir(dir); got != target {
		t.Fatalf("readInstallDataDir() = %q, want %q", got, target)
	}
}

func TestDetectUsesInstallerDataDir(t *testing.T) {
	useTempConfigDir(t)
	target := t.TempDir()
	useInstallDir(t, target+"\r\n")
	t.Setenv(envProfile, "test")

	p, err := Detect()
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}
	if p.DataDir != filepath.Clean(target) {
		t.Fatalf("DataDir = %q, want %q", p.DataDir, filepath.Clean(target))
	}
	if want := filepath.Join(target, "logs"); p.LogDir != want {
		t.Errorf("LogDir = %q, want %q", p.LogDir, want)
	}
}

func TestDetectStoredDataDirWinsOverInstaller(t *testing.T) {
	useTempConfigDir(t)
	stored := t.TempDir()
	if err := SaveConfig(Test, Config{DataDir: stored}); err != nil {
		t.Fatal(err)
	}
	useInstallDir(t, t.TempDir())
	t.Setenv(envProfile, "test")

	p, err := Detect()
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}
	if p.DataDir != stored {
		t.Fatalf("DataDir = %q, want stored %q", p.DataDir, stored)
	}
}

func TestDetectEnvWinsOverInstaller(t *testing.T) {
	useTempConfigDir(t)
	envDir := t.TempDir()
	useInstallDir(t, t.TempDir())
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

func TestDetectFallsBackWhenInstallerDirUnusable(t *testing.T) {
	useTempConfigDir(t)
	parent := t.TempDir()
	blocked := filepath.Join(parent, "blocked")
	if err := os.WriteFile(blocked, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	useInstallDir(t, blocked)
	t.Setenv(envProfile, "test")

	p, err := Detect()
	if err != nil {
		t.Fatalf("Detect error: %v", err)
	}
	if want := defaultDataDir(configDirName(Test)); p.DataDir != want {
		t.Fatalf("DataDir = %q, want fallback %q", p.DataDir, want)
	}
}
