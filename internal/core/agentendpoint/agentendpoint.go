// Package agentendpoint 定义受管 Agent 在本地网关上的固定入口前缀。
//
// 前缀既写进客户端配置（Base URL 的字面量组成部分），也决定请求进入网关时
// 允许归属的受管客户端。两者必须来自同一张表，否则配置写出的地址与网关
// 接受的路径会各自漂移。前缀只做路由分组，真正可用的凭据仍由网关校验。
package agentendpoint

import (
	"fmt"
	"strings"
)

// bindings 是客户端 → 入口前缀的唯一真相；多个客户端可共用一个前缀。
var bindings = []struct {
	client string
	prefix string
}{
	{"codex", "/codex"},
	{"claude", "/claude"},
	{"claude-desktop", "/claude"},
	{"opencode", "/opencode"},
	{"kilocode", "/kilocode"},
	{"mimo-code", "/mimo-code"},
	{"qwen-code", "/qwen-code"},
	{"harness-web", "/harness-web"},
	{"harness-desktop", "/harness-desktop"},
	{"kimi-code", "/kimi-code"},
	{"pi", "/pi"},
	{"factory-droid", "/factory-droid"},
	{"goose", "/goose"},
}

// prefixClients 是前缀 → 允许的客户端集合，启动时从 bindings 派生。
var prefixClients = index()

func index() map[string][]string {
	clients := make(map[string][]string, len(bindings))
	for _, item := range bindings {
		clients[item.prefix] = append(clients[item.prefix], item.client)
	}
	return clients
}

// codexRuntimes 把 Codex 客户端自己声明的来源标识映射到可区分的运行形态。
// 三类运行形态共用同一个入口前缀，也只有它们各自的来源标识能把它们分开，
// 因此只认固定标记，不猜测未知值。
var codexRuntimes = []struct {
	client  string
	markers []string
}{
	{"codex-cli", []string{"codex_cli_rs", "codex-cli"}},
	{"codex-desktop", []string{"codex desktop", "codex_desktop", "codex-desktop"}},
	{"codex-vscode", []string{"codex_vscode", "codex-vscode"}},
}

// Runtime 返回可记入请求归属的运行形态：Codex 按来源标识细化，其余客户端原样返回。
// 来源标识按 originator 优先、User-Agent 次之的顺序匹配，因为桌面应用会沿用 CLI 的 User-Agent。
func Runtime(client, originator, userAgent string) string {
	if client != "codex" {
		return client
	}
	for _, source := range []string{originator, userAgent} {
		value := strings.ToLower(source)
		for _, runtime := range codexRuntimes {
			for _, marker := range runtime.markers {
				if strings.Contains(value, marker) {
					return runtime.client
				}
			}
		}
	}
	return client
}

// Prefix 返回客户端的固定入口前缀。
func Prefix(client string) (string, bool) {
	for _, item := range bindings {
		if item.client == client {
			return item.prefix, true
		}
	}
	return "", false
}

// Clients 返回前缀允许的受管客户端；未知前缀返回 nil。
func Clients(prefix string) []string {
	return prefixClients[prefix]
}

// BaseURL 返回客户端的本地入口根地址，不含协议路径。
// Codex 由客户端追加 /v1/responses，Claude 由客户端追加 /v1/messages。
func BaseURL(client string, port int) (string, bool) {
	prefix, ok := Prefix(client)
	if !ok {
		return "", false
	}
	return fmt.Sprintf("http://127.0.0.1:%d%s", port, prefix), true
}
