package stepfun

import (
	"encoding/json"
	"io"
	"omniproxy/internal/core/proxy"
)

// 只有已公开的资源标识参与错误分类，原始错误消息不进入日志或客户端。
func resourceFailure(status int, body io.Reader) error {
	if status != 402 && status != 429 {
		return nil
	}
	raw, err := io.ReadAll(io.LimitReader(body, 65537))
	defer clear(raw)
	var value struct {
		Error struct {
			Code string `json:"code"`
			Type string `json:"type"`
		} `json:"error"`
	}
	if err != nil || len(raw) > 65536 || json.Unmarshal(raw, &value) != nil {
		return nil
	}
	if value.Error.Code == "" {
		value.Error.Code = value.Error.Type
	}
	if status == 402 && value.Error.Code == "insufficient_credit" {
		return proxy.Failure("upstream_insufficient_credit", 402)
	}
	if status == 429 {
		switch value.Error.Code {
		case "project_credit_limit_exceeded", "member_project_credit_limit_exceeded":
			return proxy.Failure("upstream_"+value.Error.Code, 429)
		}
	}
	return nil
}
