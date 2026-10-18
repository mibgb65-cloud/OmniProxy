package wails

import (
	"errors"

	"omniproxy/internal/infrastructure/update"
	"omniproxy/internal/platform/profile"
	"omniproxy/internal/version"
)

// UpdatePreferences 是已解析的渠道与仓库，不触发联网。
type UpdatePreferences struct {
	Channel string `json:"channel"`
	Repo    string `json:"repo"`
}

func updatePreferences(cfg profile.Config, current string) UpdatePreferences {
	channel := cfg.UpdateChannel
	if channel == "" {
		channel = "stable"
		if version.IsPrerelease(current) {
			channel = "beta"
		}
	}
	repo := cfg.UpdateRepo
	if repo == "" {
		repo = update.DefaultRepo
	}
	return UpdatePreferences{Channel: channel, Repo: repo}
}

// GetUpdatePreferences 返回更新选择；Beta 安装默认继续接收 Beta，明确选择始终优先。
func (s *AppService) GetUpdatePreferences() (UpdatePreferences, error) {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	cfg, err := profile.LoadConfig(s.prof.Kind)
	if err != nil {
		return UpdatePreferences{}, err
	}
	return updatePreferences(cfg, version.Version), nil
}

// SetUpdateChannel 保存用户选择，复用设置锁避免覆盖目录／关闭行为配置。
func (s *AppService) SetUpdateChannel(channel string) (UpdatePreferences, error) {
	if channel != "stable" && channel != "beta" {
		return UpdatePreferences{}, errors.New("update: invalid channel")
	}
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	cfg, err := profile.LoadConfig(s.prof.Kind)
	if err != nil {
		return UpdatePreferences{}, err
	}
	cfg.UpdateChannel = channel
	if err := profile.SaveConfig(s.prof.Kind, cfg); err != nil {
		return UpdatePreferences{}, err
	}
	return updatePreferences(cfg, version.Version), nil
}
