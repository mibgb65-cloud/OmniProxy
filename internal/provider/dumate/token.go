package dumate

import (
	"context"
	"sync"
	"time"
)

const (
	maxTokenEntries = 128
	tokenFetchLimit = 30 * time.Second
)

// TokenCache 按账号与凭据版本缓存 gateway token，并合并并发刷新。
type TokenCache struct {
	mu      sync.Mutex
	entries map[string]*tokenEntry
	now     func() time.Time
}

type tokenEntry struct {
	token   Token
	pending *tokenCall
}

type tokenCall struct {
	done  chan struct{}
	token Token
	err   error
}

// NewTokenCache 构造有界的内存缓存。
func NewTokenCache() *TokenCache {
	return &TokenCache{entries: map[string]*tokenEntry{}, now: time.Now}
}

// Get 返回仍然有效的 token；同一 key 的并发调用只发起一次兑换。
func (c *TokenCache) Get(ctx context.Context, key string, fetch func(context.Context) (Token, error)) (Token, error) {
	if c == nil || fetch == nil || key == "" {
		return Token{}, Failure(CodeCredentialUnavailable)
	}
	c.mu.Lock()
	if entry := c.entries[key]; entry != nil {
		if entry.pending != nil {
			call := entry.pending
			c.mu.Unlock()
			return waitToken(ctx, call)
		}
		if c.valid(entry.token) {
			token := entry.token
			c.mu.Unlock()
			return token, nil
		}
	}
	call := &tokenCall{done: make(chan struct{})}
	entry := c.entries[key]
	if entry == nil {
		entry = &tokenEntry{}
		c.entries[key] = entry
		c.evictLocked()
	}
	entry.pending = call
	c.mu.Unlock()

	// 兑换不受触发请求的取消影响，等待同一 token 的其他请求才能共享结果。
	fetchCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), tokenFetchLimit)
	token, err := fetch(fetchCtx)
	cancel()
	c.finish(key, call, token, err)
	return waitToken(ctx, call)
}

func (c *TokenCache) finish(key string, call *tokenCall, token Token, err error) {
	c.mu.Lock()
	call.token, call.err = token, err
	// 兑换期间条目可能已被 Invalidate 或新调用替换，只能更新自己占用中的条目。
	if entry := c.entries[key]; entry != nil && entry.pending == call {
		entry.pending = nil
		if err == nil && token.Value != "" {
			entry.token = token
		} else if !c.valid(entry.token) {
			delete(c.entries, key)
		}
	}
	c.mu.Unlock()
	close(call.done)
}

func waitToken(ctx context.Context, call *tokenCall) (Token, error) {
	select {
	case <-call.done:
		if call.err != nil {
			return Token{}, call.err
		}
		return call.token, nil
	case <-ctx.Done():
		return Token{}, Failure(CodeConsoleUnavailable)
	}
}

func (c *TokenCache) valid(token Token) bool {
	return token.Value != "" && c.now().Add(TokenRefreshMargin).Before(token.ExpiresAt)
}

// Invalidate 丢弃一个 key 的缓存；401 或重新导入后调用。
func (c *TokenCache) Invalidate(key string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// evictLocked 在超限时丢弃最早过期且没有在途刷新的条目。
func (c *TokenCache) evictLocked() {
	for len(c.entries) > maxTokenEntries {
		var oldestKey string
		var oldest time.Time
		for key, entry := range c.entries {
			if entry.pending != nil {
				continue
			}
			if oldestKey == "" || entry.token.ExpiresAt.Before(oldest) {
				oldestKey, oldest = key, entry.token.ExpiresAt
			}
		}
		if oldestKey == "" {
			return
		}
		delete(c.entries, oldestKey)
	}
}
