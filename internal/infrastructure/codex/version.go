package codex

import (
	"context"
	"os/exec"
	"strings"
)

// InstalledVersion 查询当前可执行核心，不读取默认授权或修改客户端。
func InstalledVersion(ctx context.Context) (string, error) {
	path, err := exec.LookPath("codex")
	if err != nil {
		return "", ErrUnavailable
	}
	command := exec.CommandContext(ctx, path, "--version")
	hideWindow(command)
	body, err := command.Output()
	if err != nil || len(body) > 256 {
		return "", ErrUnavailable
	}
	version := strings.TrimSpace(string(body))
	if !strings.HasPrefix(version, "codex-cli ") {
		return "", ErrProtocol
	}
	return version, nil
}
