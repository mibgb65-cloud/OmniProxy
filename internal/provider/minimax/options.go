// Package minimax 提供 MiniMax 两平台、两类 Key 的原生文本 API。
package minimax

import (
	"bytes"
	"errors"
)

// Kind 沿用已有内置提供商标识。
const Kind = "minimax"

// DefaultBaseURL 是新账号默认国内入口，实际请求按账号地区解析。
const DefaultBaseURL = "https://api.minimax.cn/v1"

// Options 保存公开地区和计费选择，不根据 Key 前缀或请求错误猜测资源池。
type Options struct {
	Region      string `json:"region"`
	BillingMode string `json:"billingMode"`
	Version     int64  `json:"version"`
}

// Valid 只有明确选择四类通道后才允许发出请求。
func (o Options) Valid() bool {
	return (o.Region == "cn" || o.Region == "global") && (o.BillingMode == "payg" || o.BillingMode == "subscription") && o.Version >= 0
}

// BaseURL 两类 Key 共用协议路径，但不能互换凭证或自动回退。
func (o Options) BaseURL() (string, error) {
	if !o.Valid() {
		return "", errors.New("minimax: invalid_configuration")
	}
	if o.Region == "global" {
		return "https://api.minimax.io/v1", nil
	}
	return DefaultBaseURL, nil
}

// SamePool 比较账号声明的地区／计费身份，版本变化不改变池名称。
func (o Options) SamePool(other Options) bool {
	return o.Valid() && other.Valid() && o.Region == other.Region && o.BillingMode == other.BillingMode
}

// QuotaURL 使用文档给出的 www 管理主机，不从任意输入拼接 URL。
func (o Options) QuotaURL() (string, error) {
	if !o.Valid() || o.BillingMode != "subscription" {
		return "", errors.New("minimax: invalid_configuration")
	}
	if o.Region == "global" {
		return "https://www.minimax.io/v1/token_plan/remains", nil
	}
	return "https://www.minimax.cn/v1/token_plan/remains", nil
}

// ValidKeyForOptions 拒绝已知类型前缀的误配，旧版不透明 Key 的来源仍由用户确认。
func ValidKeyForOptions(o Options, key []byte) bool {
	if !o.Valid() || !ValidAPIKey(key) {
		return false
	}
	if bytes.HasPrefix(key, []byte("sk-api-")) && o.BillingMode != "payg" {
		return false
	}
	if bytes.HasPrefix(key, []byte("sk-cp-")) && o.BillingMode != "subscription" {
		return false
	}
	return true
}
