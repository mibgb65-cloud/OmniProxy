package wails

import (
	"testing"

	"omniproxy/internal/platform/profile"
)

func TestProxyPortReportsProfileDefaults(t *testing.T) {
	service, prof := newTestService(t)
	state, err := service.ProxyPort()
	if err != nil {
		t.Fatal(err)
	}
	if state.Port != prof.ProxyPort {
		t.Fatalf("Port = %d, want %d", state.Port, prof.ProxyPort)
	}
	if want := profile.DefaultProxyPort(prof.Kind); state.DefaultPort != want {
		t.Fatalf("DefaultPort = %d, want %d", state.DefaultPort, want)
	}
	if state.EnvLocked || state.Running {
		t.Fatalf("state = %+v, want unlocked and stopped", state)
	}
}

func TestSetProxyPortPersistsAndRepointsGateway(t *testing.T) {
	service, prof := newTestService(t)
	target, listener := occupyPort(t)
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	update, err := service.SetProxyPort(target)
	if err != nil {
		t.Fatalf("SetProxyPort error: %v", err)
	}
	if update.Requested != target {
		t.Fatalf("Requested = %d, want %d", update.Requested, target)
	}
	if !update.Adjusted && update.State.Port != target {
		t.Fatalf("port = %d, want requested %d", update.State.Port, target)
	}
	if prof.ProxyPort != update.State.Port {
		t.Fatalf("profile port = %d, want %d", prof.ProxyPort, update.State.Port)
	}
	if got := service.AppInfo().ProxyPort; got != update.State.Port {
		t.Fatalf("AppInfo port = %d, want %d", got, update.State.Port)
	}
	if service.gateway != nil {
		if got := service.gateway.Info().Runtime.Port; got != update.State.Port {
			t.Fatalf("gateway port = %d, want %d", got, update.State.Port)
		}
	}
	cfg, err := profile.LoadConfig(prof.Kind)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProxyPort != update.State.Port {
		t.Fatalf("persisted port = %d, want %d", cfg.ProxyPort, update.State.Port)
	}
}

func TestSetProxyPortAvoidsOccupiedPort(t *testing.T) {
	service, prof := newTestService(t)
	occupied, listener := occupyPort(t)
	defer listener.Close()
	update, err := service.SetProxyPort(occupied)
	if err != nil {
		t.Fatalf("SetProxyPort error: %v", err)
	}
	if !update.Adjusted {
		t.Fatalf("occupied port %d was accepted: %+v", occupied, update)
	}
	if update.State.Port == occupied {
		t.Fatalf("gateway kept the occupied port %d", occupied)
	}
	if !update.State.Available {
		t.Fatalf("alternative port %d reported unavailable", update.State.Port)
	}
	cfg, err := profile.LoadConfig(prof.Kind)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProxyPort != update.State.Port {
		t.Fatalf("persisted port = %d, want %d", cfg.ProxyPort, update.State.Port)
	}
}

func TestSetProxyPortRestoresDefaultPort(t *testing.T) {
	service, prof := newTestService(t)
	if _, err := service.SetProxyPort(43210); err != nil {
		t.Fatal(err)
	}
	update, err := service.SetProxyPort(0)
	if err != nil {
		t.Fatalf("SetProxyPort(0) error: %v", err)
	}
	if update.Requested != 0 {
		t.Fatalf("Requested = %d, want 0", update.Requested)
	}
	if want := profile.DefaultProxyPort(prof.Kind); !update.Adjusted && update.State.Port != want {
		t.Fatalf("port = %d, want default %d", update.State.Port, want)
	}
}

func TestSetProxyPortKeepsRunningGateway(t *testing.T) {
	service, _ := startupGatewayService(t)
	if _, err := service.GatewayStart(); err != nil {
		t.Fatal(err)
	}
	target, listener := occupyPort(t)
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	update, err := service.SetProxyPort(target)
	if err != nil {
		t.Fatalf("SetProxyPort error: %v", err)
	}
	info := service.gateway.Info()
	if info.Runtime.State != "running" {
		t.Fatalf("gateway stopped after port change: %+v", info.Runtime)
	}
	if info.Runtime.Port != update.State.Port {
		t.Fatalf("gateway port = %d, want %d", info.Runtime.Port, update.State.Port)
	}
}

func TestSetProxyPortRejectsInvalidPort(t *testing.T) {
	service, _ := newTestService(t)
	for _, port := range []int{-1, -43210, 65536} {
		if _, err := service.SetProxyPort(port); err == nil {
			t.Fatalf("SetProxyPort(%d) accepted an invalid port", port)
		}
	}
}

func TestSetProxyPortRejectsEnvironmentLockedPort(t *testing.T) {
	service, _ := newTestService(t)
	t.Setenv("OMNIPROXY_PROXY_PORT", "43211")
	if _, err := service.SetProxyPort(43212); err == nil {
		t.Fatal("environment-locked port was overridden")
	}
	state, err := service.ProxyPort()
	if err != nil {
		t.Fatal(err)
	}
	if !state.EnvLocked {
		t.Fatal("state does not report the environment lock")
	}
}

func TestProxyPortReportsUnavailablePortWhenStopped(t *testing.T) {
	service, _ := newTestService(t)
	occupied, listener := occupyPort(t)
	defer listener.Close()
	service.prof.ProxyPort = occupied
	if _, err := service.GatewayStop(); err != nil {
		t.Fatal(err)
	}
	state, err := service.ProxyPort()
	if err != nil {
		t.Fatal(err)
	}
	if state.Port != occupied || state.Available || state.Running {
		t.Fatalf("state = %+v, want unavailable port %d", state, occupied)
	}
}

func TestProxyPortEnvLockedStateReportsPort(t *testing.T) {
	service, prof := newTestService(t)
	t.Setenv("OMNIPROXY_PROXY_PORT", "43213")
	state, err := service.ProxyPort()
	if err != nil {
		t.Fatal(err)
	}
	// 端口在启动时已解析；运行期只报告锁定状态，不改写已生效端口。
	if !state.EnvLocked || state.Port != prof.ProxyPort {
		t.Fatalf("state = %+v, want locked on %d", state, prof.ProxyPort)
	}
}
