package agentinstall

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
	"time"
)

// commandExitError 只保留退出码，避免丢失包管理器的明确失败原因。
type commandExitError struct{ code uint32 }

func (e commandExitError) Error() string { return ErrInstallFailed.Error() }
func (e commandExitError) Unwrap() error { return ErrInstallFailed }

func commandExitCode(err error) uint32 {
	var console commandExitError
	if errors.As(err, &console) {
		return console.code
	}
	var process *exec.ExitError
	if errors.As(err, &process) {
		return uint32(process.ExitCode())
	}
	return 0
}

// limitBuffer 消费全部输出但最多保留 64 KiB，不让安装器输出占用无限内存。
type limitBuffer struct {
	mu        sync.Mutex
	data      []byte
	truncated bool
}

func runObservedCommand(ctx context.Context, path string, args []string, output io.Writer) error {
	command := exec.CommandContext(ctx, path, args...)
	hideWindow(command)
	command.WaitDelay = time.Second
	command.Stdout, command.Stderr = output, &limitBuffer{}
	return command.Run()
}

func (b *limitBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	remaining := (64 << 10) - len(b.data)
	if len(data) > remaining {
		b.data = append(b.data, data[:remaining]...)
		b.truncated = true
	} else {
		b.data = append(b.data, data...)
	}
	return len(data), nil
}

func runCommand(ctx context.Context, path string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, path, args...)
	hideWindow(command)
	command.WaitDelay = time.Second
	output, diagnostic := &limitBuffer{}, &limitBuffer{}
	command.Stdout, command.Stderr = output, diagnostic
	err := command.Run()
	// stderr 永不进入 UI 或日志；调用方只展示稳定错误代码。
	return output.data, err
}
