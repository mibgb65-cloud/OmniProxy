package proxy

import (
	"bytes"
	"context"
	"io"
	"mime"
)

type usageReader struct {
	reader   io.Reader
	trace    *Trace
	ctx      context.Context
	protocol string
	stream   bool
	// strict 决定记账失败是否中断读流。协议转换路径自己生成响应，必须让记账失败可见；
	// 透明转发只能原样搬运上游字节，不能因为一次入库失败就截断已经交付的回答。
	strict         bool
	buffer         []byte
	overflow, done bool
	total          int
}

// ObserveResponse 挂在原上游读流，转换后的默认零值不会进入统计。
func ObserveResponse(ctx context.Context, trace *Trace, response Response, protocol string) Response {
	trace.observeStatus(response.StatusCode)
	trace.observeResponseHeaders(response)
	if trace == nil || response.PreExecutionQuotaRejected || response.StatusCode != 200 || response.Body == nil {
		return response
	}
	kind, _, err := mime.ParseMediaType(response.ContentType)
	if err != nil {
		return response
	}
	if kind != "application/json" && kind != "text/event-stream" {
		return response
	}
	response.Body = &usageReader{reader: response.Body, trace: trace, ctx: ctx, protocol: protocol, stream: kind == "text/event-stream", strict: true}
	return response
}

func (r *usageReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.total += n
	if r.total > maxStreamBytes {
		r.overflow = true
	}
	var reportErr error
	if !r.overflow {
		r.buffer = append(r.buffer, p[:n]...)
		if r.stream {
			for {
				end := bytes.IndexByte(r.buffer, '\n')
				if end < 0 {
					break
				}
				reportErr = r.observeLine(bytes.TrimSpace(r.buffer[:end]))
				r.buffer = r.buffer[end+1:]
			}
			if len(r.buffer) > 2<<20 {
				r.overflow = true
				r.buffer = nil
			}
		} else if len(r.buffer) > maxResponseBytes {
			r.overflow = true
			r.buffer = nil
		}
	}
	if err == io.EOF && r.stream && !r.overflow && !r.done {
		r.done = true
		reportErr = r.observeLine(bytes.TrimSpace(r.buffer))
		clear(r.buffer)
		r.buffer = nil
	}
	if err == io.EOF && !r.stream && !r.overflow && !r.done {
		r.done = true
		reportErr = r.observeBody(r.buffer)
		clear(r.buffer)
		r.buffer = nil
	}
	if reportErr != nil {
		return 0, reportErr
	}
	return n, err
}

// observeBody 观察非流式正文：整段 JSON 只有一次用量。
func (r *usageReader) observeBody(body []byte) error {
	r.trace.markTerminal()
	return r.report(ParseUsage(body, r.protocol, false))
}

// observeLine 只观察一段上游事件：终结事件据实标记，用量按协议快照累计。
func (r *usageReader) observeLine(line []byte) error {
	if !bytes.HasPrefix(line, []byte("data:")) {
		return nil
	}
	payload := bytes.TrimSpace(line[5:])
	if bytes.Equal(payload, []byte("[DONE]")) {
		r.trace.markTerminal()
		return nil
	}
	if terminalStreamEvent(payload, r.protocol) {
		r.trace.markTerminal()
	}
	return r.report(ParseUsage(payload, r.protocol, r.stream))
}

// report 只把非零用量交给 trace；记账失败在严格模式下上抛，其余情况只标记状态。
func (r *usageReader) report(usage UsageSnapshot) error {
	if !usage.InputKnown && !usage.OutputKnown {
		return nil
	}
	if err := r.trace.Report(r.ctx, usage); err != nil {
		r.trace.markAccountingFailure()
		if r.strict {
			return err
		}
	}
	return nil
}
