package codex

import "net/url"

const legacyBrowserOriginator = "codex_vscode"

// 参考旧项目已可用的 Codex 浏览器流程，兼容账号选择页的流程来源。
// 仅修改旧授权端点的公共来源标记；PKCE、state、回调和客户端 ID 均由官方组件生成。
// app-server 的 clientInfo 仍为 OmniProxy，不改变实际请求的应用归属。
func compatibleBrowserURL(value string) (string, error) {
	if !ValidAuthorizationURL(value) {
		return "", ErrProtocol
	}
	parsed, err := url.Parse(value)
	if err != nil {
		return "", ErrProtocol
	}
	if parsed.Host != "auth.openai.com" || parsed.Path != "/oauth/authorize" {
		return value, nil
	}
	query, err := url.ParseQuery(parsed.RawQuery)
	if err != nil {
		return "", ErrProtocol
	}
	query.Set("originator", legacyBrowserOriginator)
	parsed.RawQuery = query.Encode()
	result := parsed.String()
	if !ValidAuthorizationURL(result) {
		return "", ErrProtocol
	}
	return result, nil
}
