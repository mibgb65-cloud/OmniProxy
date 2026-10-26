package profile

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// ErrDataDirInUse is returned when the installer tries to switch away from a
// data directory that already holds application data. Callers surface it as
// "relocate from inside the app instead" rather than as a generic failure.
var ErrDataDirInUse = errors.New("profile: data directory still contains application data")

// InstallerLocation 显式描述安装来源与使用者目录，不能从提权进程环境推断用户。
type InstallerLocation struct {
	ConfigDir      string
	DefaultDataDir string
	InstallDir     string
}

// DataDir 按 app.json、旧安装标记、使用者默认目录的顺序解析真实数据位置。
func (l InstallerLocation) DataDir() (string, error) {
	if err := l.validate(); err != nil {
		return "", err
	}
	cfg, err := loadConfigFile(filepath.Join(l.ConfigDir, configFileName))
	if err != nil {
		return "", err
	}
	if dir := cfg.DataDir; dir != "" {
		if err := validInstallerPath(dir); err != nil {
			return "", err
		}
		if cfg.PreviousDataDir != "" {
			if err := validInstallerPath(cfg.PreviousDataDir); err != nil {
				return "", err
			}
			hasData, err := installerContainsDatabase(dir)
			if err != nil {
				return "", err
			}
			previousHasData, err := installerContainsDatabase(cfg.PreviousDataDir)
			if err != nil {
				return "", err
			}
			if !hasData && previousHasData {
				return cfg.PreviousDataDir, nil
			}
		}
		return dir, nil
	}
	// 安装器助手位于临时目录，必须读取显式传入的原安装目录，而非自身所在目录。
	dir, err := installerLegacyDataDir(l.InstallDir)
	if err != nil {
		return "", err
	}
	if dir != "" {
		return dir, nil
	}
	return l.DefaultDataDir, nil
}

// SetDataDir 只为尚无数据的安装选择目录；已有数据库必须在应用内搬迁。
func (l InstallerLocation) SetDataDir(dir string) error {
	current, err := l.DataDir()
	if err != nil {
		return err
	}
	target := strings.TrimSpace(dir)
	if err := validInstallerPath(target); err != nil {
		return err
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("profile: resolve %q: %w", target, err)
	}
	path := filepath.Join(l.ConfigDir, configFileName)
	cfg, err := loadConfigFile(path)
	if err != nil {
		return err
	}
	if strings.EqualFold(abs, filepath.Clean(current)) {
		if cfg.DataDir != "" || strings.EqualFold(abs, filepath.Clean(l.DefaultDataDir)) {
			return nil
		}
		// 固化历史标记，保证用户改动程序安装位置后仍指向原数据，不移动任何文件。
		cfg.DataDir = current
		return saveConfigFile(path, cfg)
	}
	inUse, err := installerContainsDatabase(current)
	if err != nil {
		return err
	}
	if inUse {
		return fmt.Errorf("%w: %s", ErrDataDirInUse, current)
	}
	if isNested(abs, current) || isNested(current, abs) {
		return fmt.Errorf("profile: %q and %q must not contain each other", current, abs)
	}
	if err := ensureWritableDir(abs); err != nil {
		return err
	}
	cfg.DataDir = abs
	cfg.PreviousDataDir = ""
	return saveConfigFile(path, cfg)
}

func (l InstallerLocation) validate() error {
	for _, dir := range []string{l.ConfigDir, l.DefaultDataDir, l.InstallDir} {
		if err := validInstallerPath(dir); err != nil {
			return err
		}
	}
	return nil
}
