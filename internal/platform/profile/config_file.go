package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func loadConfigFile(path string) (Config, error) {
	var cfg Config
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("profile: read config: %w", err)
	}
	defer file.Close()
	// 配置只有目录与偏好；损坏或恶意文件不能令安装器无界分配内存。
	const maxConfigBytes = 1 << 20
	data, err := io.ReadAll(io.LimitReader(file, maxConfigBytes+1))
	if err != nil {
		return cfg, fmt.Errorf("profile: read config: %w", err)
	}
	if len(data) > maxConfigBytes {
		return cfg, errors.New("profile: config exceeds size limit")
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("profile: parse %s: %w", path, err)
	}
	return cfg, nil
}

func saveConfigFile(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("profile: create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("profile: encode config: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".app-config-*.tmp")
	if err != nil {
		return fmt.Errorf("profile: create config: %w", err)
	}
	tmp := file.Name()
	defer os.Remove(tmp)
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return fmt.Errorf("profile: write config: %w", err)
	}
	// 通过同一个父目录句柄原子改名；EFS 目录中的 Win32 路径改名可能误报跨卷，
	// 不能退回 COPY_ALLOWED 或直接覆盖配置，否则会丢失原子性。
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("profile: open config directory: %w", err)
	}
	defer root.Close()
	if err := root.Rename(filepath.Base(tmp), filepath.Base(path)); err != nil {
		return fmt.Errorf("profile: replace config: %w", err)
	}
	return nil
}
