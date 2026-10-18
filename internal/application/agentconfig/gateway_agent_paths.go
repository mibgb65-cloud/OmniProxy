package agentconfig

import (
	"os"
	"path/filepath"
	"runtime"
)

func agentHome(home, env string, parts ...string) string {
	if value := os.Getenv(env); value != "" {
		return value
	}
	return filepath.Join(append([]string{home}, parts...)...)
}

func gooseConfigPath(home string) string {
	// Goose 仅接受绝对路径覆盖；其下 config 与 data 分开，不能直接将根目录视为配置目录。
	if root := os.Getenv("GOOSE_PATH_ROOT"); filepath.IsAbs(root) {
		return filepath.Join(root, "config", "config.yaml")
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "Block", "goose", "config", "config.yaml")
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support", "Block", "goose", "config.yaml")
	}
	return filepath.Join(xdgClientConfig(home, "goose"), "config.yaml")
}

func kimiCodeConfigPath(home string) (string, bool) {
	if directory := os.Getenv("KIMI_CODE_HOME"); directory != "" {
		return filepath.Join(directory, "config.toml"), false
	}
	if directory := os.Getenv("KIMI_SHARE_DIR"); directory != "" {
		return filepath.Join(directory, "config.toml"), true
	}
	current := filepath.Join(home, ".kimi-code", "config.toml")
	if _, err := os.Lstat(current); !os.IsNotExist(err) {
		return current, false
	}
	legacy := filepath.Join(home, ".kimi", "config.toml")
	if _, err := os.Lstat(legacy); !os.IsNotExist(err) {
		return legacy, true
	}
	return current, false
}
