package profile

import (
	"path/filepath"
	"testing"
)

func TestNewResolvesDefaultsPerKind(t *testing.T) {
	tests := []struct {
		kind         Kind
		wantProxy    int
		wantControl  int
		wantDirSufix string
	}{
		{Production, 3000, 3890, AppName},
		{Development, 3001, 3891, AppName + "-dev"},
		{Test, 3002, 3892, AppName + "-test"},
	}
	for _, tt := range tests {
		t.Run(string(tt.kind), func(t *testing.T) {
			p, err := New(tt.kind)
			if err != nil {
				t.Fatalf("New(%q) error: %v", tt.kind, err)
			}
			if p.ProxyPort != tt.wantProxy || p.ControlPort != tt.wantControl {
				t.Errorf("ports = %d/%d, want %d/%d", p.ProxyPort, p.ControlPort, tt.wantProxy, tt.wantControl)
			}
			if filepath.Base(p.DataDir) != tt.wantDirSufix {
				t.Errorf("data dir = %q, want suffix %q", p.DataDir, tt.wantDirSufix)
			}
			if want := filepath.Join(p.DataDir, "omniproxy.db"); p.DatabasePath() != want {
				t.Errorf("database path = %q, want %q", p.DatabasePath(), want)
			}
			if want := filepath.Join(p.DataDir, "logs"); p.LogDir != want {
				t.Errorf("log dir = %q, want %q", p.LogDir, want)
			}
		})
	}
}

func TestNewRejectsUnknownKind(t *testing.T) {
	if _, err := New(Kind("staging")); err == nil {
		t.Fatal("expected error for unknown kind")
	}
}

func TestProfilesAreIsolated(t *testing.T) {
	prod, err := New(Production)
	if err != nil {
		t.Fatal(err)
	}
	dev, err := New(Development)
	if err != nil {
		t.Fatal(err)
	}
	if prod.DataDir == dev.DataDir {
		t.Fatalf("production and development share data dir %q", prod.DataDir)
	}
	if prod.ControlPort == dev.ControlPort {
		t.Fatalf("production and development share control port %d", prod.ControlPort)
	}
}

func TestNewHonoursEnvOverrides(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv(envDataDir, dataDir)
	t.Setenv(envProxyPort, "3999")
	t.Setenv(envControlPort, "3998")

	p, err := New(Development)
	if err != nil {
		t.Fatal(err)
	}
	if p.DataDir != dataDir {
		t.Errorf("data dir = %q, want %q", p.DataDir, dataDir)
	}
	if p.ProxyPort != 3999 || p.ControlPort != 3998 {
		t.Fatalf("ports = %d/%d, want 3999/3998", p.ProxyPort, p.ControlPort)
	}
}

func TestNewRejectsInvalidPortOverride(t *testing.T) {
	t.Setenv(envProxyPort, "not-a-port")
	if _, err := New(Development); err == nil {
		t.Fatal("expected error for invalid port override")
	}
}

func TestDetectUsesProfileEnv(t *testing.T) {
	useTempConfigDir(t)
	t.Setenv(envProfile, "PRODUCTION")
	p, err := Detect()
	if err != nil {
		t.Fatal(err)
	}
	if p.Kind != Production {
		t.Fatalf("kind = %q, want %q", p.Kind, Production)
	}
	if !p.IsProduction() {
		t.Fatal("IsProduction() = false, want true")
	}
}

func TestDetectFallsBackToBuildDefault(t *testing.T) {
	useTempConfigDir(t)
	p, err := Detect()
	if err != nil {
		t.Fatal(err)
	}
	if p.Kind != defaultKind {
		t.Fatalf("kind = %q, want default %q", p.Kind, defaultKind)
	}
}

func TestSetProxyPortValidatesAndPersists(t *testing.T) {
	useTempConfigDir(t)
	if err := SetProxyPort(Test, 43210); err != nil {
		t.Fatalf("SetProxyPort error: %v", err)
	}
	cfg, err := LoadConfig(Test)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProxyPort != 43210 {
		t.Fatalf("ProxyPort = %d, want 43210", cfg.ProxyPort)
	}
	// Zero restores the profile default and must stay persisted as "unset".
	if err := SetProxyPort(Test, 0); err != nil {
		t.Fatalf("SetProxyPort(0) error: %v", err)
	}
	if cfg, err = LoadConfig(Test); err != nil || cfg.ProxyPort != 0 {
		t.Fatalf("ProxyPort after reset = %d, %v", cfg.ProxyPort, err)
	}
	for _, port := range []int{-1, maxProxyPort + 1} {
		if err := SetProxyPort(Test, port); err == nil {
			t.Fatalf("SetProxyPort(%d) accepted an invalid port", port)
		}
	}
}

func TestDetectAppliesStoredProxyPort(t *testing.T) {
	useTempConfigDir(t)
	t.Setenv(envProfile, "test")
	if err := SetProxyPort(Test, 43111); err != nil {
		t.Fatal(err)
	}
	p, err := Detect()
	if err != nil {
		t.Fatal(err)
	}
	if p.ProxyPort != 43111 {
		t.Fatalf("ProxyPort = %d, want stored 43111", p.ProxyPort)
	}
	if p.ControlPort != testControlPort {
		t.Fatalf("ControlPort = %d, want %d", p.ControlPort, testControlPort)
	}
}

func TestDetectEnvProxyPortWinsOverStored(t *testing.T) {
	useTempConfigDir(t)
	t.Setenv(envProfile, "test")
	if err := SetProxyPort(Test, 43112); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envProxyPort, "43113")
	p, err := Detect()
	if err != nil {
		t.Fatal(err)
	}
	if p.ProxyPort != 43113 {
		t.Fatalf("ProxyPort = %d, want env 43113", p.ProxyPort)
	}
	if !ProxyPortEnvOverride() {
		t.Fatal("ProxyPortEnvOverride() = false, want true")
	}
}

func TestDetectIgnoresInvalidStoredProxyPort(t *testing.T) {
	useTempConfigDir(t)
	t.Setenv(envProfile, "test")
	if err := SaveConfig(Test, Config{ProxyPort: maxProxyPort + 1}); err != nil {
		t.Fatal(err)
	}
	p, err := Detect()
	if err != nil {
		t.Fatal(err)
	}
	if want := DefaultProxyPort(Test); p.ProxyPort != want {
		t.Fatalf("ProxyPort = %d, want default %d", p.ProxyPort, want)
	}
}

func TestDefaultProxyPortMatchesProfileDefaults(t *testing.T) {
	for kind, want := range map[Kind]int{Production: 3000, Development: 3001, Test: 3002} {
		if got := DefaultProxyPort(kind); got != want {
			t.Errorf("DefaultProxyPort(%q) = %d, want %d", kind, got, want)
		}
	}
}
