// Package stepfun 提供阶跃星辰两平台、两计费通道的原生文本 API。
package stepfun

import "errors"

const Kind = "stepfun"
const DefaultBaseURL = "https://api.stepfun.com/v1"

// Options 是公开账号配置；Version 用于阻止并发编辑覆盖另一通道配置。
type Options struct {
	Region      string `json:"region"`
	BillingMode string `json:"billingMode"`
	Version     int64  `json:"version"`
}

// Valid 仅允许已核实的四类官方入口，不接受任意 URL。
func (o Options) Valid() bool {
	return (o.Region == "cn" || o.Region == "global") && (o.BillingMode == "payg" || o.BillingMode == "step_plan") && o.Version >= 0
}

// BaseURL 根据账号明确选择解析入口，不在请求失败时切换平台或计费方式。
func (o Options) BaseURL() (string, error) {
	if !o.Valid() {
		return "", errors.New("stepfun: invalid_configuration")
	}
	host := "https://api.stepfun.com"
	if o.Region == "global" {
		host = "https://api.stepfun.ai"
	}
	if o.BillingMode == "step_plan" {
		host += "/step_plan"
	}
	return host + "/v1", nil
}

// SamePool 比较计费通道，版本变化不产生新的通道身份。
func (o Options) SamePool(other Options) bool {
	return o.Valid() && other.Valid() && o.Region == other.Region && o.BillingMode == other.BillingMode
}
