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
	}, nil
}

func desktopClientPaths(cache string) (profile, normal, thirdParty string, err error) {
	normalRoot, thirdPartyRoot := filepath.Join(cache, "Claude"), filepath.Join(cache, "Claude-3p")
	thirdPartyRoot, err = desktopThirdPartyRoot(cache, thirdPartyRoot)
	if err != nil {
		return "", "", "", err
	}
	// 商店版的 Local 虚拟配置库可能遮蔽共享目录；Roaming 官方聊天设置仍不作为 3P 配置入口。
	return filepath.Join(thirdPartyRoot, "configLibrary", DesktopProfileID+".json"), filepath.Join(normalRoot, "claude_desktop_config.json"), filepath.Join(thirdPartyRoot, "claude_desktop_config.json"), nil
}
