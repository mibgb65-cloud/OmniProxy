//go:build windows

package legacy

import (
	"encoding/base64"
	"errors"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Unprotect 兼容 v1 DPAPI entropy，仅在当前 Windows 用户上下文中解密。
func Unprotect(value string) ([]byte, error) {
	if len(value) == 0 || len(value) > 32768 {
		return nil, errors.New("legacy: invalid_credential")
	}
	const prefix = "omniproxy-secret:v1:"
	if !strings.HasPrefix(value, prefix) {
		return []byte(value), nil
	}
	data, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, prefix))
	if err != nil || len(data) == 0 {
		return nil, errors.New("legacy: invalid_credential")
	}
	defer clear(data)
	entropy := []byte("OmniProxy token store v1")
	in := windows.DataBlob{Size: uint32(len(data)), Data: &data[0]}
	extra := windows.DataBlob{Size: uint32(len(entropy)), Data: &entropy[0]}
	var out windows.DataBlob
	if windows.CryptUnprotectData(&in, nil, &extra, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out) != nil {
		return nil, errors.New("legacy: credential_unavailable")
	}
	if out.Data == nil {
		return nil, errors.New("legacy: credential_unavailable")
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.Data))))
	if out.Size == 0 || out.Size > 32768 {
		return nil, errors.New("legacy: invalid_credential")
	}
	native := unsafe.Slice(out.Data, int(out.Size))
	result := append([]byte(nil), native...)
	clear(native)
	return result, nil
}
