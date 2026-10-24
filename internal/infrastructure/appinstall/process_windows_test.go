//go:build windows

package appinstall

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTargetValidationAndMalformedModes(t *testing.T) {
	for _, target := range []string{"OmniProxy.exe", `C:\OmniProxy.exe`, `C:\Windows\notepad.exe`, ""} {
		if validTarget(target) {
			t.Errorf("unsafe target accepted: %s", target)
		}
	}
	for _, args := range [][]string{{"--omniproxy-update-helper"}, {"--omniproxy-installer-stop"}, {"--omniproxy-installer-cleanup"}, {"--omniproxy-installer-launch"}, {"--omniproxy-installer-launch", `C:\Windows\notepad.exe`}, {"--omniproxy-installer-data-dir-read", "extra"}, {"--omniproxy-installer-data-dir-write"}} {
		if handled, code := HandleMode(args); !handled || code == 0 {
			t.Fatal("malformed mode escaped validation")
		}
	}
}

func TestQuitListenerUsesExactExecutableIdentity(t *testing.T) {
	called := make(chan struct{})
	stop, err := ListenForQuit(func() { close(called) })
	if err != nil {
		t.Fatal(err)
	}
	defer stop()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	signalQuit(filepath.Join(t.TempDir(), "OmniProxy.exe"))
	select {
	case <-called:
		t.Fatal("another executable identity signalled this app")
	default:
	}
	signalQuit(path)
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("quit event not received")
	}
}
