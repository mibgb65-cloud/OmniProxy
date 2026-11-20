package proxy

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// 流式上游在首个可见分片之前可能长时间思考，而下游客户端先等响应头。
// 静默超过 streamHeaderGrace 就提交 200 与响应头，之后按 streamHeartbeatInterval
// 发送 SSE 注释行，客户端不会把「正在思考」当成「服务端没有响应」。
// 宽限期保留首个事件即失败时的 HTTP 错误语义与账号轮换。
const (
	streamHeaderGrace       = 3 * time.Second
	streamHeartbeatInterval = 10 * time.Second
	streamWriteTimeout      = 15 * time.Second
	streamHeartbeatFrame    = ": keep-alive\n\n"
)

// 测试需要缩短等待；生产固定使用上面的默认值。
var (
	streamHeaderGraceDelay = streamHeaderGrace
	streamHeartbeatPeriod  = streamHeartbeatInterval
)

// needsEarlyStreamHeaders 判断是否要在等上游响应头时先提交下游 SSE 响应头。
//
// 只有 Antigravity 桌面端把响应头等待限制在 60 秒（Antigravity 日志表现为
// `timeout awaiting response headers`），而订阅的思考型模型可能先想几分钟才回第一个
// 分片。其它客户端维持原有语义：失败仍以真实 HTTP 状态码返回。
func needsEarlyStreamHeaders(client string, stream bool) bool {
	return stream && client == "antigravity"
}

// startStreamHeaderKeeper 在等待上游响应头期间先提交下游 SSE 响应头并保活。
//
// provider 的 Forward 只有在拿到上游响应头后才回调 consume，因此这段时间网关原本一个
// 字节都不写；对 60 秒响应头预算的桌面端来说就是无响应。这里先提交 200 与 SSE 响应头，
// 再按既有心跳间隔发注释行，直到正文开始复制。
//
// 返回的函数可重复调用：首次调用关闭保活并等待协程退出，此后写响应只属于调用方，
// 保证保活与正文复制不会并发写 ResponseWriter。
func startStreamHeaderKeeper(w http.ResponseWriter, committed *bool) func() {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		grace := time.NewTimer(streamHeaderGraceDelay)
		defer grace.Stop()
		select {
		case <-grace.C:
		case <-stop:
			return
		}
		// 提交失败说明客户端已断开：不再重试，由请求主流程结束这次请求。
		if commitStreamHeaders(w, committed) != nil {
			return
		}
		heartbeat := time.NewTicker(streamHeartbeatPeriod)
		defer heartbeat.Stop()
		for {
			select {
			case <-heartbeat.C:
				if writeStreamFrame(w, streamHeartbeatFrame) != nil {
					return
				}
			case <-stop:
				return
			}
		}
	}()
	var once sync.Once
	return func() {
		once.Do(func() {
			close(stop)
			<-done
		})
	}
}

// commitStreamHeaders 提交 200 与调用方已设置的响应头并立即刷新。
// committed 用来让之后的复制知道状态码已发出，不能重复 WriteHeader。
func commitStreamHeaders(w http.ResponseWriter, committed *bool) error {
	controller := http.NewResponseController(w)
	if err := controller.SetWriteDeadline(time.Now().Add(streamWriteTimeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return Failure("client_write_failed", 502)
	}
	w.WriteHeader(http.StatusOK)
	if committed != nil {
		*committed = true
	}
	if err := controller.Flush(); err != nil {
		return Failure("client_write_failed", 502)
	}
	return nil
}

// writeStreamFrame 写入一个完整 SSE 帧并刷新；帧内容由调用方保证完整。
func writeStreamFrame(w http.ResponseWriter, frame string) error {
	controller := http.NewResponseController(w)
	if err := controller.SetWriteDeadline(time.Now().Add(streamWriteTimeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return Failure("client_write_failed", 502)
	}
	if _, err := io.WriteString(w, frame); err != nil {
		return Failure("client_write_failed", 502)
	}
	if err := controller.Flush(); err != nil {
		return Failure("client_write_failed", 502)
	}
	return nil
}

// copyStream 逐行转发 SSE，并在上游静默期先提交响应头、再发心跳注释。
// 读取放在独立协程，写入仍只发生在调用协程，避免并发写 ResponseWriter。
func copyStream(w http.ResponseWriter, body io.Reader, committed *bool, protocol ...string) error {
	var trace *Trace
	if tracked, ok := w.(*trackedWriter); ok {
		trace = tracked.trace
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("X-Accel-Buffering", "no")
	scanner := bufio.NewScanner(io.LimitReader(body, maxStreamBytes+1))
	lineLimit := 512 << 10
	if len(protocol) > 0 && protocol[0] == "responses" {
		lineLimit = MaxResponsesEventBytes
	}
	scanner.Buffer(make([]byte, 4096), lineLimit)

	lines, scanErr, stop := pumpStream(scanner)
	defer close(stop)

	headersSent := committed != nil && *committed
	// commitHeaders 只执行一次；响应头发出后无法再改状态码。
	commitHeaders := func() error {
		if headersSent {
			return nil
		}
		if err := commitStreamHeaders(w, committed); err != nil {
			return err
		}
		headersSent = true
		return nil
	}
	writeFrame := func(frame string) error {
		if err := commitHeaders(); err != nil {
			return err
		}
		return writeStreamFrame(w, frame)
	}

	grace := time.NewTimer(streamHeaderGraceDelay)
	defer grace.Stop()
	heartbeat := time.NewTicker(streamHeartbeatPeriod)
	defer heartbeat.Stop()
	firstContent := false
	eventOpen := false
	var total int
	for {
		select {
		case <-grace.C:
			if err := commitHeaders(); err != nil {
				return err
			}
		case <-heartbeat.C:
			// 事件中间不能插入注释行：心跳自带的空行会提前终结正在拼装的事件。
			if eventOpen {
				continue
			}
			if err := writeFrame(streamHeartbeatFrame); err != nil {
				return err
			}
		case line, ok := <-lines:
			if !ok {
				var readErr error
				select {
				case readErr = <-scanErr:
				default:
				}
				// 没有终结事件就结束的流与「数据行不合语法」同码不同因，日志必须能区分；
				// 单行超限另有 response_too_large，不算提前收流。
				if !errors.Is(readErr, bufio.ErrTooLong) {
					trace.markStreamFailure(streamDataFailure{kind: streamDataIncompleteStream})
				}
				return streamReadFailure(readErr)
			}
			total += len(line) + 1
			if total > maxStreamBytes {
				return Failure("response_too_large", 502)
			}
			done, content, err := streamLineState(line, trace)
			if err != nil {
				return err
			}
			firstContent = firstContent || content
			eventOpen = line != ""
			frame := line + "\n"
			if done {
				frame += "\n"
			}
			if err := writeFrame(frame); err != nil {
				return err
			}
			// 客户端在空行后才能解析完整 SSE 事件；不能在 data 行或响应头处提前计时。
			if firstContent && (line == "" || done) {
				trace.markFirstToken()
				firstContent = false
			}
			if done {
				return nil
			}
		}
	}
}

// pumpStream 把扫描结果交给主协程；stop 关闭后协程不再阻塞在发送上。
func pumpStream(scanner *bufio.Scanner) (chan string, chan error, chan struct{}) {
	lines := make(chan string)
	scanErr := make(chan error, 1)
	stop := make(chan struct{})
	go func() {
		defer close(lines)
		for scanner.Scan() {
			select {
			case lines <- scanner.Text():
			case <-stop:
				return
			}
		}
		scanErr <- scanner.Err()
	}()
	return lines, scanErr, stop
}

// streamLineState 校验单个事件行并报告终结事件与首个可见内容。
func streamLineState(line string, trace *Trace) (bool, bool, error) {
	if !strings.HasPrefix(line, "data:") {
		return false, false, nil
	}
	data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	if data == "[DONE]" {
		return true, trace.hasFirstContent([]byte(data)), nil
	}
	if data != "" {
		var event struct {
			Type string `json:"type"`
		}
		// Responses 的终结事件有三种：completed、incomplete（命中输出上限或内容过滤）
		// 与 failed（按错误处理）。incomplete 是完整交付的收尾事件，不能当成流被截断。
		if json.Unmarshal([]byte(data), &event) == nil &&
			(event.Type == "response.completed" || event.Type == "response.incomplete" || event.Type == "message_stop") {
			return true, trace.hasFirstContent([]byte(data)), nil
		}
		if failure, failed := inspectStreamData([]byte(data)); failed {
			// 失败原因只保留固定分类与上游短码；说明文字不进日志也不回显。
			trace.markStreamFailure(failure)
			return false, false, Failure(failure.code, 502)
		}
	}
	return false, trace.hasFirstContent([]byte(data)), nil
}
