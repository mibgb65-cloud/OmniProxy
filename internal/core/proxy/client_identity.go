package proxy

import (
	"net/http"

	"omniproxy/internal/core/agentendpoint"
)

// ValidClientUserAgent 限制非敏感身份头的长度和字符，禁止注入其他 HTTP 头。
func ValidClientUserAgent(value string) bool {
	if len(value) > 1024 {
		return false
	}
	for i := 0; i < len(value); i++ {
		if value[i] < 32 || value[i] > 126 {
			return false
		}
	}
	return true
}

// recordedClient 把接入 Key 得到的归属细化成可显示的运行形态。
// 只读取来源标识并映射成固定标识，原始头部不入库。
func recordedClient(client string, header http.Header) string {
	return agentendpoint.Runtime(client, header.Get("originator"), header.Get("User-Agent"))
}
