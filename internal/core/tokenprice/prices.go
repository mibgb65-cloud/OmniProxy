// Package tokenprice 提供美元基础参考价；不是实时账单或订阅的官方美元余额。
package tokenprice

import "strings"

// Price 单位为 USD / 百万 Token；缓存未知时由调用方展示成本区间。
type Price struct {
	Input, CachedInput, Output float64
}

// 新模型价格核对于 2026-10-20：
// https://developers.openai.com/api/docs/pricing
// 其他基础参考价迁移自旧版 frontend/src/domain/billing/pricing.js。
// 不根据前缀猜未发布型号，不将人民币单价擅自换成美元，不对聚合用量套长上下文倍率。
var prices = map[string]Price{
	"gpt-6-astra":           {10, 1, 50},
	"gpt-6.1-sol":           {2, 0.1, 10},
	"gpt-6-sol":             {2, 0.2, 10},
	"gpt-6-luna":            {0.1, 0.01, 0.5},
	"gpt-5.6-sol":           {4, 0.4, 20},
	"gpt-5.6-terra":         {2.5, 0.25, 15},
	"gpt-5.6-luna":          {1, 0.1, 6},
	"gpt-5.5":               {2.5, 0.25, 15},
	"gpt-5.4":               {2.5, 0.25, 15},
	"gpt-5.4-mini":          {0.75, 0.075, 4.5},
	"gpt-5.3-codex":         {1.75, 0.175, 14},
	"claude-opus-4-5":       {5, 5, 25},
	"claude-opus-4-6":       {5, 5, 25},
	"claude-opus-4-7":       {5, 5, 25},
	"claude-sonnet-4":       {3, 3, 15},
	"claude-sonnet-4-5":     {3, 3, 15},
	"claude-sonnet-4-6":     {3, 3, 15},
	"claude-haiku-4-5":      {1, 1, 5},
	"deepseek-v4-pro":       {0.435, 0.435, 0.87},
	"deepseek-v4-flash":     {0.14, 0.14, 0.28},
	"gemini-2.5-pro":        {1.25, 1.25, 10},
	"gemini-2.5-flash":      {0.3, 0.3, 2.5},
	"gemini-2.5-flash-lite": {0.1, 0.1, 0.4},
	"kimi-k2.6":             {0.95, 0.95, 4},
	"kimi-k2.5":             {0.6, 0.6, 3},
	"kimi-k2":               {0.6, 0.6, 2.5},
	"minimax-m2":            {0.3, 0.3, 1.2},
	"minimax-m2.1":          {0.3, 0.3, 1.2},
	"minimax-m2.5":          {0.3, 0.3, 1.2},
	"minimax-m2.7":          {0.3, 0.3, 1.2},
	"minimax-m2-highspeed":  {0.6, 0.6, 2.4},
	"mimo-v2.5":             {0.4, 0.4, 2},
	"mimo-v2-pro":           {1, 1, 3},
}

// Lookup 只按实际模型匹配，未知型号绝不回退为某个 GPT 的价格。
func Lookup(model string) (Price, bool) {
	model = strings.TrimSuffix(strings.ToLower(strings.TrimSpace(model)), "[1m]")
	price, ok := prices[model]
	return price, ok
}

// Cost 返回缓存未知时的上下界；输入总量已经包含缓存读取，不能重复计费。
func (p Price) Cost(input, output int64, cached *int64) (float64, float64) {
	if cached != nil {
		value := (float64(input-*cached)*p.Input + float64(*cached)*p.CachedInput + float64(output)*p.Output) / 1e6
		return value, value
	}
	return (float64(input)*p.CachedInput + float64(output)*p.Output) / 1e6,
		(float64(input)*p.Input + float64(output)*p.Output) / 1e6
}
