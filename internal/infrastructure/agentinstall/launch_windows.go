//go:build windows

package agentinstall

import (
	"context"
	"errors"
	"golang.org/x/sys/windows"
	"os"
	"path/filepath"
)

// Launch 仅打开固定客户端，不在参数中传递任务、工作区、配置内容或凭据。
func Launch(ctx context.Context, id string) error {
	product, err := Find(id)
	if err != nil {
		return err
	}
	arguments := ""
	var names []string
	switch id {
	case "harness-cli":
		arguments = "web"
	case "opencode-cli":
	case "opencode-desktop":
		names = []string{"OpenCode.exe", "opencode.exe"}
	case "harness-desktop":
		names = []string{"DeepSeek Harness.exe"}
	default:
		return errors.New("agentinstall: launch_failed")
	}
	path := ""
	if product.Command != "" {
		path = cliPath(product)
	} else {
		detection := New().Inspect(ctx, id)
		if detection.State != "installed" || !filepath.IsAbs(detection.Path) {
			return errors.New("agentinstall: launch_failed")
		}
		for _, name := range names {
			candidate := filepath.Join(detection.Path, name)
			if info, err := os.Stat(candidate); err == nil && info.Mode().IsRegular() {
				path = candidate
				break
			}
		}
	}
	if !filepath.IsAbs(path) || len(path) > 4096 || ctx.Err() != nil {
		return errors.New("agentinstall: launch_failed")
	}
	file, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return errors.New("agentinstall: launch_failed")
	}
	params, err := windows.UTF16PtrFromString(arguments)
	if err != nil {
		return errors.New("agentinstall: launch_failed")
	}
	if windows.ShellExecute(0, nil, file, params, nil, windows.SW_SHOWNORMAL) != nil {
		return errors.New("agentinstall: launch_failed")
	}
	return nil
}
