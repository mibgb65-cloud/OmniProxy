//go:build windows

package codex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"golang.org/x/sys/windows"
	"path/filepath"
	"unicode/utf16"
	"unsafe"
)

type windowsCredential struct {
	Flags, Type             uint32
	Target, Comment         *uint16
	Written                 windows.Filetime
	Size                    uint32
	Blob                    *byte
	Persist, Count          uint32
	Attributes, Alias, User uintptr
}

// UseToken 只定位受控隔离授权；兼容官方旧 keyring 和新 age 加密存储。
func UseToken(home, expectedID string, consume func([]byte, string) error) error {
	if consume == nil {
		return ErrProtocol
	}
	return useStoredAuth(home, func(data []byte) error {
		var auth struct {
			Tokens struct {
				Access    string `json:"access_token"`
				AccountID string `json:"account_id"`
			} `json:"tokens"`
		}
		if json.Unmarshal(data, &auth) != nil || len(auth.Tokens.Access) < 1 || len(auth.Tokens.Access) > 32768 || auth.Tokens.AccountID == "" || expectedID != "" && expectedID != auth.Tokens.AccountID {
			return ErrProtocol
		}
		key := []byte(auth.Tokens.Access)
		auth.Tokens.Access = ""
		defer clear(key)
		for _, ch := range key {
			if ch < 33 || ch > 126 {
				return ErrProtocol
			}
		}
		return consume(key, auth.Tokens.AccountID)
	})
}

func useStoredAuth(home string, consume func([]byte) error) error {
	return useStoredAuthContext(context.Background(), home, consume)
}

func useStoredAuthContext(ctx context.Context, home string, consume func([]byte) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !filepath.IsAbs(home) || consume == nil {
		return ErrProtocol
	}
	path, err := filepath.EvalSymlinks(home)
	if err != nil {
		return ErrProtocol
	}
	canonical, err := canonicalHome(home)
	if err != nil {
		return err
	}
	for _, p := range []string{canonical, path} {
		sum := sha256.Sum256([]byte(p))
		hash := hex.EncodeToString(sum[:])[:16]
		data, err := readWindowsCredential("cli|" + hash + ".Codex Auth")
		if err != nil {
			passphrase, e := readWindowsCredential("secrets|" + hash + ".codex")
			if e != nil {
				continue
			}
			data, err = readEncryptedAuthContext(ctx, path, passphrase)
			clear(passphrase)
			if err != nil {
				return err
			}
		}
		defer clear(data)
		if err := ctx.Err(); err != nil {
			return err
		}
		return consume(data)
	}
	return ErrUnavailable
}

// Rust canonicalize 解析 Windows junction；filepath.EvalSymlinks 在部分挂载点不会。
func canonicalHome(home string) (string, error) {
	name, err := windows.UTF16PtrFromString(home)
	if err != nil {
		return "", ErrProtocol
	}
	handle, err := windows.CreateFile(name, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	if err != nil {
		return "", ErrUnavailable
	}
	defer windows.CloseHandle(handle)
	buffer := make([]uint16, 32768)
	n, err := windows.GetFinalPathNameByHandle(handle, &buffer[0], uint32(len(buffer)), 0)
	if err != nil || n == 0 || n >= uint32(len(buffer)) {
		return "", ErrProtocol
	}
	return windows.UTF16ToString(buffer[:n]), nil
}
func readWindowsCredential(name string) ([]byte, error) {
	api := windows.NewLazySystemDLL("advapi32.dll")
	target, err := windows.UTF16PtrFromString(name)
	if err != nil {
		return nil, ErrProtocol
	}
	var credential *windowsCredential
	ok, _, _ := api.NewProc("CredReadW").Call(uintptr(unsafe.Pointer(target)), 1, 0, uintptr(unsafe.Pointer(&credential)))
	if ok == 0 {
		return nil, ErrUnavailable
	}
	defer api.NewProc("CredFree").Call(uintptr(unsafe.Pointer(credential)))
	if credential.Size == 0 || credential.Size > 64<<10 || credential.Size%2 != 0 || credential.Blob == nil {
		return nil, ErrProtocol
	}
	blob := unsafe.Slice(credential.Blob, credential.Size)
	defer clear(blob)
	units := make([]uint16, len(blob)/2)
	defer clear(units)
	for i := range units {
		units[i] = uint16(blob[i*2]) | uint16(blob[i*2+1])<<8
	}
	return []byte(string(utf16.Decode(units))), nil
}
