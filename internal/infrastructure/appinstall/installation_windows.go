//go:build windows

package appinstall

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// 隔离安装验收只通过链接参数改编译副本，不接受运行时环境改写正式身份。
var uninstallKeyName = "OmniProxyOmniProxy"

func validTarget(path string) bool {
	return filepath.IsAbs(path) && len(path) < 4096 && strings.EqualFold(filepath.Base(path), "OmniProxy.exe") && filepath.Dir(path) != filepath.VolumeName(path)+string(filepath.Separator)
}

func installation(target string) (string, error) {
	if !validTarget(target) {
		return "", errors.New("update: install_unavailable")
	}
	for _, hive := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		key, err := registry.OpenKey(hive, `Software\Microsoft\Windows\CurrentVersion\Uninstall\`+uninstallKeyName, registry.QUERY_VALUE|registry.WOW64_64KEY)
		if err != nil {
			continue
		}
		dir, _, _ := key.GetStringValue("InstallLocation")
		if dir == "" {
			icon, _, _ := key.GetStringValue("DisplayIcon")
			dir = filepath.Dir(strings.Trim(strings.TrimSuffix(icon, ",0"), `"`))
		}
		key.Close()
		if strings.EqualFold(filepath.Clean(dir), filepath.Dir(filepath.Clean(target))) {
			info, err := os.Lstat(filepath.Join(dir, "uninstall.exe"))
			if err != nil || !info.Mode().IsRegular() {
				return "", errors.New("update: install_unavailable")
			}
			if hive == registry.CURRENT_USER {
				return "user", nil
			}
			return "machine", nil
		}
	}
	return "", errors.New("update: install_unavailable")
}

// Installed 便携或开发二进制不提供静默覆盖更新。
func Installed() bool {
	target, err := os.Executable()
	if err != nil {
		return false
	}
	_, err = installation(target)
	return err == nil
}
