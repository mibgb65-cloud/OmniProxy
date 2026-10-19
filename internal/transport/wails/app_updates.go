package wails

import (
	"context"
	"errors"
	"path/filepath"
	"runtime"
	"time"

	"omniproxy/internal/application/updater"
	"omniproxy/internal/infrastructure/appinstall"
	"omniproxy/internal/infrastructure/update"
	"omniproxy/internal/platform/profile"
	"omniproxy/internal/version"
)

func (s *AppService) applicationUpdater() (*updater.Service, error) {
	prefs, err := s.GetUpdatePreferences()
	if err != nil {
		return nil, err
	}
	if prefs.Channel != "stable" && prefs.Channel != "beta" {
		return nil, errors.New("update: invalid_channel")
	}
	s.updatesMu.Lock()
	if s.updates == nil {
		opts := updater.Options{Directory: filepath.Join(s.prof.DataDir, "updates"), Version: version.Version, OS: runtime.GOOS, Arch: runtime.GOARCH, CanInstall: appinstall.Installed()}
		opts.Checker = func(repo string, beta bool) *update.Checker {
			checker := newUpdateChecker(repo)
			checker.IncludePrereleases = beta
			return checker
		}
		opts.Launch = func(path, digest, receipt, targetVersion string, size int64) error {
			return appinstall.Launch(appinstall.Request{Package: path, Digest: digest, Receipt: receipt, Version: targetVersion, Size: size})
		}
		configureUpdateQA(&opts, s.prof)
		s.updates = updater.New(opts)
	}
	service := s.updates
	s.updatesMu.Unlock()
	if err := service.Configure(prefs.Repo, prefs.Channel == "beta"); err != nil {
		return nil, err
	}
	return service, nil
}

// ApplicationUpdate 返回本地快照，打开侧栏或浮窗不会因此联网。
func (s *AppService) ApplicationUpdate() (updater.Snapshot, error) {
	service, err := s.applicationUpdater()
	if err != nil {
		return updater.Snapshot{}, err
	}
	return service.Snapshot(), nil
}

// StartApplicationUpdate 前端只确认版本，不提供下载地址或执行路径。
func (s *AppService) StartApplicationUpdate(expected string) (updater.Snapshot, error) {
	service, err := s.applicationUpdater()
	if err != nil {
		return updater.Snapshot{}, err
	}
	return service.Start(expected)
}

// CancelApplicationUpdate 取消下载并保留可重试的状态。
func (s *AppService) CancelApplicationUpdate() (updater.Snapshot, error) {
	service, err := s.applicationUpdater()
	if err != nil {
		return updater.Snapshot{}, err
	}
	return service.Cancel()
}

// InstallApplicationUpdate 安装辅助程序就绪后才正常退出应用，取消 UAC 不退出。
func (s *AppService) InstallApplicationUpdate(expected string) (updater.Snapshot, error) {
	if s.quitForUpdate == nil {
		return updater.Snapshot{}, errors.New("update: install_unavailable")
	}
	service, err := s.applicationUpdater()
	if err != nil {
		return updater.Snapshot{}, err
	}
	state, err := service.Install(expected)
	if err == nil {
		go func() { time.Sleep(250 * time.Millisecond); s.quitForUpdate() }()
	}
	return state, err
}

func (s *AppService) startUpdateChecks() {
	if !s.prof.IsProduction() {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.stopUpdateChecks = cancel
	go func() {
		timer := time.NewTimer(30 * time.Second)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				prefs, err := s.GetUpdatePreferences()
				if err == nil && prefs.AutoCheck {
					service, serviceErr := s.applicationUpdater()
					if serviceErr == nil && !service.Busy() {
						s.checkApplicationUpdate(ctx, service)
					}
				}
				timer.Reset(6 * time.Hour)
			}
		}
	}()
}

// 检查后重新读取用户选择，关闭开关不会被检查开始时的旧配置覆盖。
func (s *AppService) checkApplicationUpdate(ctx context.Context, service *updater.Service) (updater.Snapshot, error) {
	state, err := service.Check(ctx)
	if err != nil || !s.prof.IsProduction() {
		return state, err
	}
	// 与偏好写入共用锁，关闭开关或切换渠道不能夹在复核与启动之间。
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	cfg, err := profile.LoadConfig(s.prof.Kind)
	if err != nil {
		s.log.Warn("automatic update download settings unavailable", "module", "updater", "code", "settings_failed")
		return state, nil
	}
	prefs := updatePreferences(cfg, version.Version)
	if !prefs.AutoDownload {
		return state, nil
	}
	if (prefs.Channel != "stable" && prefs.Channel != "beta") || service.Configure(prefs.Repo, prefs.Channel == "beta") != nil {
		s.log.Warn("automatic update download configuration changed", "module", "updater", "code", "configuration_changed")
		return state, nil
	}
	state, err = service.StartAutomatic()
	if err != nil {
		// 下载错误留在共享快照中展示，不把成功的版本检查误报为检查失败。
		s.log.Warn("automatic update download not started", "module", "updater", "code", "download_start_failed")
	}
	return state, nil
}
