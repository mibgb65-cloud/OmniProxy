package agentinstall

import (
	"context"
	"errors"
	"os/exec"
	"runtime"
)

// ErrUpdateInstallFailed 不暴露安装器输出。
var ErrUpdateInstallFailed = errors.New("agentinstall: update_failed")

// Update 仅显式升级固定 Claude 商店包，保留系统完整性校验和安装策略。
func (b *Backend) Update(ctx context.Context, id string, report func(Progress)) error {
	product, err := Find(id)
	if err != nil {
		return err
	}
	if product.ID != "claude-desktop" || runtime.GOOS != "windows" {
		return ErrUpdateUnsupported
	}
	path, err := exec.LookPath("winget.exe")
	if err != nil {
		return ErrUnavailable
	}
	// 商店目录不提供可比较的版本；只允许对此已检测到更新的精确目标查询实际更新。
	args := []string{"upgrade", "--id", product.PackageID, "--exact", "--source", product.Source, "--include-unknown", "--silent", "--disable-interactivity", "--accept-package-agreements", "--accept-source-agreements"}
	if b.runInstall != nil {
		err = b.runInstall(ctx, path, args, report)
	} else {
		_, err = b.run(ctx, path, args...)
	}
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrUpdateInstallFailed
	}
	return nil
}

// DesktopVersionAtLeast 核对实际 MSIX 版本达到目标；未知版本永远不能作为更新成功。
func DesktopVersionAtLeast(current, expected string) bool {
	installed, ok := desktopVersion(current)
	if !ok {
		return false
	}
	target, ok := desktopVersion(expected)
	if !ok {
		return false
	}
	for index := range installed {
		if installed[index] != target[index] {
			return installed[index] > target[index]
		}
	}
	return true
}
