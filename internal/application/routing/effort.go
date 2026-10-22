package routing

import (
	"context"

	"omniproxy/internal/application/modeleffort"
)

// EffortPins 返回全部固定档位，键是模型 ID；网关在每次 Messages 请求上读取。
// 写入由受管客户端配置动作统一完成：档位必须属于该模型已确认的枚举。
func (s *Service) EffortPins(ctx context.Context) (map[string]string, error) {
	return modeleffort.Pins(ctx, s.db)
}
