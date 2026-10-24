package proxy

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// failingSession 用固定的失败原因代替账号策略，断言网关不再把它折叠成 502。
type failingSession struct{ err error }

func (s failingSession) Next(context.Context) (PassthroughAttempt, error) {
	return PassthroughAttempt{}, s.err
}
func (failingSession) Retryable(PassthroughAttempt, int) bool                       { return false }
func (failingSession) Switchable() bool                                             { return true }
func (failingSession) Done(context.Context, PassthroughAttempt, PassthroughOutcome) {}

type failingBinding struct{ err error }

func (b failingBinding) OpenPassthrough(context.Context, PassthroughRequest) (PassthroughSession, error) {
	return failingSession(b), nil
}

// TestPassthroughSocketSurfacesSessionFailure 断言账号侧的真实失败原因原样交给客户端：
// 全部压成 502 upstream_unavailable 会让客户端以为只是上游抖动而无限重连。
func TestPassthroughSocketSurfacesSessionFailure(t *testing.T) {
	server := New(Options{Port: 0, Passthrough: failingBinding{Failure("credential_unavailable", 503)}, Upstream: unusedUpstream(t)})
	if _, err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = server.Stop(context.Background()) }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	dialURL := "ws" + strings.TrimPrefix(server.Status().Address, "http") + "/codex/v1/responses"
	socket, resp, err := websocket.Dial(ctx, dialURL, nil)
	if socket != nil {
		socket.CloseNow()
	}
	if err == nil || resp == nil {
		t.Fatalf("expected handshake rejection, got err=%v resp=%v", err, resp)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &body) != nil {
		t.Fatalf("unexpected body %q", raw)
	}
	if resp.StatusCode != http.StatusServiceUnavailable || body.Error.Code != "credential_unavailable" {
		t.Fatalf("status = %d code = %q body = %q", resp.StatusCode, body.Error.Code, raw)
	}
}
