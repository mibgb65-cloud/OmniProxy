//go:build windows

package agentconfig

import (
	"os"
	"path/filepath"
	"strings"
)

func desktopThirdPartyRoot(cache, fallback string) (string, error) {
	// MSIX 文件系统虚拟化会优先读取已有 Local 副本，只有真实配置库存在时才切换，避免把空包缓存当作配置。
	for _, family := range []string{"AnthropicPBC.Claude_fnn82j28hfe8t", "Claude_pzs8sxrjxfjjc"} {
		root := filepath.Join(cache, "Packages", family, "LocalCache", "Local", "Claude-3p")
		metadata := filepath.Join(root, "configLibrary", "_meta.json")
		info, err := os.Lstat(metadata)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil || !info.Mode().IsRegular() || info.Size() > maxConfigBytes {
			return "", failure("invalid_path")
		}
		// 商店数据可被系统重定向到其他盘的 WpSystem；使用同一实体的规范路径，不放宽配置读写的链接保护。
		real, err := canonicalConfigPath(metadata)
		suffix := filepath.Join("Packages", family, "LocalCache", "Local", "Claude-3p", "configLibrary", "_meta.json")
		if err != nil || !strings.HasSuffix(strings.ToLower(real), strings.ToLower(string(filepath.Separator)+suffix)) {
			return "", failure("invalid_path")
		}
		return filepath.Dir(filepath.Dir(real)), nil
	}
	return fallback, nil
}
