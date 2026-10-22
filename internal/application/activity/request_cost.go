package activity

import (
	"database/sql"

	"omniproxy/internal/core/tokenprice"
)

// attachCost 只用基础参考价估算单条请求的美元成本，不是账单金额。
// 用量不完整或型号没有已确认价格时保持未知；缓存明细未知时保留上下界区间。
func (item *RequestItem) attachCost(input, output int64, cached sql.NullInt64) {
	item.CostMin, item.CostMax, item.CostKnown = 0, 0, false
	if !item.UsageComplete || input < 0 || output < 0 {
		return
	}
	price, ok := tokenprice.Lookup(item.Model)
	if !ok {
		return
	}
	var cachedInput *int64
	if cached.Valid {
		cachedInput = &cached.Int64
	}
	item.CostMin, item.CostMax = price.Cost(input, output, cachedInput)
	item.CostKnown = true
}
