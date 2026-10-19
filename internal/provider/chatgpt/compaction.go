package chatgpt

import (
	"context"
	"encoding/json"
	"omniproxy/internal/core/proxy"
)

// Compact 只调用固定官方订阅压缩端点，不修改不透明历史或强制推理流参数。
func (c *Client) Compact(ctx context.Context, key []byte, accountID string, request proxy.Request, consume func(proxy.Response) error) error {
	if !request.Compaction || request.Protocol != "responses" || request.Stream || !json.Valid(request.Body) {
		return proxy.Failure("invalid_request", 400)
	}
	return c.send(ctx, key, accountID, request.Body, true, request.CodexMetadata, consume)
}
