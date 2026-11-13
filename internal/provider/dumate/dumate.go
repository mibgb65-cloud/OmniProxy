// Package dumate 接入百度搭子 DuMate 个人版的登录态与 OpenAI 兼容推理网关。
//
// 登录态来自搭子桌面端（只读）或用户粘贴的浏览器请求 Cookie；Cookie 只在
// 受控回调内使用，换取 gateway token 后以 Bearer 访问固定 HTTPS 网关。
package dumate

import (
	"time"

	catalog "omniproxy/internal/core/modelcatalog"
)

const (
	// Kind 是内置服务商的稳定标识。
	Kind = "dumate"
	// Name 是界面展示名称。
	Name = "百度搭子 DuMate"
	// DefaultBaseURL 是模型网关基址；上游没有 /models 端点。
	DefaultBaseURL = "https://dumate-svc.baidu.com/gateway/apis/v1"
	// ConsoleBase 是百度智能云控制台；登录校验、token 换取、积分与签到都在这里。
	ConsoleBase = "https://console.bce.baidu.com"
	// LoginURL 是浏览器登录入口，仅用于向用户展示。
	LoginURL = "https://login.bce.baidu.com/?from=oa_dumate&redirect=https%3A%2F%2Fconsole.bce.baidu.com%2F"
	// UserAgent 复用桌面端标识，避免控制台把请求当成未知客户端。
	UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) DuMate/1.0.82 Chrome/130.0.0.0 Electron/33.0.0 Safari/537.36"

	// ConsoleTimeout 约束单次控制台请求；签到总预算由应用层控制。
	ConsoleTimeout = 20 * time.Second
	// GatewayResponseHeaderTimeout 允许推理模型在首个分片前思考较长时间。
	GatewayResponseHeaderTimeout = 3 * time.Minute
	// TLSHandshakeTimeout 与其它适配器保持一致。
	TLShandshakeTimeout = 10 * time.Second
	// TokenTTL 是上游未给出有效过期时间时的回退值；参考实现同样按 1 小时缓存。
	TokenTTL = time.Hour
	// TokenRefreshMargin 是提前刷新的余量。
	TokenRefreshMargin = 5 * time.Minute
)

const (
	// ModelText 是桌面端使用的智能路由别名，上游会隐藏实际底层模型。
	ModelText = "model-text"
	// ModelAutoL0 是参考实现保留的第二条回退别名。
	ModelAutoL0 = "dm-auto-model/text.L0"
)

// Models 返回网关公布的模型别名；不代表底层模型或容量已被验证。
func Models() []string { return []string{ModelText, ModelAutoL0} }

// DisplayModels 返回提供商卡片使用的展示名称，请求 ID 仍由 Models 提供。
func DisplayModels() []string {
	models := Models()
	for i, id := range models {
		models[i] = ModelName(id)
	}
	return models
}

// SupportsModel 只接受已经确认存在的两条别名，不做模糊匹配。
func SupportsModel(model string) bool {
	for _, id := range Models() {
		if id == model {
			return true
		}
	}
	return false
}

// ModelName 返回固定短名称；上游没有可确认的底层模型名。
func ModelName(model string) string {
	switch model {
	case ModelText:
		return "DuMate Auto（智能路由）"
	case ModelAutoL0:
		return "DuMate Auto L0"
	default:
		return model
	}
}

// ModelMetadata 只发布可以确认的文本能力；上下文与输出上限未经真机验证，
// 因此不写入容量，避免客户端据此截断或超发请求。
func ModelMetadata(model string) catalog.Metadata {
	if !SupportsModel(model) {
		return catalog.Metadata{}
	}
	return catalog.Metadata{InputModalities: []string{"text"}}
}
