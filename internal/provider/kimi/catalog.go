// Package kimi 提供 Kimi 普通 API Key 的原生文本协议，不转换思考及工具历史。
package kimi

import (
	"errors"
	catalog "omniproxy/internal/core/modelcatalog"
)

// Kind 沿用已有内置提供商标识。
const Kind = "kimi"

// DefaultBaseURL 与用户引用的国内开放平台文档一致。
const DefaultBaseURL = "https://api.moonshot.cn/v1"

// Options 保存公开平台选择与并发配置版本，不包含 Key。
type Options struct {
	Region  string `json:"region"`
	Version int64  `json:"version"`
}

// Valid 只接受两个已确认的官方平台。
func (o Options) Valid() bool { return (o.Region == "cn" || o.Region == "global") && o.Version >= 0 }

// BaseURL 固定解析平台，错误时不尝试另一个平台的认证。
func (o Options) BaseURL() (string, error) {
	if !o.Valid() {
		return "", errors.New("kimi: invalid_configuration")
	}
	if o.Region == "global" {
		return "https://api.moonshot.ai/v1", nil
	}
	return DefaultBaseURL, nil
}

// Models 返回当前四款官方模型，排除已下线模型。
func Models() []string {
	return []string{"kimi-k3", "kimi-k2.7-code", "kimi-k2.7-code-highspeed", "kimi-k2.6"}
}

// SupportsModel 限制对话路由到已核实模型。
func SupportsModel(id string) bool {
	for _, model := range Models() {
		if model == id {
			return true
		}
	}
	return false
}

// Supports 按协议开放能力；Responses 目前官方明确只支持 K3。
func Supports(id, protocol string) bool {
	if !SupportsModel(id) {
		return false
	}
	return protocol == "chat" || protocol == "messages" && id != "kimi-k2.7-code-highspeed" || protocol == "responses" && id == "kimi-k3"
}

// ModelMetadata 的标称窗口与真实长输入验证保持区别。
func ModelMetadata(id string) catalog.Metadata {
	if !SupportsModel(id) {
		return catalog.Metadata{}
	}
	window, tools := int64(262144), true
	m := catalog.Metadata{ContextWindow: &window, ToolUse: &tools, InputModalities: []string{"text"}}
	if id == "kimi-k3" {
		window = 1048576
		output := int64(1048576)
		m.OutputTokenLimit = &output
		m.ReasoningEfforts = []catalog.Effort{{Value: "low", Description: "低强度推理"}, {Value: "high", Description: "高强度推理"}, {Value: "max", Description: "最高强度推理"}}
		m.DefaultEffort = "max"
	} else if id == "kimi-k2.6" {
		m.Description = "默认思考，可通过原生 thinking 参数关闭；不提供推理强度档位"
	} else {
		m.Description = "始终思考并保留历史推理；不提供推理强度档位"
	}
	return m
}
