//go:build !windows

package legacy

import "errors"

// Unprotect 不将 Windows 凭据降级为跨平台明文。
func Unprotect(string) ([]byte, error) { return nil, errors.New("legacy: unsupported_platform") }
