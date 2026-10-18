package proxy

import (
	"bytes"
	"context"
	"io"
	"mime"
)

type usageReader struct {
	reader         io.Reader
	trace          *Trace
	ctx            context.Context
	protocol       string
	stream         bool
	buffer         []byte
	overflow, done bool
	total          int
}

// ObserveResponse 挂在原上游读流，转换后的默认零值不会进入统计。
func ObserveResponse(ctx context.Context, trace *Trace, response Response, protocol string) Response {
	trace.observeStatus(response.StatusCode)
	trace.observeResponseHeaders(response)
	if trace == nil || response.StatusCode != 200 || response.Body == nil {
		return response
	}
	kind, _, err := mime.ParseMediaType(response.ContentType)
	if err != nil {
		return response
	}
	if kind != "application/json" && kind != "text/event-stream" {
		return response
	}
	response.Body = &usageReader{reader: response.Body, trace: trace, ctx: ctx, protocol: protocol, stream: kind == "text/event-stream"}
	return response
}
func (r *usageReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.total += n
	if r.total > maxStreamBytes {
		r.overflow = true
	}
	if !r.overflow {
		r.buffer = append(r.buffer, p[:n]...)
		if r.stream {
			for {
				end := bytes.IndexByte(r.buffer, '\n')
				if end < 0 {
					break
				}
				line := bytes.TrimSpace(r.buffer[:end])
				if bytes.HasPrefix(line, []byte("data:")) {
					usage := ParseUsage(bytes.TrimSpace(line[5:]), r.protocol, true)
					if usage.InputKnown || usage.OutputKnown {
						if e := r.trace.Report(r.ctx, usage); e != nil {
							return 0, e
						}
					}
				}
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
		line := bytes.TrimSpace(r.buffer)
		if bytes.HasPrefix(line, []byte("data:")) {
			usage := ParseUsage(bytes.TrimSpace(line[5:]), r.protocol, true)
			if usage.InputKnown || usage.OutputKnown {
				if e := r.trace.Report(r.ctx, usage); e != nil {
					return 0, e
				}
			}
		}
		clear(r.buffer)
		r.buffer = nil
	}
	if err == io.EOF && !r.stream && !r.overflow && !r.done {
		r.done = true
		usage := ParseUsage(r.buffer, r.protocol, false)
		if usage.InputKnown || usage.OutputKnown {
			if e := r.trace.Report(r.ctx, usage); e != nil {
				return 0, e
			}
		}
		clear(r.buffer)
		r.buffer = nil
	}
	return n, err
}
