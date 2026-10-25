package wails

import (
	"testing"

	"omniproxy/internal/platform/profile"
)

// 自动激活依赖进程存活：开启后关闭窗口应留在托盘，而不是退出应用。
func TestEnablingAutoActivationKeepsApplicationRunningInTray(t *testing.T) {
	service, prof := newTestService(t)
	if err := profile.SetCloseBehavior(prof.Kind, profile.CloseExit); err != nil {
		t.Fatal(err)
	}
	service.ensureTrayCloseBehavior()
	cfg, err := profile.LoadConfig(prof.Kind)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CloseBehavior != profile.CloseToTray {
		t.Fatalf("close behavior = %q, want tray", cfg.CloseBehavior)
	}
}

// 用户之后显式改回退出时不重复改写，桌面设置仍是唯一来源。
func TestTrayCloseBehaviorIsNotReappliedWithoutEnablement(t *testing.T) {
	service, prof := newTestService(t)
	if err := profile.SetCloseBehavior(prof.Kind, profile.CloseToTray); err != nil {
		t.Fatal(err)
	}
	service.ensureTrayCloseBehavior()
	if err := profile.SetCloseBehavior(prof.Kind, profile.CloseExit); err != nil {
		t.Fatal(err)
	}
	service.ensureBackgroundActivation()
	cfg, err := profile.LoadConfig(prof.Kind)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CloseBehavior != profile.CloseExit {
		t.Fatalf("close behavior = %q, want exit", cfg.CloseBehavior)
	}
}
