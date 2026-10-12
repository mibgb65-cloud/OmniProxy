// Package routing 提供不依赖数据库、HTTP 框架或桌面的模型匹配与账号选择规则。
package routing

import (
	"encoding/json"
	"errors"
	"strings"
)

// ErrInvalidInput 表示模型名或待改写正文不符合协议。
var ErrInvalidInput = errors.New("routing: invalid input")

// ValidModel 校验客户端模型名与别名，保持原代理的 128 字节和 ASCII 字符约束。
func ValidModel(model string) bool {
	if len(model) == 0 || len(model) > 128 {
		return false
	}
	for _, char := range model {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("-_.:/", char)) {
			return false
		}
	}
	return true
}

// RewriteModel 只改写顶层 model；其余提供商参数保持 JSON 语义不变。
func RewriteModel(body []byte, model string) ([]byte, error) {
	if !ValidModel(model) {
		return nil, ErrInvalidInput
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil || fields == nil {
		return nil, ErrInvalidInput
	}
	encoded, err := json.Marshal(model)
	if err != nil {
		return nil, ErrInvalidInput
	}
	fields["model"] = encoded
	return json.Marshal(fields)
}
