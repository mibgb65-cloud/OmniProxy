package mimo

import (
	"bytes"
	"errors"
	"net/url"
	"strings"
)

// Options 保存公开计费通道和集群；版本用于防止目录及在途请求混用配置。
type Options struct {
	Region      string `json:"region"`
	BillingMode string `json:"billingMode"`
	Version     int64  `json:"version"`
}

// Valid 普通按量没有集群选项，Token Plan 只接受三个官方集群。
func (o Options) Valid() bool {
	return o.Version >= 0 && (o.BillingMode == "payg" && o.Region == "" || o.BillingMode == "token_plan" && (o.Region == "cn" || o.Region == "sgp" || o.Region == "ams"))
}

// SamePool 比较会影响计费／地域归属的配置，不包含乐观锁版本。
func (o Options) SamePool(other Options) bool {
	return o.Region == other.Region && o.BillingMode == other.BillingMode
}

// Channel 为模型路由提供明确的计费与区域名称。
func (o Options) Channel() string {
	if o.BillingMode == "payg" {
		return "mimo-api"
	}
	return "mimo-" + o.Region + "-plan"
}

// BaseURL 套餐固定使用所选集群，按量保留已有受限的官方或回环入口。
func (o Options) BaseURL(paygBase string) (string, error) {
	if !o.Valid() {
		return "", errors.New("mimo: invalid_configuration")
	}
	if o.BillingMode == "token_plan" {
		return "https://token-plan-" + o.Region + ".xiaomimimo.com/v1", nil
	}
	if planBase(paygBase) {
		return "", errors.New("mimo: invalid_configuration")
	}
	return endpoint(paygBase, "/v1")
}

// ValidKeyForOptions 套餐只接受个人／团队专属 Key，普通通道拒绝套餐 Key。
func ValidKeyForOptions(o Options, key []byte) bool {
	if !o.Valid() || !validKey(key) {
		return false
	}
	plan := bytes.HasPrefix(key, []byte("tp-")) && len(key) > 3 || bytes.HasPrefix(key, []byte("ttp-")) && len(key) > 4
	if o.BillingMode == "token_plan" {
		return plan
	}
	return !bytes.HasPrefix(key, []byte("tp-")) && !bytes.HasPrefix(key, []byte("ttp-"))
}

// AccountModels 不将普通 API 的高速版误列为套餐权益。
func AccountModels(o Options) []string {
	if !o.Valid() {
		return nil
	}
	if o.BillingMode == "token_plan" {
		return []string{"mimo-v2.6-pro", "mimo-v2.6-flash"}
	}
	return Models()
}

// SupportsAccountModel 依据计费通道限制模型，而非根据客户端传入的别名猜通道。
func SupportsAccountModel(o Options, model string) bool {
	for _, id := range AccountModels(o) {
		if id == model {
			return true
		}
	}
	return false
}

func planBase(base string) bool {
	u, err := url.Parse(base)
	return err == nil && strings.HasPrefix(strings.ToLower(u.Hostname()), "token-plan-")
}

func keyMatchesBase(base string, key []byte) bool {
	o := Options{BillingMode: "payg"}
	if planBase(base) {
		o.BillingMode, o.Region = "token_plan", "cn"
	}
	return ValidKeyForOptions(o, key)
}
