package subscription

import (
	"context"
	"sync"
)

// operationGate 串行化官方授权／额度操作，不覆盖模型网络消费；排队必须响应取消。
type operationGate struct {
	once sync.Once
	slot chan struct{}
}

func (g *operationGate) Lock(ctx context.Context) error {
	g.once.Do(func() { g.slot = make(chan struct{}, 1) })
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case g.slot <- struct{}{}:
		// 取得名额与取消可能同时发生，不能让失效请求继续打开官方组件。
		if err := ctx.Err(); err != nil {
			g.Unlock()
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (g *operationGate) Unlock() { <-g.slot }
