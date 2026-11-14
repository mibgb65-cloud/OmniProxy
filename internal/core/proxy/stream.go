package proxy

import (
	"bufio"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
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

// copyStream 逐行转发 SSE，并在上游静默期先提交响应头、再发心跳注释。
// 读取放在独立协程，写入仍只发生在调用协程，避免并发写 ResponseWriter。
func copyStream(w http.ResponseWriter, body io.Reader, committed *bool, protocol ...string) error {
	var trace *Trace
	if tracked, ok := w.(*trackedWriter); ok {
		trace = tracked.trace
	}
	controller := http.NewResponseController(w)
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

	headersSent := false
	// commitHeaders 只执行一次；响应头发出后无法再改状态码。
	commitHeaders := func() error {
		if headersSent {
			return nil
		}
		if err := controller.SetWriteDeadline(time.Now().Add(streamWriteTimeout)); err != nil && !errors.Is(err, http.ErrNotSupported) {
			return Failure("client_write_failed", 502)
		}
		w.WriteHeader(http.StatusOK)
		*committed = true
		headersSent = true
		if err := controller.Flush(); err != nil {
			return Failure("client_write_failed", 502)
		}
		return nil
	}
	writeFrame := func(frame string) error {
		if err := commitHeaders(); err != nil {
			return err
		}
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
	done := data == "[DONE]"
	if !done {
		var event struct {
			Type string `json:"type"`
		}
		if json.Unmarshal([]byte(data), &event) == nil {
			done = event.Type == "response.completed" || event.Type == "message_stop"
		}
	}
	if data != "" && !done && (!json.Valid([]byte(data)) || hasUpstreamError([]byte(data))) {
		return false, false, Failure("upstream_stream_failed", 502)
	}
	return done, trace.hasFirstContent([]byte(data)), nil
}
