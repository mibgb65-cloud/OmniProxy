// Package profile defines the OmniProxy runtime profile.
//
// Production, development and test environments are strictly isolated: each
// profile resolves its own application ID, data directory, proxy port, control
// port and log directory. Development and production must never share a data
// directory (see docs/architecture/OVERVIEW.md and 技术架构选择.md §24).
package profile

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Kind identifies a runtime profile.
type Kind string

// Supported runtime profiles.
const (
	Production  Kind = "production"
	Development Kind = "development"
	Test        Kind = "test"
)

// Default network ports per profile (技术架构选择.md §24).
const (
	productionProxyPort    = 3000
	productionControlPort  = 3890
	developmentProxyPort   = 3001
	developmentControlPort = 3891
	testProxyPort          = 3002
	testControlPort        = 3892
)

// AppName is the product name used for directory and identifier resolution.
const AppName = "OmniProxy"

// Proxy port bounds accepted from the persisted configuration and the UI.
const (
	minProxyPort = 1
	maxProxyPort = 65535
)

// DefaultProxyPort returns the built-in proxy port of a profile. It is the
// value used when the user never changed the port.
func DefaultProxyPort(kind Kind) int {
	switch kind {
	case Development:
		return developmentProxyPort
	case Test:
		return testProxyPort
	default:
		return productionProxyPort
	}
}

// ProxyPortEnvOverride reports whether OMNIPROXY_PROXY_PORT pins the proxy
// port. The environment always wins over the persisted preference, so the UI
// must not let the user believe a change took effect.
func ProxyPortEnvOverride() bool {
	return strings.TrimSpace(os.Getenv(envProxyPort)) != ""
}

// Environment variable names honoured by the profile resolver.
const (
	envProfile     = "OMNIPROXY_PROFILE"
	envDataDir     = "OMNIPROXY_DATA_DIR"
	envConfigDir   = "OMNIPROXY_CONFIG_DIR"
	envProxyPort   = "OMNIPROXY_PROXY_PORT"
	envControlPort = "OMNIPROXY_CONTROL_PORT"
)

// Profile describes a fully resolved runtime environment.
type Profile struct {
	Kind        Kind
	AppID       string
	DataDir     string
	LogDir      string
	ProxyPort   int
	ControlPort int
}

// New resolves a profile of the given kind, honouring environment overrides.
func New(kind Kind) (*Profile, error) {
	p := &Profile{Kind: kind}
	switch kind {
	case Production:
		p.AppID = "com.omniproxy.app"
		p.ProxyPort, p.ControlPort = productionProxyPort, productionControlPort
		p.DataDir = defaultDataDir(AppName)
	case Development:
		p.AppID = "com.omniproxy.app.dev"
		p.ProxyPort, p.ControlPort = developmentProxyPort, developmentControlPort
		p.DataDir = defaultDataDir(AppName + "-dev")
	case Test:
		p.AppID = "com.omniproxy.app.test"
		p.ProxyPort, p.ControlPort = testProxyPort, testControlPort
		p.DataDir = defaultDataDir(AppName + "-test")
	default:
		return nil, fmt.Errorf("profile: unknown kind %q", kind)
	}
	if err := p.applyEnvOverrides(); err != nil {
		return nil, err
	}
	p.LogDir = filepath.Join(p.DataDir, "logs")
	return p, nil
}

// Detect resolves the profile for the current process. The OMNIPROXY_PROFILE
// environment variable overrides the build-tag default.
//
// The proxy port resolves in this order: OMNIPROXY_PROXY_PORT > persisted
// app.json > profile default.
//
// The data directory resolves in this order: OMNIPROXY_DATA_DIR > persisted
// app.json > profile default. An interrupted relocation (a crash between
// persisting the new location and moving the data at shutdown) is detected
// and reverted here.
func Detect() (*Profile, error) {
	kind := defaultKind
	if v := strings.TrimSpace(os.Getenv(envProfile)); v != "" {
		kind = Kind(strings.ToLower(v))
	}
	p, err := New(kind)
	if err != nil {
		return nil, err
	}
	cfg, err := LoadConfig(kind)
	if err != nil {
		return nil, err
	}
	// The persisted port only replaces the profile default: OMNIPROXY_PROXY_PORT
	// still wins, and a corrupt value falls back to the default instead of
	// preventing the application from starting.
	if !ProxyPortEnvOverride() && cfg.ProxyPort >= minProxyPort && cfg.ProxyPort <= maxProxyPort {
		p.ProxyPort = cfg.ProxyPort
	}
	if strings.TrimSpace(os.Getenv(envDataDir)) != "" {
		return p, nil // an explicit environment override always wins
	}
	// The installer choice only replaces the built-in default: the persisted
	// app.json preference (below) still wins, so a relocation made in the app
	// is never undone by a reinstall.
	if dir := installDataDir(); dir != "" && !pathsEqual(dir, p.DataDir) {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			slog.Warn("installer data directory is unavailable, using the profile default",
				"dir", dir, "default", p.DataDir, "error", err)
		} else {
			p.DataDir = dir
			p.LogDir = filepath.Join(dir, "logs")
		}
	}
	dir := cfg.effectiveDataDir()
	if dir == "" || pathsEqual(dir, p.DataDir) {
		return p, nil
	}
	if !pathsEqual(dir, cfg.DataDir) {
		// The stored location has no data while the previous one does: the
		// shutdown migration never completed. Persist the correction.
		if err := RevertDataDir(kind); err != nil {
			slog.Error("failed to persist data directory revert", "error", err)
		}
	}
	p.DataDir = dir
	p.LogDir = filepath.Join(dir, "logs")
	return p, nil
}

// DatabasePath returns the SQLite database file location for this profile.
func (p *Profile) DatabasePath() string {
	return filepath.Join(p.DataDir, databaseFileName)
}

// IsProduction reports whether this is the production profile.
func (p *Profile) IsProduction() bool { return p.Kind == Production }

func (p *Profile) applyEnvOverrides() error {
	if v := strings.TrimSpace(os.Getenv(envDataDir)); v != "" {
		p.DataDir = v
	}
	port, err := envInt(envProxyPort)
	if err != nil {
		return err
	}
	if port != 0 {
		p.ProxyPort = port
	}
	port, err = envInt(envControlPort)
	if err != nil {
		return err
	}
	if port != 0 {
		p.ControlPort = port
	}
	return nil
}

func envInt(name string) (int, error) {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return 0, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("profile: invalid %s=%q: %w", name, raw, err)
	}
	return v, nil
}

// defaultDataDir resolves the per-user application data directory. It uses the
// OS configuration directory (e.g. %APPDATA% on Windows) and never falls back
// to the current working directory.
func defaultDataDir(name string) string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return name
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, name)
}
