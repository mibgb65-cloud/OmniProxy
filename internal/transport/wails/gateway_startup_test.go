package wails

import (
	"context"
	"net"
	"os"
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

func TestGatewayStartupReportsPortConflictAndAllowsManualRetry(t *testing.T) {
	service, _ := startupGatewayService(t)
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	service.gateway = gateway.New(proxy.Options{Port: listener.Addr().(*net.TCPAddr).Port}, nil, nil, nil)
	if _, err := service.SetGatewayAutoStart(true); err != nil {
		t.Fatal(err)
	}
	if err := service.startGatewayOnStartup(); err == nil {
		t.Fatal("port conflict silently ignored")
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
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := service.GatewayStart(); err != nil {
		t.Fatalf("manual retry failed: %v", err)
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
