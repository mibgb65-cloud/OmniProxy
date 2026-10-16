package agentconfig

import (
	"os"
	"path/filepath"
)

const DesktopProfileID = "00000000-0000-4000-8000-000000309006"

// DefaultPaths 读取客户端的现有数据根目录，不创建另一份 CLI 会话空间。
func DefaultPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, failure("invalid_path")
	}
	codex := os.Getenv("CODEX_HOME")
	if codex == "" {
		codex = filepath.Join(home, ".codex")
	}
	claude := os.Getenv("CLAUDE_CONFIG_DIR")
	if claude == "" {
		claude = filepath.Join(home, ".claude")
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return Paths{}, failure("invalid_path")
	}
	profile, normal, thirdParty, err := desktopClientPaths(cache)
	if err != nil {
		return Paths{}, err
	}
	return Paths{
		Codex: filepath.Join(codex, "config.toml"), Claude: filepath.Join(claude, "settings.json"),
		ClaudeDesktop: profile, ClaudeDesktopNormal: normal, ClaudeDesktopThirdParty: thirdParty,
		OpenCode:           openCodePath(home),
		HarnessWeb:         filepath.Join(harnessHome(home), "profiles", "web", "cordis.patch.yml"),
		HarnessDesktop:     filepath.Join(harnessHome(home), "profiles", "desktop", "cordis.patch.yml"),
		HarnessCredentials: filepath.Join(harnessHome(home), ".credentials.yaml"),
	}, nil
}

func harnessHome(home string) string {
	if value := os.Getenv("DSH_HOME"); value != "" {
		return value
	}
	return filepath.Join(home, ".dsh")
}

func openCodePath(home string) string {
	if value := os.Getenv("OPENCODE_CONFIG"); value != "" {
		return value
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(home, ".config")
	}
	base = filepath.Join(base, "opencode")
	if custom := os.Getenv("OPENCODE_CONFIG_DIR"); custom != "" {
		base = custom
	}
	jsonc := filepath.Join(base, "opencode.jsonc")
	if _, err := os.Lstat(jsonc); !os.IsNotExist(err) {
		return jsonc
	}
	return filepath.Join(base, "opencode.json")
}

func desktopClientPaths(cache string) (profile, normal, thirdParty string, err error) {
	normalRoot, thirdPartyRoot := filepath.Join(cache, "Claude"), filepath.Join(cache, "Claude-3p")
	// Claude 的 MSIX 清单将 Claude-3p 排除出文件系统虚拟化，与普通安装共用此入口。
	// 包缓存即使残留配置库也不能作为生效依据，否则一键配置会写入客户端不读取的位置。
	return filepath.Join(thirdPartyRoot, "configLibrary", DesktopProfileID+".json"), filepath.Join(normalRoot, "claude_desktop_config.json"), filepath.Join(thirdPartyRoot, "claude_desktop_config.json"), nil
}
