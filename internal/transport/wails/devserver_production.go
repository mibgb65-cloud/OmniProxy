//go:build production

package wails

import (
	"context"
	"log/slog"
)

// 生产版使用内嵌前端，环境中的开发地址不能引入额外网络访问或阻塞启动。
func waitForFrontend(_ context.Context, _ *slog.Logger) error {
	return nil
}
