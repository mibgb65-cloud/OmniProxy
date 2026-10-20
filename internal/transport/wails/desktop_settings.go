package wails

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"

	"omniproxy/internal/platform/profile"
)

type autostartController interface {
	IsEnabled() (bool, error)
	EnableWithOptions(application.AutostartOptions) error
	Disable() error
}

// DesktopSettings 包含系统开机启动状态、软件启动时的网关偏好与窗口关闭行为。
type DesktopSettings struct {
	AutoStart          bool   `json:"autoStart"`
	AutoStartAvailable bool   `json:"autoStartAvailable"`
	GatewayAutoStart   bool   `json:"gatewayAutoStart"`
	CloseBehavior      string `json:"closeBehavior"`
}

// DesktopSettings 从系统读取开机启动状态，从 profile 配置读取网关启动和窗口关闭偏好。
func (s *AppService) DesktopSettings() (DesktopSettings, error) {
	cfg, err := profile.LoadConfig(s.prof.Kind)
	if err != nil {
		return DesktopSettings{}, err
	}
	closeBehavior, err := profile.ParseCloseBehavior(cfg.CloseBehavior)
	if err != nil {
		return DesktopSettings{}, err
	}
	result := DesktopSettings{CloseBehavior: closeBehavior, GatewayAutoStart: cfg.GatewayAutoStart}
	// Dev and production builds use the same output path. Registering a dev
	// binary would later launch whichever profile last replaced that path.
	if !s.prof.IsProduction() {
		return result, nil
	}
	if s.autostart == nil {
		return DesktopSettings{}, fmt.Errorf("desktop settings: autostart is unavailable")
	}
	autoStart, err := s.autostart.IsEnabled()
	if err != nil {
		return DesktopSettings{}, fmt.Errorf("desktop settings: read autostart status: %w", err)
	}
	result.AutoStart = autoStart
	result.AutoStartAvailable = true
	return result, nil
}

// SetAutoStart changes the current user's OS login registration.
func (s *AppService) SetAutoStart(enabled bool) (bool, error) {
	if !s.prof.IsProduction() {
		return false, fmt.Errorf("desktop settings: autostart is only available in the production profile")
	}
	if s.autostart == nil {
		return false, fmt.Errorf("desktop settings: autostart is unavailable")
	}
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	if enabled {
		if err := s.autostart.EnableWithOptions(application.AutostartOptions{Identifier: s.prof.AppID}); err != nil {
			return false, fmt.Errorf("desktop settings: enable autostart: %w", err)
		}
	} else if err := s.autostart.Disable(); err != nil {
		return false, fmt.Errorf("desktop settings: disable autostart: %w", err)
	}
	return enabled, nil
}

// SetCloseBehavior persists the user's window-close preference.
func (s *AppService) SetCloseBehavior(value string) (string, error) {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	if err := profile.SetCloseBehavior(s.prof.Kind, value); err != nil {
		return "", err
	}
	return value, nil
}

// SetGatewayAutoStart 保存下次启动应用时的网关偏好，不立即启停网关。
func (s *AppService) SetGatewayAutoStart(enabled bool) (bool, error) {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	if err := profile.SetGatewayAutoStart(s.prof.Kind, enabled); err != nil {
		return false, err
	}
	return enabled, nil
}

func (s *AppService) shouldCloseToTray() bool {
	cfg, err := profile.LoadConfig(s.prof.Kind)
	if err != nil {
		s.log.Error("failed to read close behavior; closing application", "error", err)
		return false
	}
	mode, err := profile.ParseCloseBehavior(cfg.CloseBehavior)
	if err != nil {
		s.log.Error("invalid close behavior; closing application", "error", err)
		return false
	}
	return mode == profile.CloseToTray
}
