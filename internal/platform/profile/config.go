package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// configFileName is the persisted application configuration file. It lives at
// a fixed location derived from the OS config dir, independent of the data
// directory, so the data-directory preference survives a relocation.
const configFileName = "app.json"

// databaseFileName is the SQLite database file inside the data directory.
const databaseFileName = "omniproxy.db"

// ErrSameDataDir is returned when the requested data directory equals the
// active one.
var ErrSameDataDir = errors.New("profile: data directory unchanged")

// Config is the persisted application configuration.
type Config struct {
	// DataDir is the data directory the application should use. Empty means
	// "use the profile default".
	DataDir string `json:"dataDir,omitempty"`
	// PreviousDataDir records the location active before the last change.
	// It lets startup detect an interrupted migration (a crash between
	// persisting the new location and moving the data) and revert.
	PreviousDataDir string `json:"previousDataDir,omitempty"`
	// UpdateRepo is the "owner/repo" checked for new releases on GitHub.
	// 留空时使用项目正式发布仓库。
	UpdateRepo string `json:"updateRepo,omitempty"`
	// UpdateChannel 留空时依据应用版本选择；明确选择不会在升级后被覆盖。
	UpdateChannel string `json:"updateChannel,omitempty"`
	// CloseBehavior controls what closing the main window does. Empty keeps
	// the historical default of exiting the application.
	CloseBehavior string `json:"closeBehavior,omitempty"`
}

// ConfigFilePath returns the fixed path of the persisted configuration for
// the given profile kind. OMNIPROXY_CONFIG_DIR overrides the directory (used
// by tests and portable installations); the file itself never moves with the
// data directory.
func ConfigFilePath(kind Kind) string {
	base := strings.TrimSpace(os.Getenv(envConfigDir))
	if base == "" {
		base = defaultDataDir(configDirName(kind))
	}
	return filepath.Join(base, configFileName)
}

// LoadConfig reads the persisted configuration. A missing file yields an
// empty configuration; a corrupt file is an error so the problem surfaces
// instead of silently resetting the user's choice.
func LoadConfig(kind Kind) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(ConfigFilePath(kind))
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("profile: read config: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("profile: parse %s: %w", ConfigFilePath(kind), err)
	}
	return cfg, nil
}

// SaveConfig atomically writes the configuration (temp file + rename).
func SaveConfig(kind Kind, cfg Config) error {
	path := ConfigFilePath(kind)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("profile: create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("profile: encode config: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("profile: write config: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("profile: replace config: %w", err)
	}
	return nil
}

// SetDataDir validates and persists a new data directory. The change takes
// effect on the next start; the caller performs the actual data move at
// shutdown (see AppService.ServiceShutdown).
func SetDataDir(kind Kind, currentDir, newDir string) (Config, error) {
	target := strings.TrimSpace(newDir)
	if target == "" {
		return Config{}, fmt.Errorf("profile: data directory must not be empty")
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return Config{}, fmt.Errorf("profile: resolve %q: %w", target, err)
	}
	if pathsEqual(abs, currentDir) {
		return Config{}, ErrSameDataDir
	}
	if isNested(abs, currentDir) || isNested(currentDir, abs) {
		return Config{}, fmt.Errorf("profile: %q and %q must not contain each other", currentDir, abs)
	}
	if err := ensureWritableDir(abs); err != nil {
		return Config{}, err
	}

	cfg, err := LoadConfig(kind)
	if err != nil {
		return Config{}, err
	}
	cfg.PreviousDataDir = currentDir
	cfg.DataDir = abs
	if err := SaveConfig(kind, cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// RevertDataDir restores the previous data directory in the persisted
// configuration. Used when a relocation fails at shutdown.
func RevertDataDir(kind Kind) error {
	cfg, err := LoadConfig(kind)
	if err != nil {
		return err
	}
	if cfg.PreviousDataDir == "" {
		return nil
	}
	cfg.DataDir = cfg.PreviousDataDir
	cfg.PreviousDataDir = ""
	return SaveConfig(kind, cfg)
}

// effectiveDataDir resolves the data directory from the persisted config,
// guarding against an interrupted migration: if the stored location has no
// database while the previous one does, the shutdown move never ran, so the
// previous location is still authoritative.
func (c Config) effectiveDataDir() string {
	if c.DataDir == "" {
		return ""
	}
	if c.PreviousDataDir == "" || pathsEqual(c.PreviousDataDir, c.DataDir) {
		return c.DataDir
	}
	newHasDB := fileExists(filepath.Join(c.DataDir, databaseFileName))
	oldHasDB := fileExists(filepath.Join(c.PreviousDataDir, databaseFileName))
	if !newHasDB && oldHasDB {
		return c.PreviousDataDir
	}
	return c.DataDir
}

// ensureWritableDir creates dir if needed and verifies it is writable.
func ensureWritableDir(dir string) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("profile: create %s: %w", dir, err)
	}
	probe := filepath.Join(dir, ".omniproxy-write-test")
	f, err := os.OpenFile(probe, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("profile: %s is not writable: %w", dir, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("profile: %s is not writable: %w", dir, err)
	}
	if err := os.Remove(probe); err != nil {
		return fmt.Errorf("profile: clean up write probe in %s: %w", dir, err)
	}
	return nil
}

func configDirName(kind Kind) string {
	switch kind {
	case Production:
		return AppName
	case Development:
		return AppName + "-dev"
	case Test:
		return AppName + "-test"
	default:
		return AppName
	}
}

func pathsEqual(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}

// isNested reports whether child is inside parent (or equal).
func isNested(child, parent string) bool {
	c := filepath.Clean(child)
	p := filepath.Clean(parent)
	if p == c {
		return true
	}
	rel, err := filepath.Rel(p, c)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
