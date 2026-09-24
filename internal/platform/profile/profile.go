// Package profile defines the OmniProxy runtime profile.
//
// Production, development and test environments are strictly isolated: each
// profile resolves its own application ID, data directory, proxy port, control
// port and log directory. Development and production must never share a data
// directory (see docs/architecture/OVERVIEW.md and 技术架构选择.md §24).
package profile

import (
	"fmt"
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

// Environment variable names honoured by the profile resolver.
const (
	envProfile     = "OMNIPROXY_PROFILE"
	envDataDir     = "OMNIPROXY_DATA_DIR"
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
func Detect() (*Profile, error) {
	kind := defaultKind
	if v := strings.TrimSpace(os.Getenv(envProfile)); v != "" {
		kind = Kind(strings.ToLower(v))
	}
	return New(kind)
}

// DatabasePath returns the SQLite database file location for this profile.
func (p *Profile) DatabasePath() string {
	return filepath.Join(p.DataDir, "omniproxy.db")
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
