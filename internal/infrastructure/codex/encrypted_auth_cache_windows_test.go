//go:build windows

package codex

import (
	"bytes"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"omniproxy/internal/infrastructure/credential"
)

func cacheFixture(t *testing.T) (*encryptedAuthCache, *int, string) {
	t.Helper()
	calls := new(int)
	cache := newEncryptedAuthCache(func(body, pass []byte) ([]byte, error) {
		(*calls)++
		if string(pass) != "synthetic-pass" {
			return nil, ErrProtocol
		}
		return append([]byte("synthetic-auth:"), body...), nil
	})
	path := filepath.Join(t.TempDir(), "secrets", "codex_auth.age")
	t.Cleanup(func() {
		cache.mu.Lock()
		defer cache.mu.Unlock()
		for path := range cache.entries {
			cache.remove(path)
		}
	})
	return cache, calls, path
}

func TestEncryptedAuthCacheSourcePassphraseAndHomeIsolation(t *testing.T) {
	cache, calls, path := cacheFixture(t)
	pass := []byte("synthetic-pass")
	original, err := cache.read(path, []byte("one"), pass)
	if err != nil {
		t.Fatal(err)
	}
	first := cache.entries[path].secret
	clear(original)
	for range 3 {
		got, err := cache.read(path, []byte("one"), pass)
		if err != nil || string(got) != "synthetic-auth:one" {
			t.Fatal("cached authorization copy was lost")
		}
		clear(got)
	}
	if *calls != 1 {
		t.Fatal("unchanged encrypted source repeated KDF")
	}
	got, err := cache.read(path, []byte("two"), pass)
	clear(got)
	if err != nil || *calls != 2 {
		t.Fatal("changed ciphertext reused stale authorization")
	}
	if _, err := first.Open(); !errors.Is(err, credential.ErrNotFound) {
		t.Fatal("replaced protected value was not cleared")
	}
	if _, err := cache.read(path, []byte("two"), []byte("wrong")); !errors.Is(err, ErrProtocol) || len(cache.entries) != 0 {
		t.Fatal("changed system passphrase bypassed decryption failure")
	}
	for _, file := range []string{path, path + "-other-profile"} {
		got, err := cache.read(file, []byte("two"), pass)
		clear(got)
		if err != nil {
			t.Fatal(err)
		}
	}
	if *calls != 5 || len(cache.entries) != 2 {
		t.Fatal("different authorization homes shared one protected entry")
	}
}

func TestEncryptedAuthCacheFixedExpiryCapacityAndClearHome(t *testing.T) {
	cache, calls, path := cacheFixture(t)
	now := time.Unix(1_000_000, 0)
	cache.now = func() time.Time { return now }
	read := func(path string) {
		t.Helper()
		data, err := cache.read(path, []byte("one"), []byte("synthetic-pass"))
		clear(data)
		if err != nil {
			t.Fatal(err)
		}
	}
	read(path)
	expired := cache.entries[path].secret
	now = now.Add(time.Minute)
	read(path)
	now = now.Add(encryptedAuthCacheLifetime - time.Minute)
	read(path)
	if *calls != 2 {
		t.Fatal("cache hits extended the fixed expiry")
	}
	if _, err := expired.Open(); !errors.Is(err, credential.ErrNotFound) {
		t.Fatal("expired protected authorization was not cleared")
	}
	first := cache.entries[path].secret
	for index := range encryptedAuthCacheLimit {
		now = now.Add(time.Second)
		read(filepath.Join(path, string(rune('a'+index))))
	}
	if len(cache.entries) != encryptedAuthCacheLimit {
		t.Fatal("encrypted authorization cache exceeded its bound")
	}
	if _, err := first.Open(); !errors.Is(err, credential.ErrNotFound) {
		t.Fatal("eviction retained the old protected value")
	}
	read(path)
	cache.clearHome(filepath.Dir(filepath.Dir(path)))
	if _, ok := cache.entries[path]; ok {
		t.Fatal("logout kept the protected authorization entry")
	}
}

func TestEncryptedAuthCacheConcurrentMissHasOneDecryption(t *testing.T) {
	cache, calls, path := cacheFixture(t)
	var group sync.WaitGroup
	values := make([][]byte, 24)
	for index := range values {
		group.Go(func() {
			var err error
			values[index], err = cache.read(path, []byte("one"), []byte("synthetic-pass"))
			if err != nil {
				t.Error(err)
			}
		})
	}
	group.Wait()
	if *calls != 1 {
		t.Fatal("concurrent readers allocated multiple KDF workspaces")
	}
	clear(values[0])
	for _, value := range values[1:] {
		if !bytes.Equal(value, []byte("synthetic-auth:one")) {
			t.Fatal("concurrent consumers shared plaintext storage")
		}
		clear(value)
	}
}
