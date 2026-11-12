package river

import (
	"encoding/json"
	"omniproxy/internal/core/proxy"
)

// validateTools 只校验工具的形状与名字，不限制工具数量。
//
// River 官方 Limits 仅公布 50 MiB 请求体、图片格式与超时，Tools 页面也没有给出函数工具
// 的数量上限；2026-11-12 用生产 Key 直接向 POST /v1/chat/completions 发送 176 个函数工具
// 实测返回 200。DeepSeek Harness 这类把内置工具与多个 MCP 工具合并发送的客户端常超过
// 128 个，此前会被本地计数拦成笼统的 400 invalid_request 且请求根本没有到达上游。
// 数量只受请求体大小、JSON 深度与上游最终校验约束；本层不截断、不丢弃任何工具。
func validateTools(raw json.RawMessage, protocol string) error {
	if len(raw) == 0 {
		return nil
	}
	var tools []struct {
		Type     string `json:"type"`
		Name     string `json:"name"`
		Function *struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if json.Unmarshal(raw, &tools) != nil || string(raw) == "null" {
		return proxy.Failure("invalid_request", 400)
	}
	for _, tool := range tools {
		if tool.Type != "function" {
			return proxy.Failure("unsupported_content", 400)
		}
		name := tool.Name
		if protocol == "chat" {
			if tool.Function == nil {
				return proxy.Failure("invalid_request", 400)
			}
			name = tool.Function.Name
		}
		if name == "" || len(name) > 128 {
			return proxy.Failure("invalid_request", 400)
		}
	}
	return nil
}
