// Package codex 通过本机官方 CLI 的 stdio 协议管理隔离的 ChatGPT 登录。
package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"sync"
)

const maxFrameBytes = 1 << 20

var ErrUnavailable = errors.New("subscription: codex_unavailable")
var ErrProtocol = errors.New("subscription: protocol_failed")

// Client 是订阅服务需要的最小官方协议面，测试可以注入隔离实现。
type Client interface {
	Begin(context.Context, string) (Login, error)
	Wait(context.Context, string) error
	Account(context.Context, bool) (*Account, error)
	AccountID(context.Context) (string, error)
	RateLimits(context.Context) (RateLimits, error)
	ConsumeReset(context.Context, string, string) (string, error)
	Cancel(context.Context, string) error
	Logout(context.Context) error
	Close()
}

type wireResponse struct {
	result json.RawMessage
	err    error
}
type notification struct {
	method string
	params json.RawMessage
}

type rpcClient struct {
	cmd      *exec.Cmd
	input    io.WriteCloser
	mu       sync.Mutex
	writeMu  sync.Mutex
	sequence uint64
	pending  map[uint64]chan wireResponse
	notices  chan notification
	done     chan struct{}
	once     sync.Once
}

// Open 只使用独立 CODEX_HOME 与系统凭证库；不读取用户默认登录或把令牌写入文件。
func Open(ctx context.Context, home string) (Client, error) {
	if !filepath.IsAbs(home) {
		return nil, ErrProtocol
	}
	path, err := exec.LookPath("codex")
	if err != nil {
		return nil, ErrUnavailable
	}
	cmd := exec.Command(path, "app-server", "--listen", "stdio://", "-c", `cli_auth_credentials_store="keyring"`, "-c", `forced_login_method="chatgpt"`, "-c", `model_provider="openai"`)
	hideWindow(cmd)
	cmd.Env = isolatedEnvironment(home)
	cmd.Stderr = io.Discard // 官方进程输出可能包含授权信息，不送入应用日志。
	input, err := cmd.StdinPipe()
	if err != nil {
		return nil, ErrUnavailable
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		_ = input.Close()
		return nil, ErrUnavailable
	}
	if err := cmd.Start(); err != nil {
		_ = input.Close()
		_ = output.Close()
		return nil, ErrUnavailable
	}
	client := &rpcClient{cmd: cmd, input: input, pending: map[uint64]chan wireResponse{}, notices: make(chan notification, 32), done: make(chan struct{})}
	go client.read(output)
	var result json.RawMessage
	if err := client.call(ctx, "initialize", map[string]any{"clientInfo": map[string]string{"name": "omniproxy", "title": "OmniProxy", "version": "0.1.0"}, "capabilities": map[string]bool{"experimentalApi": true}}, &result); err != nil {
		client.Close()
		return nil, err
	}
	if err := client.write(map[string]any{"method": "initialized"}); err != nil {
		client.Close()
		return nil, err
	}
	return client, nil
}

func (c *rpcClient) call(ctx context.Context, method string, params any, result any) error {
	c.mu.Lock()
	c.sequence++
	id := c.sequence
	channel := make(chan wireResponse, 1)
	c.pending[id] = channel
	c.mu.Unlock()
	defer func() { c.mu.Lock(); delete(c.pending, id); c.mu.Unlock() }()
	if err := c.write(map[string]any{"id": id, "method": method, "params": params}); err != nil {
		return err
	}
	select {
	case response := <-channel:
		if response.err != nil {
			return response.err
		}
		if json.Unmarshal(response.result, result) != nil {
			return ErrProtocol
		}
		return nil
	case <-ctx.Done():
		return errors.New("subscription: operation_cancelled")
	case <-c.done:
		return ErrProtocol
	}
}

func (c *rpcClient) write(value any) error {
	data, err := json.Marshal(value)
	if err != nil || len(data) > maxFrameBytes {
		return ErrProtocol
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if _, err := c.input.Write(append(data, '\n')); err != nil {
		return ErrProtocol
	}
	return nil
}

func (c *rpcClient) read(output io.Reader) {
	defer close(c.done)
	// 帧损坏或输出超限时也结束子进程，避免请求等待到超时。
	defer func() { _ = c.cmd.Process.Kill(); _ = c.cmd.Wait() }()
	scanner := bufio.NewScanner(output)
	scanner.Buffer(make([]byte, 4096), maxFrameBytes)
	for scanner.Scan() {
		var frame struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
			Result json.RawMessage `json:"result"`
			Error  json.RawMessage `json:"error"`
		}
		if json.Unmarshal(scanner.Bytes(), &frame) != nil {
			return
		}
		if frame.Method != "" {
			if frame.Method == "account/login/completed" {
				select {
				case c.notices <- notification{frame.Method, frame.Params}:
				case <-c.done:
					return
				default:
					return
				}
			}
			continue
		}
		var id uint64
		if json.Unmarshal(frame.ID, &id) != nil {
			continue
		}
		response := wireResponse{result: frame.Result}
		if len(frame.Error) > 0 && string(frame.Error) != "null" {
			response.err = errors.New("subscription: request_failed")
		}
		c.mu.Lock()
		channel := c.pending[id]
		c.mu.Unlock()
		if channel != nil {
			select {
			case channel <- response:
			default:
			}
		}
	}
}

func (c *rpcClient) Close() {
	c.once.Do(func() { _ = c.input.Close(); _ = c.cmd.Process.Kill(); <-c.done })
}
