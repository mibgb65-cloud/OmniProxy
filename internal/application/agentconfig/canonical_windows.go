//go:build windows

package agentconfig

import (
	"errors"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows"
)

// canonicalConfigPath 通过目标句柄解析目录链接，不枚举 WpSystem 等受保护祖先目录。
func canonicalConfigPath(path string) (string, error) {
	if !filepath.IsAbs(path) || len(path) > 4096 {
		return "", errors.New("invalid configuration path")
	}
	// 普通目录沿用已验证的解析方式；包数据的保护祖先目录无法枚举时才使用句柄解析。
	if real, err := filepath.EvalSymlinks(path); err == nil {
		return real, nil
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	handle, err := windows.CreateFile(name, windows.FILE_READ_ATTRIBUTES, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OPEN_REPARSE_POINT, 0)
	if err != nil {
		return "", err
	}
	defer windows.CloseHandle(handle)
	buffer := make([]uint16, 4096)
	length, err := windows.GetFinalPathNameByHandle(handle, &buffer[0], uint32(len(buffer)), 0)
	if err != nil || length == 0 || length >= uint32(len(buffer)) {
		return "", errors.New("configuration path resolution failed")
	}
	real := windows.UTF16ToString(buffer[:length])
	if strings.HasPrefix(real, `\\?\UNC\`) {
		real = `\\` + strings.TrimPrefix(real, `\\?\UNC\`)
	} else {
		real = strings.TrimPrefix(real, `\\?\`)
	}
	return filepath.Clean(real), nil
}
