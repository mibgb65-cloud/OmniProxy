//go:build windows

package codex

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// DeleteCredentials 只删除该隔离配置的本机 keyring 条目，不撤销远端授权或调用 account/logout。
// 同时处理旧版 token bundle 和新版 age passphrase；加密文件由调用方删除隔离目录时清理。
func DeleteCredentials(home string) error {
	targets, err := credentialTargets(home)
	if err != nil {
		return err
	}
	return deleteCredentialTargets(targets, deleteWindowsCredential)
}

func credentialTargets(home string) ([]string, error) {
	if !filepath.IsAbs(home) || strings.ContainsRune(home, 0) {
		return nil, ErrProtocol
	}
	path, err := filepath.EvalSymlinks(home)
	if err != nil && !os.IsNotExist(err) {
		return nil, ErrUnavailable
	}
	canonical, canonicalErr := canonicalHome(home)
	if canonicalErr != nil {
		if !os.IsNotExist(err) {
			return nil, canonicalErr
		}
		// 目录已丢失时仍清理常规路径的两个官方命名形式；缺失条目是幂等成功。
		path = filepath.Clean(home)
		canonical = path
		if !strings.HasPrefix(path, `\\?\`) {
			if strings.HasPrefix(path, `\\`) {
				canonical = `\\?\UNC\` + strings.TrimPrefix(path, `\\`)
			} else {
				canonical = `\\?\` + path
			}
		}
	}
	var targets []string
	seen := map[string]bool{}
	for _, value := range []string{canonical, path} {
		sum := sha256.Sum256([]byte(value))
		hash := hex.EncodeToString(sum[:])[:16]
		for _, target := range []string{"cli|" + hash + ".Codex Auth", "secrets|" + hash + ".codex"} {
			if !seen[target] {
				targets = append(targets, target)
				seen[target] = true
			}
		}
	}
	return targets, nil
}

func deleteCredentialTargets(targets []string, remove func(string) error) error {
	for _, target := range targets {
		if err := remove(target); err != nil && !errors.Is(err, windows.ERROR_NOT_FOUND) {
			return ErrUnavailable
		}
	}
	return nil
}

func deleteWindowsCredential(target string) error {
	name, err := windows.UTF16PtrFromString(target)
	if err != nil {
		return ErrProtocol
	}
	ok, _, callErr := windows.NewLazySystemDLL("advapi32.dll").NewProc("CredDeleteW").Call(uintptr(unsafe.Pointer(name)), 1, 0)
	if ok == 0 {
		return callErr
	}
	return nil
}
