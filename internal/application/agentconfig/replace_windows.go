//go:build windows

package agentconfig

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

func replaceConfigFile(temporary, path string, body []byte, revision string) error {
	err := os.Rename(temporary, path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, windows.ERROR_NOT_SAME_DEVICE) {
		return configWriteError(err)
	}
	parent, err := windows.UTF16PtrFromString(filepath.Dir(path))
	if err != nil {
		return failure("invalid_path")
	}
	attributes, err := windows.GetFileAttributes(parent)
	if err != nil || attributes&windows.FILE_ATTRIBUTE_ENCRYPTED == 0 {
		return failure("write_failed")
	}
	// Claude 的加密目录可能拒绝同目录重命名。不能降级成覆盖复制或关闭加密：
	// 独占原文件句柄后再次核对版本，保留加密与 ACL，写入失败时恢复原字节。
	return writeEncryptedConfig(path, body, revision)
}

func configWriteError(err error) error {
	if errors.Is(err, windows.ERROR_SHARING_VIOLATION) || errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return failure("config_locked")
	}
	return failure("write_failed")
}

func writeEncryptedConfig(path string, body []byte, revision string) (resultErr error) {
	if len(body) > maxConfigBytes {
		return failure("unsafe_config")
	}
	info, statErr := os.Lstat(path)
	exists := statErr == nil
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) || exists && !info.Mode().IsRegular() {
		return failure("unsafe_config")
	}
	disposition := uint32(windows.CREATE_NEW)
	if exists {
		disposition = windows.OPEN_EXISTING
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return failure("invalid_path")
	}
	handle, err := windows.CreateFile(name, windows.GENERIC_READ|windows.GENERIC_WRITE|windows.DELETE, 0, nil, disposition, windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if errors.Is(err, windows.ERROR_FILE_EXISTS) || errors.Is(err, windows.ERROR_ALREADY_EXISTS) || errors.Is(err, windows.ERROR_FILE_NOT_FOUND) {
		return failure("config_changed")
	}
	if err != nil {
		return configWriteError(err)
	}
	f := os.NewFile(uintptr(handle), path)
	defer func() {
		if !exists && resultErr != nil {
			// 在独占句柄关闭前标记删除，只清理本次创建的对象，不按路径删除外部替换文件。
			remove := byte(1)
			if windows.SetFileInformationByHandle(handle, windows.FileDispositionInfo, &remove, uint32(unsafe.Sizeof(remove))) != nil {
				resultErr = errors.Join(resultErr, failure("write_failed"))
			}
		}
		if f.Close() != nil {
			resultErr = errors.Join(resultErr, failure("write_failed"))
		}
	}()
	info, err = f.Stat()
	real, resolveErr := canonicalConfigPath(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxConfigBytes || resolveErr != nil || !strings.EqualFold(real, filepath.Clean(path)) {
		return failure("unsafe_config")
	}
	before, err := io.ReadAll(io.LimitReader(f, maxConfigBytes+1))
	if err != nil || len(before) > maxConfigBytes {
		return failure("read_failed")
	}
	defer clear(before)
	if digest(before, exists) != revision {
		return failure("config_changed")
	}
	return rewriteWithRollback(f, body, before)
}

type configWriter interface {
	WriteAt([]byte, int64) (int, error)
	Truncate(int64) error
	Sync() error
}

func rewriteWithRollback(f configWriter, body, before []byte) error {
	if rewriteConfig(f, body) == nil {
		return nil
	}
	if rewriteConfig(f, before) != nil {
		return errors.Join(failure("rollback_failed"), failure("write_failed"))
	}
	return failure("write_failed")
}

func rewriteConfig(f configWriter, body []byte) error {
	if n, err := f.WriteAt(body, 0); err != nil || n != len(body) {
		return failure("write_failed")
	}
	if f.Truncate(int64(len(body))) != nil || f.Sync() != nil {
		return failure("write_failed")
	}
	return nil
}
