package proxy

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"

	"github.com/coder/websocket"
)

// Codex 的 generate=false 只预热连接，不得变成真实上游生成或占用账号额度。
// 预热 ID 及输入仅在当前连接保存，下一帧通过 previous_response_id 衔接上下文。
func socketWarmup(ctx context.Context, socket *websocket.Conn) (string, error) {
	value := make([]byte, 16)
	if _, err := rand.Read(value); err != nil {
		return "", Failure("upstream_unavailable", 502)
	}
	id := "resp_warmup_" + hex.EncodeToString(value)
	for _, kind := range []string{"response.created", "response.completed"} {
		status := "in_progress"
		if kind == "response.completed" {
			status = "completed"
		}
		data, err := json.Marshal(map[string]any{"type": kind, "response": map[string]any{"id": id, "object": "response", "status": status, "output": []json.RawMessage{}}})
		if err != nil {
			return "", err
		}
		if err = socket.Write(ctx, websocket.MessageText, data); err != nil {
			return "", err
		}
	}
	return id, nil
}
