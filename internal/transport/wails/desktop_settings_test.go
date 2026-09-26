package wails

import (
	"errors"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"

	"omniproxy/internal/platform/profile"
)

type fakeAutostart struct {
	enabled    bool
	identifier string
	statusErr  error
	enableErr  error
	disableErr error
}

func newProductionDesktopService(t *testing.T) (*AppService, *profile.Profile) {
	t.Helper()
	svc, prof := newTestService(t)
	prof.Kind = profile.Production
	prof.AppID = "com.omniproxy.app"
	return svc, prof
}

func (f *fakeAutostart) IsEnabled() (bool, error) { return f.enabled, f.statusErr }

func (f *fakeAutostart) EnableWithOptions(opts application.AutostartOptions) error {
	if f.enableErr != nil {
		return f.enableErr
	}
	f.enabled = true
	f.identifier = opts.Identifier
	return nil
}

func (f *fakeAutostart) Disable() error {
	if f.disableErr != nil {
		return f.disableErr
	}
	f.enabled = false
	return nil
}

func TestDesktopSettingsUsesOSStatusAndDefaultsToExit(t *testing.T) {
	svc, _ := newProductionDesktopService(t)
	startup := &fakeAutostart{enabled: true}
	svc.autostart = startup
	got, err := svc.DesktopSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !got.AutoStart || !got.AutoStartAvailable || got.CloseBehavior != profile.CloseExit {
		t.Fatalf("desktop settings = %+v", got)
	}
	startup.statusErr = errors.New("status unavailable")
	if _, err := svc.DesktopSettings(); err == nil {
		t.Fatal("OS status failure should be returned")
	}
}

func TestDesktopSettingsDoesNotRegisterDevelopmentBuild(t *testing.T) {
	svc, _ := newTestService(t)
	got, err := svc.DesktopSettings()
	if err != nil {
		t.Fatal(err)
	}
	if got.AutoStart || got.AutoStartAvailable || got.CloseBehavior != profile.CloseExit {
		t.Fatalf("development settings = %+v", got)
	}
	if _, err := svc.SetAutoStart(true); err == nil {
		t.Fatal("development build must not write a Windows startup entry")
	}
}

func TestSetAutoStartUsesProfileIdentifierAndKeepsFailureVisible(t *testing.T) {
	svc, prof := newProductionDesktopService(t)
	startup := &fakeAutostart{}
	svc.autostart = startup
	if enabled, err := svc.SetAutoStart(true); err != nil || !enabled {
		t.Fatalf("enable = %t, %v", enabled, err)
	}
	if startup.identifier != prof.AppID {
		t.Fatalf("identifier = %q, want %q", startup.identifier, prof.AppID)
	}
	if enabled, err := svc.SetAutoStart(false); err != nil || enabled || startup.enabled {
		t.Fatalf("disable = %t, %v; startup = %+v", enabled, err, startup)
	}
	startup.enableErr = errors.New("registry denied")
	if _, err := svc.SetAutoStart(true); err == nil || startup.enabled {
		t.Fatalf("failed enable must not report success: %v", err)
	}
	startup.enabled = true
	startup.disableErr = errors.New("registry denied")
	if _, err := svc.SetAutoStart(false); err == nil || !startup.enabled {
		t.Fatalf("failed disable must keep the original state: %v", err)
	}
}

func TestCloseBehaviorPersistsAndChangesCloseDecision(t *testing.T) {
	svc, _ := newTestService(t)
	if svc.shouldCloseToTray() {
		t.Fatal("default close behavior should exit")
	}
	if got, err := svc.SetCloseBehavior(profile.CloseToTray); err != nil || got != profile.CloseToTray {
		t.Fatalf("set tray = %q, %v", got, err)
	}
	if !svc.shouldCloseToTray() {
		t.Fatal("close behavior did not take effect")
	}
	if _, err := svc.SetCloseBehavior("invalid"); err == nil || !svc.shouldCloseToTray() {
		t.Fatalf("invalid behavior should fail without changing the saved preference: %v", err)
	}
	if got, err := svc.SetCloseBehavior(profile.CloseExit); err != nil || got != profile.CloseExit || svc.shouldCloseToTray() {
		t.Fatalf("set exit = %q, %v", got, err)
	}
}
