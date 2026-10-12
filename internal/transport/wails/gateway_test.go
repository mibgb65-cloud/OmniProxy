package wails

import (
	"net"
	"path/filepath"
	"testing"

	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/platform/profile"
)

func TestGatewayBindingsLifecycle(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	prof, err := profile.New(profile.Test)
	if err != nil {
		t.Fatal(err)
	}
	prof.DataDir, prof.ProxyPort = t.TempDir(), port
	db, err := database.Open(database.Config{Path: filepath.Join(prof.DataDir, "gateway.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := NewAppService(prof, nil, db, nil, nil)
	info, err := s.GatewayInfo()
	if err != nil || info.Runtime.State != "stopped" {
		t.Fatalf("initial status: %+v %v", info, err)
	}
	info, err = s.GatewayStart()
	if err != nil || info.Runtime.State != "running" {
		t.Fatalf("start: %+v %v", info, err)
	}
	info, err = s.GatewayStop()
	if err != nil || info.Runtime.State != "stopped" {
		t.Fatalf("stop: %+v %v", info, err)
	}
}

func TestGatewayBindingsRejectMissingConfiguration(t *testing.T) {
	s := NewAppService(nil, nil, nil, nil, nil)
	if _, err := s.GatewayInfo(); err == nil {
		t.Fatal("missing configuration accepted")
	}
	if _, err := s.GatewayStart(); err == nil {
		t.Fatal("missing configuration started")
	}
}
