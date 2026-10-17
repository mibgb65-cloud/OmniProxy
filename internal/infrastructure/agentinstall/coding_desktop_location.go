package agentinstall

import (
	"os"
	"path/filepath"
	"strings"
)

func mimoDesktopName(id string) string {
	switch id {
	case "mimo-desktop-cn":
		return "Xiaomi MiMo"
	case "mimo-desktop-global":
		return "Xiaomi MiMo AI"
	}
	return ""
}

func codingDesktopLocation(id, display, publisher, version, location, icon string) (string, bool) {
	name := mimoDesktopName(id)
	if name == "" {
		return location, codingDesktopMatches(id, display, publisher)
	}
	if validVersion(version) {
		display = strings.TrimSuffix(display, " "+version)
	}
	if !strings.EqualFold(display, name) {
		return "", false
	}
	if codingDesktopMatches(id, display, publisher) && filepath.IsAbs(location) {
		return location, true
	}
	// MiMo 的已验证 NSIS 注册项不填发布者或安装目录，只提供 DisplayIcon。
	// 只接受对应地区的精确产品名和现存固定主程序，绝不执行卸载命令或图标参数。
	if publisher != "" || !validVersion(version) || len(icon) > 4096 {
		return "", false
	}
	path := strings.Trim(strings.TrimSuffix(icon, ",0"), "\"")
	if !filepath.IsAbs(path) || !strings.EqualFold(filepath.Base(path), name+".exe") {
		return "", false
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	return filepath.Dir(path), true
}
