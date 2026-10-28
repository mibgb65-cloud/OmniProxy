package wails

import (
	"context"
	"net"
	"os"
	"strconv"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"

	"omniproxy/internal/application/gateway"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/platform/profile"
)

func startupGatewayService(t *testing.T) (*AppService, *profile.Profile) {
	t.Helper()
	service, prof := newTestService(t)
	service.gateway = gateway.New(proxy.Options{Port: 0}, nil, nil, nil)
	t.Cleanup(func() {
		if err := service.shutdownGateway(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return service, prof
}

func occupyPort(t *testing.T) (int, net.Listener) {
	t.Helper()
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return listener.Addr().(*net.TCPAddr).Port, listener
}

func TestGatewayStartupDefaultsToStoppedAndPreferenceDoesNotStartImmediately(t *testing.T) {
	service, _ := startupGatewayService(t)
	if err := service.startGatewayOnStartup(); err != nil {
		t.Fatal(err)
	}
	if got, err := service.DesktopSettings(); err != nil || got.GatewayAutoStart {
		t.Fatalf("default desktop preference: %+v, %v", got, err)
	}
	if enabled, err := service.SetGatewayAutoStart(true); err != nil || !enabled {
		t.Fatalf("save preference: %t, %v", enabled, err)
	}
	if service.gateway.Info().Runtime.State != "stopped" {
		t.Fatal("preference change started gateway immediately")
	}
	if err := service.startGatewayOnStartup(); err != nil {
		t.Fatal(err)
	}
	if got, err := service.DesktopSettings(); err != nil || !got.GatewayAutoStart || service.gateway.Info().Runtime.State != "running" {
		t.Fatalf("enabled preference ignored: %+v, %v", got, err)
	}
	if _, err := service.SetGatewayAutoStart(false); err != nil {
		t.Fatal(err)
	}
	if service.gateway.Info().Runtime.State != "running" {
		t.Fatal("preference change stopped a running gateway")
	}
	if _, err := service.GatewayStop(); err != nil {
		t.Fatal(err)
	}
	if err := service.startGatewayOnStartup(); err != nil || service.gateway.Info().Runtime.State != "stopped" {
		t.Fatalf("disabled preference ignored: %v", err)
	}
}

func TestGatewayStartupAvoidsOccupiedPort(t *testing.T) {
	service, prof := startupGatewayService(t)
	occupied, listener := occupyPort(t)
	defer listener.Close()
	service.gateway = gateway.New(proxy.Options{Port: occupied}, nil, nil, nil)
	prof.ProxyPort = occupied
	if _, err := service.SetGatewayAutoStart(true); err != nil {
		t.Fatal(err)
	}
	if err := service.startGatewayOnStartup(); err != nil {
		t.Fatalf("startup did not avoid the occupied port: %v", err)
	}
	info := service.gateway.Info()
	if info.Runtime.State != "running" {
		t.Fatalf("gateway not running: %+v", info.Runtime)
	}
	if info.Runtime.Port == occupied {
		t.Fatalf("gateway still listens on the occupied port %d", occupied)
	}
	if prof.ProxyPort != info.Runtime.Port {
		t.Fatalf("profile port %d and runtime port %d disagree", prof.ProxyPort, info.Runtime.Port)
	}
	cfg, err := profile.LoadConfig(prof.Kind)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProxyPort != info.Runtime.Port {
		t.Fatalf("persisted port = %d, want %d", cfg.ProxyPort, info.Runtime.Port)
	}
}

func TestGatewayStartupKeepsEnvironmentLockedPortConflict(t *testing.T) {
	service, prof := startupGatewayService(t)
	occupied, listener := occupyPort(t)
	defer listener.Close()
	t.Setenv("OMNIPROXY_PROXY_PORT", strconv.Itoa(occupied))
	service.gateway = gateway.New(proxy.Options{Port: occupied}, nil, nil, nil)
	prof.ProxyPort = occupied
	if _, err := service.SetGatewayAutoStart(true); err != nil {
		t.Fatal(err)
	}
	if err := service.startGatewayOnStartup(); err == nil {
		t.Fatal("environment-locked port conflict was silently avoided")
	}
	cfg, err := profile.LoadConfig(prof.Kind)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProxyPort != 0 {
		t.Fatalf("environment-locked conflict persisted port %d", cfg.ProxyPort)
	}
}

func TestGatewayStartupReportsNonPortFailure(t *testing.T) {
	service, _ := startupGatewayService(t)
	if _, err := service.SetGatewayAutoStart(true); err != nil {
		t.Fatal(err)
	}
	// 网关不可用不是端口冲突：错误必须原样上报，不能触发避让。
	service.gateway = nil
	if err := service.startGatewayOnStartup(); err == nil {
		t.Fatal("missing gateway did not report an error")
	}
}

func TestGatewayStartupDoesNotOverwriteCorruptConfig(t *testing.T) {
	service, prof := startupGatewayService(t)
	if err := os.WriteFile(profile.ConfigFilePath(prof.Kind), []byte(`{"gatewayAutoStart":"yes"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := service.startGatewayOnStartup(); err == nil {
		t.Fatal("invalid config silently ignored")
	}
	if _, err := service.SetGatewayAutoStart(true); err == nil {
		t.Fatal("invalid config silently replaced")
	}
	if service.gateway.Info().Runtime.State != "stopped" {
		t.Fatal("gateway started despite invalid config")
	}
}

func TestServiceStartupSurvivesGatewayPortConflict(t *testing.T) {
	service, prof := startupGatewayService(t)
	occupied, listener := occupyPort(t)
	defer listener.Close()
	service.gateway = gateway.New(proxy.Options{Port: occupied}, nil, nil, nil)
	prof.ProxyPort = occupied
	if _, err := service.SetGatewayAutoStart(true); err != nil {
		t.Fatal(err)
	}
	service.prof.ControlPort = 0
	if err := service.ServiceStartup(context.Background(), application.ServiceOptions{}); err != nil {
		t.Fatalf("gateway conflict prevented desktop startup: %v", err)
	}
	t.Cleanup(func() {
		if err := service.control.Shutdown(context.Background()); err != nil {
			t.Error(err)
		}
	})
	if service.gateway.Info().Runtime.State != "running" {
		t.Fatal("gateway did not start on an alternative port")
	}
	if service.gateway.Info().Runtime.Port == occupied {
		t.Fatal("gateway kept the occupied port")
	}
}
