package proxy

import (
	"encoding/json"
	"strings"
)

// ReasoningEffort 读取请求正文里客户端选择的推理档位。
//
// 三种协议各自用不同字段表达同一件事：Messages 用 output_config.effort，
// Responses 用 reasoning.effort，Chat 用 reasoning_effort。字段缺失、类型不符或
// 取值不可用时返回空串：未指定不是零档位，不能让上游默认值冒充实测结果。
func ReasoningEffort(protocol string, body []byte) string {
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil || fields == nil {
		return ""
	}
	switch protocol {
	case "messages":
		return NormalizeEffort(nestedEffort(fields["output_config"]))
	case "responses":
		return NormalizeEffort(nestedEffort(fields["reasoning"]))
	default:
		var level string
		if json.Unmarshal(fields["reasoning_effort"], &level) != nil {
			return ""
		}
		return NormalizeEffort(level)
	}
}

func nestedEffort(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var container map[string]json.RawMessage
	if json.Unmarshal(raw, &container) != nil || container == nil {
		return ""
	}
	var level string
	if json.Unmarshal(container["effort"], &level) != nil {
		return ""
	}
	return level
}

// NormalizeEffort 只保留能安全入库的档位取值：去空白、限长 32、禁止控制字符。
// 取值本身来自客户端正文，不能拿它构造日志或 SQL 之外的内容。
func NormalizeEffort(level string) string {
	level = strings.TrimSpace(level)
	if level == "" || len(level) > 32 {
		return ""
	}
	for index := 0; index < len(level); index++ {
		c := level[index]
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			return ""
		}
	}
	return level
}
