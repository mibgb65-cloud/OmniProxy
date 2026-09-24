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
