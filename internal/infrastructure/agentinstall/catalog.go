// Package agentinstall 检测本机客户端，并通过 Windows 包管理器安装固定的软件包。
package agentinstall

import "errors"

var (
	ErrInvalidProduct  = errors.New("agentinstall: invalid_product")
	ErrUnavailable     = errors.New("agentinstall: installer_unavailable")
	ErrInstallFailed   = errors.New("agentinstall: install_failed")
	ErrDetectionFailed = errors.New("agentinstall: detection_failed")
)

// Product 固定安装渠道；前端不能指定包名、命令、下载地址或安装路径。
type Product struct {
	ID, PackageID, Source, Command, OfficialURL string
	AppxNames                                   []string
}

// Detection 区分未安装和检测失败，避免将异常误报为可重新安装。
type Detection struct {
	State     string `json:"state"`
	Version   string `json:"version"`
	Path      string `json:"path"`
	ErrorCode string `json:"errorCode"`
}

// Products 返回本版支持的固定安装目标。
func Products() []Product {
	return []Product{
		{ID: "codex-cli", PackageID: "OpenAI.Codex", Source: "winget", Command: "codex", OfficialURL: "https://learn.chatgpt.com/docs/codex/cli"},
		{ID: "codex-desktop", PackageID: "9PLM9XGG6VKS", Source: "msstore", AppxNames: []string{"OpenAI.Codex"}, OfficialURL: "https://learn.chatgpt.com/docs/windows/windows-app"},
		{ID: "claude-code", PackageID: "Anthropic.ClaudeCode", Source: "winget", Command: "claude", OfficialURL: "https://code.claude.com/docs/en/setup"},
		// 商店新版改用 AnthropicPBC.Claude，旧版 Claude 仍需识别，避免安装成功后误报。
		{ID: "claude-desktop", PackageID: "9P6K58THS811", Source: "msstore", AppxNames: []string{"AnthropicPBC.Claude", "Claude"}, OfficialURL: "https://code.claude.com/docs/en/desktop"},
		{ID: "opencode-cli", Command: "opencode", OfficialURL: "https://opencode.ai/docs/"},
		{ID: "opencode-desktop", OfficialURL: "https://opencode.ai/download"},
		{ID: "harness-cli", Command: "dsh", OfficialURL: "https://github.com/deepseek-ai/deepseek-harness"},
		{ID: "harness-desktop", OfficialURL: "https://www.deepseek.com/en/harness/"},
		{ID: "kilocode-cli", Command: "kilo", OfficialURL: "https://kilo.ai/docs/code-with-ai/platforms/cli"},
		{ID: "kilocode-desktop", OfficialURL: "https://kilo.ai/docs/desktop/overview"},
		{ID: "qwen-code-cli", Command: "qwen", OfficialURL: "https://github.com/QwenLM/qwen-code"},
		{ID: "qwen-code-desktop", OfficialURL: "https://github.com/QwenLM/qwen-code/releases/tag/desktop-latest"},
		{ID: "mimo-code-cli", Command: "mimo", OfficialURL: "https://github.com/XiaomiMiMo/MiMo-Code"},
		{ID: "mimo-desktop-cn", OfficialURL: "https://mimo.xiaomimimo.com/desktop/invite/"},
		{ID: "mimo-desktop-global", OfficialURL: "https://mimo-ai.xiaomimimo.com/desktop/invite/"},
		{ID: "kimi-code-cli", Command: "kimi", OfficialURL: "https://www.kimi.com/code/docs/en/kimi-code-cli/guides/getting-started"},
		{ID: "pi-cli", Command: "pi", OfficialURL: "https://pi.dev/"},
		{ID: "factory-droid-cli", Command: "droid", OfficialURL: "https://docs.factory.com/droid-cli/quickstart"},
		{ID: "goose-cli", Command: "goose", OfficialURL: "https://goose-docs.ai/docs/getting-started/installation/"},
		{ID: "goose-desktop", OfficialURL: "https://goose-docs.ai/docs/getting-started/installation/"},
	}
}

// Find 校验完整 ID，不接受命令片段或任意软件包。
func Find(id string) (Product, error) {
	if len(id) > 32 {
		return Product{}, ErrInvalidProduct
	}
	for _, product := range Products() {
		if product.ID == id {
			return product, nil
		}
	}
	return Product{}, ErrInvalidProduct
}
