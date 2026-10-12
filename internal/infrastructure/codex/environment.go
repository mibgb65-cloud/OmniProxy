package codex

import (
	"os"
	"strings"
)

// 授权子进程不继承启动器的客户端身份和登录端点覆盖。
// 代理和自定义 CA 仍按用户网络配置使用，不能因账号隔离而丢失。
func isolatedEnvironment(home string) []string {
	blocked := map[string]bool{
		"CODEX_HOME":                         true,
		"OPENAI_API_KEY":                     true,
		"CODEX_ACCESS_TOKEN":                 true,
		"CODEX_REFRESH_TOKEN":                true,
		"OPENAI_BASE_URL":                    true,
		"CODEX_API_KEY":                      true,
		"CODEX_INTERNAL_ORIGINATOR_OVERRIDE": true,
		"CODEX_APP_SERVER_LOGIN_CLIENT_ID":   true,
		"CODEX_APP_SERVER_LOGIN_ISSUER":      true,
	}
	var out []string
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if !blocked[strings.ToUpper(key)] {
			out = append(out, item)
		}
	}
	return append(out, "CODEX_HOME="+home)
}
