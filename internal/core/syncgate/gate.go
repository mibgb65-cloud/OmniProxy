// Package syncgate 提供可取消的账号级互斥与全局排他门。
package syncgate

import (
	"context"
	"sync"
)

// Gate 按账号串行化授权／额度操作；登录提交保留全局排他。
// 不同账号不会因额度联网互相阻塞，注销和同账号借凭据仍互斥。
type Gate struct {
	mu            sync.Mutex
	changed       chan struct{}
	global        bool
	waitingGlobal int
	accounts      map[string]bool
}

func (g *Gate) Lock(ctx context.Context) error {
	return g.lock(ctx, "")
}

func (g *Gate) LockAccount(ctx context.Context, id string) error {
	return g.lock(ctx, id)
}

func (g *Gate) lock(ctx context.Context, id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.changed == nil {
		g.changed = make(chan struct{})
		g.accounts = make(map[string]bool)
	}
	if id == "" {
		g.waitingGlobal++
		defer func() { g.waitingGlobal--; g.notify() }()
	}
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !g.global && (id == "" && len(g.accounts) == 0 || id != "" && !g.accounts[id] && g.waitingGlobal == 0) {
			if id == "" {
				g.global = true
			} else {
				g.accounts[id] = true
			}
			if err := ctx.Err(); err != nil {
				if id == "" {
					g.global = false
				} else {
					delete(g.accounts, id)
				}
				g.notify()
				return err
			}
			return nil
		}
		changed := g.changed
		g.mu.Unlock()
		select {
		case <-changed:
		case <-ctx.Done():
		}
		g.mu.Lock()
	}
}

func (g *Gate) notify() {
	close(g.changed)
	g.changed = make(chan struct{})
}

func (g *Gate) Unlock() { g.UnlockAccount("") }

func (g *Gate) UnlockAccount(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if id == "" {
		g.global = false
	} else {
		delete(g.accounts, id)
	}
	g.notify()
}
