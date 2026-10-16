package agentinstall

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Backend 不持有凭据，也不读取客户端配置和会话文件。
type Backend struct {
	run        func(context.Context, string, ...string) ([]byte, error)
	runInstall func(context.Context, string, []string, func(Progress)) error
}

// New 创建只在用户明确安装时执行包管理器的运行适配器。
func New() *Backend { return &Backend{run: runCommand, runInstall: runInstallCommand} }

// Available 检测系统安装器是否存在，不触发联网或安装。
func (b *Backend) Available() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	_, err := exec.LookPath("winget.exe")
	return err == nil
}

// Inspect 仅检测可执行程序、卸载注册表和当前用户的 MSIX 包。
func (b *Backend) Inspect(ctx context.Context, id string) Detection {
	product, err := Find(id)
	if err != nil {
		return unknown("invalid_product")
	}
	if runtime.GOOS != "windows" {
		return unknown("unsupported_platform")
	}
	if product.Command == "" {
		return b.inspectDesktop(ctx, product)
	}
	return b.inspectCLI(ctx, product)
}

// Install 保留现有版本，由 WinGet 校验包完整性；不传入升级或校验绕过参数。
func (b *Backend) Install(ctx context.Context, id string, report func(Progress)) error {
	product, err := Find(id)
	if err != nil {
		return err
	}
	if product.PackageID == "" {
		return ErrUnavailable
	}
	if runtime.GOOS != "windows" {
		return ErrUnavailable
	}
	path, err := exec.LookPath("winget.exe")
	if err != nil {
		return ErrUnavailable
	}
	args := installArguments(product)
	var installErr error
	if b.runInstall != nil {
		installErr = b.runInstall(ctx, path, args, report)
	} else {
		_, installErr = b.run(ctx, path, args...)
	}
	if installErr != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrInstallFailed
	}
	// 包管理器可能更新用户 PATH；当前进程也加入已检测的 CLI 目录，接入无需重启。
	if product.Command != "" {
		if executable := cliPath(product); executable != "" {
			directory := filepath.Dir(executable)
			if err := os.Setenv("PATH", directory+string(os.PathListSeparator)+os.Getenv("PATH")); err != nil {
				return ErrInstallFailed
			}
		}
	}
	return nil
}

func installArguments(product Product) []string {
	args := []string{"install", "--id", product.PackageID, "--exact", "--source", product.Source, "--no-upgrade", "--silent", "--disable-interactivity", "--accept-package-agreements", "--accept-source-agreements"}
	if product.Source == "winget" {
		args = append(args, "--scope", "user")
	}
	return args
}

var versionPattern = regexp.MustCompile(`^[0-9][0-9A-Za-z.+-]{0,127}$`)

func unknown(code string) Detection  { return Detection{State: "unknown", ErrorCode: code} }
func validVersion(value string) bool { return versionPattern.MatchString(value) }

func (b *Backend) inspectCLI(ctx context.Context, product Product) Detection {
	path := cliPath(product)
	if path == "" {
		return Detection{State: "not_installed"}
	}
	if strings.EqualFold(filepath.Ext(path), ".cmd") || strings.EqualFold(filepath.Ext(path), ".bat") {
		// npm 启动脚本可确认安装；不通过 shell 执行未知目录中的脚本来读取版本。
		return Detection{State: "installed", Path: path}
	}
	output, err := b.run(ctx, path, "--version")
	if err != nil || len(output) > 256 {
		return Detection{State: "unknown", Path: path, ErrorCode: "version_unavailable"}
	}
	text := strings.TrimSpace(string(output))
	version := strings.TrimPrefix(text, "codex-cli ")
	if product.Command == "claude" {
		fields := strings.Fields(text)
		if len(fields) == 0 {
			return unknown("version_unavailable")
		}
		version = fields[0]
	}
	if !validVersion(version) || product.Command == "codex" && !strings.HasPrefix(text, "codex-cli ") {
		return unknown("version_unavailable")
	}
	return Detection{State: "installed", Version: version, Path: path}
}

func cliPath(product Product) string {
	if path, err := exec.LookPath(product.Command); err == nil && len(path) <= 4096 && filepath.IsAbs(path) {
		return path
	}
	local := os.Getenv("LOCALAPPDATA")
	home, _ := os.UserHomeDir()
	candidates := []string{}
	if local != "" {
		candidates = append(candidates, filepath.Join(local, "Microsoft", "WinGet", "Links", product.Command+".exe"))
		if product.Command == "codex" {
			candidates = append(candidates, filepath.Join(local, "Programs", "OpenAI", "Codex", "bin", "codex.exe"))
		}
	}
	if roaming := os.Getenv("APPDATA"); roaming != "" {
		candidates = append(candidates, filepath.Join(roaming, "npm", product.Command+".cmd"))
	}
	if node := os.Getenv("NVM_SYMLINK"); node != "" {
		candidates = append(candidates, filepath.Join(node, product.Command+".cmd"))
	}
	if home != "" {
		candidates = append(candidates, filepath.Join(home, ".local", "bin", product.Command+".exe"))
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
			return path
		}
	}
	return ""
}
