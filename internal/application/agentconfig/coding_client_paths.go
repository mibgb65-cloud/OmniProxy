package agentconfig

import (
	"os"
	"path/filepath"
)

func qwenHome(home string) string {
	if value := os.Getenv("QWEN_HOME"); value != "" {
		return value
	}
	return filepath.Join(home, ".qwen")
}

func xdgClientConfig(home, product string) string {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, product)
}

// 配置按客户端实际读取顺序选择最后一个已有文件，避免写入会被旧兼容文件覆盖的位置。
func preferredClientFile(directory string, names []string, fallback string) string {
	for i := len(names) - 1; i >= 0; i-- {
		candidate := filepath.Join(directory, names[i])
		if _, err := os.Lstat(candidate); !os.IsNotExist(err) {
			return candidate
		}
	}
	return filepath.Join(directory, fallback)
}

func kiloConfigPath(home string) string {
	if file := os.Getenv("KILO_CONFIG"); file != "" {
		return file
	}
	directory := os.Getenv("KILO_CONFIG_DIR")
	if directory == "" {
		directory = xdgClientConfig(home, "kilo")
	}
	return preferredClientFile(directory, []string{"config.json", "kilo.json", "kilo.jsonc", "opencode.json", "opencode.jsonc"}, "kilo.jsonc")
}

func mimoCodeConfigPath(home string) string {
	if file := os.Getenv("MIMOCODE_CONFIG"); file != "" {
		return file
	}
	directory := os.Getenv("MIMOCODE_CONFIG_DIR")
	if directory == "" {
		if root := os.Getenv("MIMOCODE_HOME"); root != "" {
			directory = filepath.Join(root, "config")
		} else {
			directory = xdgClientConfig(home, "mimocode")
		}
	}
	return preferredClientFile(directory, []string{"config.json", "mimocode.json", "mimocode.jsonc"}, "mimocode.jsonc")
}
