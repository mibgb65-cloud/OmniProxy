package subscription

import (
	"context"
	"time"

	"omniproxy/internal/infrastructure/codex"
)

func readAccountID(ctx context.Context, client codex.Client) (string, error) {
	// 用量接口依赖网络；附加身份信息不可阻塞已成功的授权保存。
	idCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	id, err := client.AccountID(idCtx)
	if err != nil || id == "" {
		return "", failure("account_id_pending")
	}
	return id, nil
}
