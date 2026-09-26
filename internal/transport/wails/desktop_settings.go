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

// DesktopSettings describes the current OS autostart status and close behavior.
type DesktopSettings struct {
	AutoStart          bool   `json:"autoStart"`
	AutoStartAvailable bool   `json:"autoStartAvailable"`
	CloseBehavior      string `json:"closeBehavior"`
}

// DesktopSettings reads autostart from the OS and close behavior from the
// profile config, so external OS changes are reflected on refresh.
func (s *AppService) DesktopSettings() (DesktopSettings, error) {
	cfg, err := profile.LoadConfig(s.prof.Kind)
	if err != nil {
		return DesktopSettings{}, err
	}
	closeBehavior, err := profile.ParseCloseBehavior(cfg.CloseBehavior)
	if err != nil {
		return DesktopSettings{}, err
	}
	result := DesktopSettings{CloseBehavior: closeBehavior}
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
