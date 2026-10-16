package antigravity

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"io"
	"strings"

	"omniproxy/internal/core/proxy"
)

// Forward 只消费当前订阅基础额度，不携带付费积分类型或跨通道重试。
func (c *Client) Forward(ctx context.Context, cred Credentials, request proxy.Request, consume func(proxy.Response) error) error {
	body, err := chatPayload(request, cred)
	if err != nil {
		return err
	}
	defer clear(body)
	endpoint := inferenceURL + "/v1internal:generateContent"
	if request.Stream {
		endpoint = inferenceURL + "/v1internal:streamGenerateContent?alt=sse"
	}
	resp, err := c.request(ctx, endpoint, body, cred.AccessToken, "application/json")
	if err != nil {
		return ProxyError(err)
	}
	defer resp.Body.Close()
	state := responseState{model: request.Model, id: "chatcmpl-" + rand.Text()}
	if request.Stream {
		if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
			return proxy.Failure("invalid_upstream_response", 502)
		}
		reader := &chatStream{scanner: bufio.NewScanner(resp.Body), source: resp.Body, state: &state}
		reader.scanner.Buffer(make([]byte, 4096), 2<<20)
		return consume(proxy.Response{StatusCode: 200, ContentType: "text/event-stream", Body: reader})
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponse+1))
	defer clear(data)
	if err != nil || len(data) > maxResponse {
		return proxy.Failure("invalid_upstream_response", 502)
	}
	if _, err = state.accept(data); err != nil {
		return err
	}
	value, err := state.completed(false)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return proxy.Failure("invalid_upstream_response", 502)
	}
	return consume(proxy.Response{StatusCode: 200, ContentType: "application/json", Body: io.NopCloser(bytes.NewReader(encoded))})
}

// 同步 Reader 随下游读取上游，不启动独立泵协程；关闭下游即可取消网络读取。
type chatStream struct {
	scanner *bufio.Scanner
	source  io.Closer
	state   *responseState
	pending []byte
	done    bool
	err     error
}

func (r *chatStream) Read(buffer []byte) (int, error) {
	if len(buffer) == 0 {
		return 0, nil
	}
	for len(r.pending) == 0 && !r.done {
		if !r.scanner.Scan() {
			r.done = true
			if r.scanner.Err() != nil {
				r.err = proxy.Failure("invalid_upstream_response", 502)
				break
			}
			value, err := r.state.completed(true)
			if err != nil {
				r.err = err
				break
			}
			r.pending, err = streamFrame(value)
			if err != nil {
				r.err = err
				break
			}
			r.pending = append(r.pending, []byte("data: [DONE]\n\n")...)
			break
		}
		line := r.scanner.Bytes()
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		data := bytes.TrimSpace(line[5:])
		if bytes.Equal(data, []byte("[DONE]")) {
			continue
		}
		delta, err := r.state.accept(data)
		if err != nil {
			r.err = err
			r.done = true
			break
		}
		if len(delta) > 0 {
			r.pending, r.err = streamFrame(r.state.envelope(true, delta))
			if r.err != nil {
				r.done = true
			}
		}
	}
	if len(r.pending) > 0 {
		n := copy(buffer, r.pending)
		r.pending = r.pending[n:]
		return n, nil
	}
	if r.err != nil {
		return 0, r.err
	}
	return 0, io.EOF
}

func streamFrame(value object) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, proxy.Failure("invalid_upstream_response", 502)
	}
	return append(append([]byte("data: "), data...), []byte("\n\n")...), nil
}

func (r *chatStream) Close() error { r.done = true; clear(r.pending); return r.source.Close() }
