//go:build windows

package codex

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"golang.org/x/sys/windows"
)

func TestCredentialDeletionTargetsOnlyIsolatedHome(t *testing.T) {
	home := t.TempDir()
	for _, path := range []string{home, filepath.Join(home, "missing")} {
		targets, err := credentialTargets(path)
		if err != nil || len(targets) < 2 || len(targets) > 4 {
			t.Fatal("isolated credential targets unavailable")
		}
		sum := sha256.Sum256([]byte(path))
		hash := hex.EncodeToString(sum[:])[:16]
		for _, target := range []string{"cli|" + hash + ".Codex Auth", "secrets|" + hash + ".codex"} {
			if !slices.Contains(targets, target) {
				t.Fatal("missing isolated legacy credential target")
			}
		}
		var deleted []string
		if err := deleteCredentialTargets(targets, func(target string) error {
			deleted = append(deleted, target)
			return windows.ERROR_NOT_FOUND
		}); err != nil || !slices.Equal(deleted, targets) {
			t.Fatal("missing credential deletion was not idempotent")
		}
	}
	for _, path := range []string{"", "../outside", "relative", home + "\x00"} {
		if _, err := credentialTargets(path); err == nil {
			t.Fatal("invalid credential location accepted")
		}
	}
}

func TestCredentialDeletionReturnsSafeFailure(t *testing.T) {
	count := 0
	err := deleteCredentialTargets([]string{"isolated-one", "isolated-two"}, func(string) error {
		count++
		return errors.New("private-system-detail")
	})
	if !errors.Is(err, ErrUnavailable) || count != 1 {
		t.Fatal("credential failure was exposed or ignored")
	}
}

func TestDeleteCredentialsWithMissingWindowsEntries(t *testing.T) {
	// 随机临时目录从未用于登录，仅调用真实 Win32 删除 API 验证缺失条目的幂等语义。
	if err := DeleteCredentials(t.TempDir()); err != nil {
		t.Fatal("missing Windows credentials did not delete idempotently")
	}
}
