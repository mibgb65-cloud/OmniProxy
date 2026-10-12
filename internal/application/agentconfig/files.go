package agentconfig

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maxConfigBytes = 512 << 10

func digest(body []byte, exists bool) string {
	prefix := byte(0)
	if exists {
		prefix = 1
	}
	sum := sha256.Sum256(append([]byte{prefix}, body...))
	return hex.EncodeToString(sum[:])
}
func read(path string) ([]byte, bool, error) {
	if !filepath.IsAbs(path) {
		return nil, false, failure("invalid_path")
	}
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return []byte{}, false, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxConfigBytes {
		return nil, false, failure("unsafe_config")
	}
	real, err := canonicalConfigPath(path)
	if err != nil || !strings.EqualFold(real, filepath.Clean(path)) {
		return nil, false, failure("unsafe_config")
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, false, failure("read_failed")
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, maxConfigBytes+1))
	if err != nil || len(b) > maxConfigBytes {
		return nil, false, failure("read_failed")
	}
	return b, true, nil
}

// writeAtomic 在提交前检查原文件版本；Windows 加密目录使用独占写入兼容不能重命名的情况。
func writeAtomic(path string, body []byte, revision string) error {
	if len(body) > maxConfigBytes {
		return failure("unsafe_config")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return failure("write_failed")
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".omniproxy-*.tmp")
	if err != nil {
		return failure("write_failed")
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err != nil {
		f.Close()
		return failure("write_failed")
	}
	if _, err = f.Write(body); err != nil {
		f.Close()
		return failure("write_failed")
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return failure("write_failed")
	}
	if err = f.Close(); err != nil {
		return failure("write_failed")
	}
	current, exists, err := read(path)
	if err != nil {
		return err
	}
	defer clear(current)
	if digest(current, exists) != revision {
		return failure("config_changed")
	}
	return replaceConfigFile(name, path, body, revision)
}
