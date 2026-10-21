//go:build windows

package codex

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"omniproxy/internal/infrastructure/credential"
)

func TestEncryptedAuthReadsSourceDespiteSameSizeAndTimestamp(t *testing.T) {
	home := t.TempDir()
	t.Cleanup(func() { officialAuthCache.clearHome(home) })
	path := filepath.Join(home, "secrets", "codex_auth.age")
	pass := []byte("test-only-passphrase")
	modified := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	var size int64
	for _, auth := range []string{"synthetic-one", "synthetic-two"} {
		writeEncryptedAuthFixture(t, home, auth, 10)
		if err := os.Chtimes(path, modified, modified); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil || !info.ModTime().Equal(modified) || size != 0 && size != info.Size() {
			t.Fatal("fixture did not preserve ciphertext size and timestamp")
		}
		size = info.Size()
		got, err := readEncryptedAuth(home, pass)
		if err != nil || string(got) != auth {
			t.Fatal("source content change was hidden by cached file metadata")
		}
		clear(got)
		officialAuthCache.mu.Lock()
		secret := officialAuthCache.entries[path].secret
		officialAuthCache.mu.Unlock()
		got, err = readEncryptedAuth(home, pass)
		clear(got)
		if err != nil {
			t.Fatal(err)
		}
		officialAuthCache.mu.Lock()
		current := officialAuthCache.entries[path].secret
		officialAuthCache.mu.Unlock()
		if secret != current {
			t.Fatal("unchanged official ciphertext was unnecessarily decrypted")
		}
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err := readEncryptedAuth(home, pass); err == nil {
		t.Fatal("deleted source remained accessible through cache")
	}
	officialAuthCache.mu.Lock()
	_, retained := officialAuthCache.entries[path]
	officialAuthCache.mu.Unlock()
	if retained {
		t.Fatal("deleted source kept a protected cache entry")
	}
}

func TestDeleteCredentialsClearsIsolatedProtectedAuthCache(t *testing.T) {
	home := t.TempDir()
	t.Cleanup(func() { officialAuthCache.clearHome(home) })
	writeEncryptedAuthFixture(t, home, "synthetic-logout", 10)
	data, err := readEncryptedAuth(home, []byte("test-only-passphrase"))
	clear(data)
	if err != nil {
		t.Fatal(err)
	}
	officialAuthCache.mu.Lock()
	secret := officialAuthCache.entries[filepath.Join(home, "secrets", "codex_auth.age")].secret
	officialAuthCache.mu.Unlock()
	// 随机测试目录没有真实 keyring 条目，仅验证幂等删除与内存失效。
	if err := DeleteCredentials(home); err != nil {
		t.Fatal(err)
	}
	if _, err := secret.Open(); !errors.Is(err, credential.ErrNotFound) {
		t.Fatal("logout retained its protected authorization snapshot")
	}
}

func TestEncryptedAuthRejectsInvalidPassphraseAndSourceBounds(t *testing.T) {
	home := t.TempDir()
	for _, pass := range [][]byte{nil, make([]byte, (64<<10)+1)} {
		if _, err := readEncryptedAuth(home, pass); !errors.Is(err, ErrProtocol) {
			t.Fatal("invalid system passphrase size accepted")
		}
	}
	if _, err := readEncryptedAuth("relative-home", []byte("synthetic")); !errors.Is(err, ErrProtocol) {
		t.Fatal("relative authorization home accepted")
	}
	if err := os.MkdirAll(filepath.Join(home, "secrets"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, body := range [][]byte{nil, make([]byte, maxEncryptedAuthBytes+1), []byte("corrupt")} {
		if err := os.WriteFile(filepath.Join(home, "secrets", "codex_auth.age"), body, 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := readEncryptedAuth(home, []byte("synthetic")); err == nil {
			t.Fatal("invalid encrypted source accepted")
		}
	}
}
