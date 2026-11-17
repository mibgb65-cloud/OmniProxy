package workbuddyai

import "strings"

// Build 描述一个 WorkBuddy 构建（国际版／国内版）的站点、客户端指纹与能力差异。
//
// 两个构建共用同一套插件 API（登录、续期、目录、对话、计费），差异只在站点、
// 登录平台标识、UA 中段与「是否为 Claude 模型提供原生 Anthropic Messages 端点」。
// 会话、凭据与错误码形态完全一致，因此用一份实现按 Build 参数化，而不是复制适配器。
// 事实来源：社区插件 @magpie-community/opencode-workbuddy-auth 0.2.36 的 SITES，
// 未经真实账号在 OmniProxy 验证。
type Build struct {
	// Kind 是内置服务商的稳定标识；每个构建一个 kind，账号按它隔离。
	Kind string
	// Name 是兜底展示名称（界面文案由前端 i18n 覆盖）。
	Name string
	// BaseURL 是固定基址；所有端点都相对它追加。
	BaseURL string
	// Platform 是 `/v2/plugin/auth/state` 的 platform 取值。
	Platform string
	// AppName 是 UA 第二段：官方桌面版 product.json 的 applicationName。
	AppName string
	// AppVersion 是桌面版版本；登录页的 version 参数与 X-IDE-Version 都用它。
	AppVersion string
	// CLIVersion 是桌面版内置 CLI 的版本；只出现在 UA 末尾。
	CLIVersion string
	// Messages 表示该构建为 Claude 模型提供原生 Anthropic Messages 端点
	// （`<base>/v1/messages`）；Chat 端点在这些模型上忽略推理档位。
	Messages bool
	// ImagesExceptGLM 表示 glm-* 虽声明 supportsImages，上游仍会丢弃图片
	// （国内版实测能读图，因此只有国际版为 true）。
	ImagesExceptGLM bool
}

// UserAgent 是官方桌面 CLI 的 UA 形态；不是这个形态会被 10085 拒绝。
func (b Build) UserAgent() string {
	return "WorkBuddy/" + b.AppVersion + " " + b.AppName + "/" + b.AppVersion + " CLI/" + b.CLIVersion
}

// ShortUserAgent 是登录、轮询与计费接口的短 UA 形态。
func (b Build) ShortUserAgent() string { return "WorkBuddy/" + b.AppVersion }

// MessagesNative 表示该构建把 model 放在原生 Anthropic Messages 上服务。
func (b Build) MessagesNative(model string) bool {
	return b.Messages && strings.HasPrefix(model, "claude-")
}

var (
	// International 是国际版（www.workbuddy.ai，kind workbuddy_ai）。
	International = Build{
		Kind:            Kind,
		Name:            Name,
		BaseURL:         DefaultBaseURL,
		Platform:        "workbuddy-ai",
		AppName:         "workbuddy-ai",
		AppVersion:      AppVersion,
		CLIVersion:      CLIVersion,
		ImagesExceptGLM: true,
		Messages:        false,
	}
	// China 是国内版（copilot.tencent.com，kind workbuddy）。
	China = Build{
		Kind:       ChinaKind,
		Name:       ChinaName,
		BaseURL:    ChinaBaseURL,
		Platform:   "workbuddy",
		AppName:    "WorkBuddy",
		AppVersion: AppVersion,
		CLIVersion: CLIVersion,
		Messages:   true,
	}
)

// BuildOf 按 kind 返回对应的构建；未知 kind 返回 false。
func BuildOf(kind string) (Build, bool) {
	switch kind {
	case International.Kind:
		return International, true
	case China.Kind:
		return China, true
	}
	return Build{}, false
}
