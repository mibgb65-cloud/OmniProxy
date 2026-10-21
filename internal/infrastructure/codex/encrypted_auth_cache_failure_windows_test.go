//go:build windows

package codex

import (
	"errors"
	"testing"

	"omniproxy/internal/infrastructure/credential"
)

func TestEncryptedAuthCacheProtectionAndCorruptionFailClosed(t *testing.T) {
	cache, calls, path := cacheFixture(t)
	pass := []byte("synthetic-pass")
	data, err := cache.read(path, []byte("one"), pass)
	clear(data)
	if err != nil {
		t.Fatal(err)
	}
	cache.entries[path].secret.Clear()
	if _, err := cache.read(path, []byte("one"), pass); !errors.Is(err, ErrUnavailable) || *calls != 1 || len(cache.entries) != 0 {
		t.Fatal("corrupt cache silently fell back or returned stale data")
	}
	var plaintext []byte
	cache.decrypt = func([]byte, []byte) ([]byte, error) {
		plaintext = []byte("synthetic-protection-failure")
		return plaintext, nil
	}
	cache.protect = func([]byte) (*credential.MemorySecret, error) {
		return nil, errors.New("private-protector-detail")
	}
	if _, err := cache.read(path, []byte("one"), pass); !errors.Is(err, ErrUnavailable) || len(cache.entries) != 0 {
		t.Fatal("protection failure was exposed or downgraded to plaintext caching")
	}
	for _, value := range plaintext {
		if value != 0 {
			t.Fatal("protection failure retained plaintext")
		}
	}
}

func TestEncryptedAuthCacheDecryptionFailureAndSizeBounds(t *testing.T) {
	for _, size := range []int{0, maxAuthValueBytes + 1} {
		cache, _, path := cacheFixture(t)
		plaintext := make([]byte, size)
		cache.decrypt = func([]byte, []byte) ([]byte, error) { return plaintext, nil }
		if _, err := cache.read(path, nil, nil); !errors.Is(err, ErrProtocol) || len(cache.entries) != 0 {
			t.Fatal("invalid authorization size was cached")
		}
	}
	cache, _, path := cacheFixture(t)
	plaintext := []byte("synthetic-failed-decrypt-output")
	cache.decrypt = func([]byte, []byte) ([]byte, error) { return plaintext, ErrProtocol }
	if _, err := cache.read(path, nil, nil); !errors.Is(err, ErrProtocol) || len(cache.entries) != 0 {
		t.Fatal("failed decryption was cached")
	}
	for _, value := range plaintext {
		if value != 0 {
			t.Fatal("failed decryption retained plaintext output")
		}
	}
}
