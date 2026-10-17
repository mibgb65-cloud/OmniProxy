//go:build windows

package agentinstall

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/windows/registry"
)

func (b *Backend) inspectDesktop(ctx context.Context, product Product) Detection {
	if detected, found := registryDesktop(ctx, product); found {
		return detected
	}
	if len(product.AppxNames) == 0 {
		return Detection{State: "not_installed"}
	}
	path := filepath.Join(os.Getenv("SystemRoot"), "System32", "WindowsPowerShell", "v1.0", "powershell.exe")
	// AppxNames 只来自固定目录，逐个精确查询，避免同名或通配符软件被误判。
	script := "$ErrorActionPreference='Stop'; [Console]::OutputEncoding=New-Object System.Text.UTF8Encoding($false); $items=@(foreach ($name in @('" + strings.Join(product.AppxNames, "','") + "')) { Get-AppxPackage -Name $name | Select-Object Name,@{n='Version';e={$_.Version.ToString()}},InstallLocation,@{n='Status';e={$_.Status.ToString()}} }); ConvertTo-Json -InputObject $items -Compress"
	output, err := b.run(ctx, path, "-NoProfile", "-NonInteractive", "-Command", script)
	if err != nil {
		return unknown("detection_failed")
	}
	return inspectDesktopPackages(output, product.AppxNames)
}

func registryDesktop(ctx context.Context, product Product) (Detection, bool) {
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		for _, base := range []string{`Software\Microsoft\Windows\CurrentVersion\Uninstall`, `Software\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall`} {
			key, err := registry.OpenKey(root, base, registry.READ)
			if err != nil {
				if !errors.Is(err, registry.ErrNotExist) {
					return unknown("detection_failed"), true
				}
				continue
			}
			names, err := key.ReadSubKeyNames(4097)
			closeErr := key.Close()
			if err != nil && !errors.Is(err, io.EOF) || closeErr != nil || len(names) > 4096 {
				return unknown("detection_failed"), true
			}
			for _, name := range names {
				if ctx.Err() != nil {
					return unknown("detection_failed"), true
				}
				entry, err := registry.OpenKey(root, base+`\`+name, registry.QUERY_VALUE)
				if err != nil {
					if !errors.Is(err, registry.ErrNotExist) {
						return unknown("detection_failed"), true
					}
					continue
				}
				display, _, _ := entry.GetStringValue("DisplayName")
				publisher, _, _ := entry.GetStringValue("Publisher")
				version, _, _ := entry.GetStringValue("DisplayVersion")
				location, _, _ := entry.GetStringValue("InstallLocation")
				icon, _, _ := entry.GetStringValue("DisplayIcon")
				if err := entry.Close(); err != nil {
					return unknown("detection_failed"), true
				}
				matches := product.ID == "codex-desktop" && (display == "Codex" || display == "ChatGPT") && strings.HasPrefix(publisher, "OpenAI") || product.ID == "claude-desktop" && display == "Claude" && strings.HasPrefix(publisher, "Anthropic") || product.ID == "opencode-desktop" && (display == "OpenCode" || display == "opencode") && (strings.HasPrefix(publisher, "Anomaly") || strings.HasPrefix(publisher, "SST")) || product.ID == "harness-desktop" && display == "DeepSeek Harness" && strings.HasPrefix(publisher, "DeepSeek")
				codingLocation, codingMatch := codingDesktopLocation(product.ID, display, publisher, version, location, icon)
				if codingMatch {
					location = codingLocation
				}
				if matches || codingMatch {
					if !validVersion(version) || len(location) > 4096 {
						return unknown("detection_failed"), true
					}
					return Detection{State: "installed", Version: version, Path: location}, true
				}
			}
		}
	}
	return Detection{}, false
}
