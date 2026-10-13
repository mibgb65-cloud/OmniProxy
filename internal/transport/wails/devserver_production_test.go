//go:build production

package wails

import (
	"context"
	"testing"
)

func TestProductionIgnoresDevServer(t *testing.T) {
	t.Setenv("FRONTEND_DEVSERVER_URL", "https://example.com")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := waitForFrontend(ctx, nil); err != nil {
		t.Fatalf("生产版不应等待开发服务：%v", err)
	}
}
