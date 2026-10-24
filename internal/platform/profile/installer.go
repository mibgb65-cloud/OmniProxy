package profile

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ErrDataDirInUse is returned when the installer tries to switch away from a
// data directory that already holds application data. Callers surface it as
// "relocate from inside the app instead" rather than as a generic failure.
var ErrDataDirInUse = errors.New("profile: data directory still contains application data")

// InstallerDataDir reports the data directory the application currently uses,
// so the Windows installer can show it on an upgrade instead of guessing.
func InstallerDataDir(kind Kind) (string, error) {
	p, err := New(kind)
	if err != nil {
		return "", err
	}
	cfg, err := LoadConfig(kind)
	if err != nil {
		return "", err
	}
	if dir := cfg.effectiveDataDir(); dir != "" {
		return dir, nil
	}
	return p.DataDir, nil
}

// InstallerSetDataDir records the data directory chosen during installation.
//
// The installer may only choose a location that has no data yet: switching away
// from a directory that already holds a database would leave the accounts,
// credentials and request history there while the application starts on an
// empty directory, which looks like a fresh install. Relocating existing data
// stays a user action inside the application, where it is moved and can be
// reverted.
func InstallerSetDataDir(kind Kind, dir string) error {
	current, err := InstallerDataDir(kind)
	if err != nil {
		return err
	}
	target := strings.TrimSpace(dir)
	if target == "" {
		return fmt.Errorf("profile: data directory must not be empty")
	}
	if !filepath.IsAbs(target) {
		return fmt.Errorf("profile: %q is not an absolute path", target)
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("profile: resolve %q: %w", target, err)
	}
	if pathsEqual(abs, current) {
		return nil
	}
	if containsDatabase(current) {
		return fmt.Errorf("%w: %s", ErrDataDirInUse, current)
	}
	if isNested(abs, current) || isNested(current, abs) {
		return fmt.Errorf("profile: %q and %q must not contain each other", current, abs)
	}
	if err := ensureWritableDir(abs); err != nil {
		return err
	}
	cfg, err := LoadConfig(kind)
	if err != nil {
		return err
	}
	cfg.DataDir = abs
	cfg.PreviousDataDir = ""
	return SaveConfig(kind, cfg)
}

// containsDatabase reports whether dir already holds an OmniProxy database,
// including the isolated v2 layout used by upgraded installations.
func containsDatabase(dir string) bool {
	if dir == "" {
		return false
	}
	if fileExists(filepath.Join(dir, databaseFileName)) {
		return true
	}
	return fileExists(filepath.Join(dir, "v2", databaseFileName))
}
