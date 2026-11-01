package codex

import (
	"context"
	"testing"
	"time"
)

func waitQAThread(ctx context.Context, t *testing.T, client *rpcClient, id string, compact bool) {
	t.Helper()
	for range 1000 {
		var result struct {
			Thread struct {
				Turns []struct {
					Status string `json:"status"`
					Items  []struct {
						Type string `json:"type"`
					} `json:"items"`
				} `json:"turns"`
			} `json:"thread"`
		}
		if err := client.call(ctx, "thread/read", map[string]any{"threadId": id, "includeTurns": true}, &result); err != nil {
			// 新线程的 rollout 尚未物化时读取可能暂不可用；有界等待，不能当作压缩已经成功。
			if err.Error() != "subscription: request_failed" || ctx.Err() != nil {
				t.Fatal("QA thread state unavailable")
			}
		}
		if turns := result.Thread.Turns; len(turns) > 0 {
			last := turns[len(turns)-1]
			if last.Status == "failed" {
				t.Fatal("isolated native compaction turn failed")
			}
			if last.Status == "completed" {
				if !compact {
					return
				}
				for _, item := range last.Items {
					if item.Type == "contextCompaction" {
						return
					}
				}
			}
		}
		select {
		case <-ctx.Done():
			t.Fatal("QA thread timed out")
		case <-time.After(20 * time.Millisecond):
		}
	}
	t.Fatal("QA thread did not reach expected completed state")
}
