package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"

	"github.com/coder/websocket"
)

// socketCancelledStatus 与 HTTP 侧 request_cancelled 一致：未完成的回答不是传输失败。
const socketCancelledStatus = 499

// socketTurnFrame 只解析开启一轮所需的字段，不保留正文或凭据。
type socketTurnFrame struct {
	Type     string `json:"type"`
	Model    string `json:"model"`
	Generate *bool  `json:"generate"`
}

// socketAccounting 为透明 WebSocket 补齐逐轮记账。
//
// 透明转发的帧字节必须原样中继，因此只能在已经转发的帧上旁路识别轮次与用量：
// 客户端每次 response.create 开一轮，上游终结事件写入那一轮的用量。
// 解析失败只丢统计，不改变交付给任何一侧的字节。
// 两个方向的中继各自在独立 goroutine 里运行，轮次状态必须由 mu 保护。
type socketAccounting struct {
	mu       sync.Mutex
	server   *Server
	client   string
	snapshot TargetSnapshot
	model    string
	protocol string
	turn     *socketTurn
}

type socketTurn struct{ trace *Trace }

// newSocketAccounting 绑定本次连接实际使用的账号身份、运行形态与协议分类。
// client 必须已经由 recordedClient 细化到 codex-cli／codex-desktop 这一级，
// 否则请求记录会退回笼统的 codex。
func newSocketAccounting(server *Server, client string, attempt PassthroughAttempt, path string) *socketAccounting {
	return &socketAccounting{server: server, client: client, snapshot: attempt.Snapshot, model: attempt.Model, protocol: passthroughProtocol(path)}
}

// clientFrame 在转发客户端帧之前开一轮：没有记账就不能把消费送到上游。
func (a *socketAccounting) clientFrame(ctx context.Context, kind websocket.MessageType, data []byte) error {
	if a == nil || kind != websocket.MessageText {
		return nil
	}
	// 帧解析不需要持锁，避免长帧阻塞上游方向的统计。
	var frame socketTurnFrame
	if json.Unmarshal(data, &frame) != nil || frame.Type != "response.create" {
		return nil
	}
	// generate:false 是预热轮，不产生回答也不产生用量；旧协议入口同样不记账。
	if frame.Generate != nil && !*frame.Generate {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	// 上轮没有终结事件就又开始新一轮：先按未完成收尾，不让它一直悬着。
	a.closeTurn(ctx, socketCancelledStatus, "request_cancelled")
	return a.beginTurn(ctx, frame.Model)
}

// beginTurn 为新的一轮建立父记录与账号尝试；失败时同时收尾已经建立的记录。
func (a *socketAccounting) beginTurn(ctx context.Context, model string) error {
	if model == "" {
		model = a.model
	}
	request := Request{Model: model, Stream: true}
	trace, err := a.server.beginTrace(ctx, a.protocol, "websocket", a.client)
	if err != nil {
		a.logFailure()
		return err
	}
	a.turn = &socketTurn{trace: trace}
	if err = trace.BeginAttempt(ctx, request, a.snapshot); err == nil {
		// 父记录必须带上真实账号身份：额度估算按账号／成员／工作区过滤本机记录。
		err = trace.Update(ctx, request, a.snapshot)
	}
	if err != nil {
		a.logFailure()
		a.closeTurn(ctx, http.StatusServiceUnavailable, "accounting_unavailable")
		return err
	}
	return nil
}

// upstreamFrame 在帧已交给客户端之后读取首字与用量，统计失败不影响已送达的字节。
func (a *socketAccounting) upstreamFrame(ctx context.Context, kind websocket.MessageType, data []byte) {
	if a == nil || kind != websocket.MessageText {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.turn == nil {
		return
	}
	if !terminalStreamEvent(data, a.protocol) {
		if a.turn.trace.hasFirstContent(data) {
			a.turn.trace.markFirstToken()
		}
		return
	}
	// 与 HTTP 流一致：终结事件就是这一轮的结束边界，之后连接仍可继续下一轮。
	if a.turn.trace.Report(ctx, ParseUsage(data, a.protocol, true)) != nil {
		a.logFailure()
	}
	a.closeTurn(ctx, http.StatusOK, "")
}

// closeTurn 结束当前一轮并清空；没有终结事件就断开时按未完成的回答收尾。
// 调用方必须持有 a.mu。
func (a *socketAccounting) closeTurn(ctx context.Context, status int, code string) {
	if a == nil || a.turn == nil {
		return
	}
	if err := a.turn.trace.EndAttempt(status, code); err != nil && code == "" {
		// 成功轮次必须交出真实尝试结果，否则请求详情会缺一条尝试。
		a.logFailure()
		code = "accounting_unavailable"
		status = http.StatusServiceUnavailable
	}
	a.turn.trace.completeLogged(status, code, a.server.opts.Logger)
	a.turn = nil
}

// close 用中继的真实结束原因收尾，取消与上游抖动必须区分。
func (a *socketAccounting) close(ctx context.Context, err error) {
	if a == nil {
		return
	}
	status, code := socketOutcome(err)
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closeTurn(ctx, status, code)
}

// socketOutcome 把中继结束原因映射成稳定状态码与错误码。
// 上游只关闭连接而没有交付终结事件时，这一轮是未完成的回答，不是传输成功。
// 客户端在终结事件之前断开同样记为取消，与 HTTP 侧的 request_cancelled 语义一致。
func socketOutcome(err error) (int, string) {
	var relay *socketRelayError
	if errors.As(err, &relay) {
		if relay.closeCode == websocket.StatusNormalClosure || relay.closeCode == websocket.StatusGoingAway || relay.direction == "client_disconnected" {
			return socketCancelledStatus, "request_cancelled"
		}
		var failure *Error
		if errors.As(relay.failure, &failure) {
			return failure.StatusCode, failure.Code
		}
	}
	var failure *Error
	if errors.As(err, &failure) {
		return failure.StatusCode, failure.Code
	}
	return socketCancelledStatus, "request_cancelled"
}

// logFailure 只输出稳定分类，不含请求内容、账号或底层错误文本。
func (a *socketAccounting) logFailure() {
	if a.server.opts.Logger == nil {
		return
	}
	a.server.opts.Logger.Warn("gateway websocket accounting failed", "module", "gateway", "code", "accounting_unavailable")
}
