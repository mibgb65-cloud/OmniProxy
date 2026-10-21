package codex

import (
	"crypto/sha256"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"omniproxy/internal/infrastructure/credential"
)

const encryptedAuthCacheLimit = 16
const encryptedAuthCacheLifetime = 5 * time.Minute

type authFingerprint struct{ ciphertext, passphrase [32]byte }
type encryptedAuthEntry struct {
	fingerprint authFingerprint
	secret      *credential.MemorySecret
	expires     time.Time
}
type encryptedAuthCache struct {
	mu      sync.Mutex
	entries map[string]encryptedAuthEntry
	now     func() time.Time
	decrypt func([]byte, []byte) ([]byte, error)
	protect func([]byte) (*credential.MemorySecret, error)
}

var officialAuthCache = newEncryptedAuthCache(decryptAuthPayload)

func newEncryptedAuthCache(decrypt func([]byte, []byte) ([]byte, error)) *encryptedAuthCache {
	return &encryptedAuthCache{
		entries: make(map[string]encryptedAuthEntry), now: time.Now,
		decrypt: decrypt, protect: credential.NewMemorySecret,
	}
}

func (c *encryptedAuthCache) read(path string, ciphertext, passphrase []byte) ([]byte, error) {
	key := authFingerprint{sha256.Sum256(ciphertext), sha256.Sum256(passphrase)}
	// 官方 KDF 单次需要约 1 GiB；不同调用入口也必须串行，避免多个工作区叠加。
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	for path, entry := range c.entries {
		if !entry.expires.After(now) {
			c.remove(path)
		}
	}
	if entry, ok := c.entries[path]; ok {
		if entry.fingerprint == key {
			data, err := entry.secret.Open()
			if err != nil {
				clear(data)
				c.remove(path)
				return nil, ErrUnavailable
			}
			return data, nil
		}
		c.remove(path)
	}
	data, err := c.decrypt(ciphertext, passphrase)
	if err != nil {
		clear(data)
		return nil, err
	}
	if len(data) == 0 || len(data) > maxAuthValueBytes {
		clear(data)
		return nil, ErrProtocol
	}
	if err := c.store(path, key, data); err != nil {
		clear(data)
		return nil, err
	}
	return data, nil
}

// store 只在持有缓存锁时调用；密文保护失败不能退化为明文缓存。
func (c *encryptedAuthCache) store(path string, key authFingerprint, data []byte) error {
	protected, err := c.protect(data)
	if errors.Is(err, credential.ErrUnsupported) {
		// 非 Windows 的读取测试沿用原行为，绝不改为缓存明文。
		protected.Clear()
		return nil
	}
	if err != nil || protected == nil {
		protected.Clear()
		return ErrUnavailable
	}
	if len(c.entries) >= encryptedAuthCacheLimit {
		var oldestPath string
		var oldest time.Time
		for candidate, entry := range c.entries {
			if oldestPath == "" || entry.expires.Before(oldest) {
				oldestPath, oldest = candidate, entry.expires
			}
		}
		c.remove(oldestPath)
	}
	c.entries[path] = encryptedAuthEntry{key, protected, c.now().Add(encryptedAuthCacheLifetime)}
	return nil
}

func (c *encryptedAuthCache) remove(path string) {
	if entry, ok := c.entries[path]; ok {
		entry.secret.Clear()
		delete(c.entries, path)
	}
}

func (c *encryptedAuthCache) clearHome(home string) {
	target := filepath.Join(home, "secrets", "codex_auth.age")
	// 注销可能已删除目录；解析失败时仍按原始绝对路径幂等清理。
	realHome, err := filepath.EvalSymlinks(home)
	if err == nil {
		target = filepath.Join(realHome, "secrets", "codex_auth.age")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for path := range c.entries {
		if strings.EqualFold(path, target) {
			c.remove(path)
		}
	}
}
