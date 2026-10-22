package gateway

import "context"

// effortPins 读取用户在应用内为模型固定的思考档位。
// 读取失败只记录日志并沿用客户端选择，不让整个请求因为一次配置读取失败而中止。
func (s *Service) effortPins(ctx context.Context) map[string]string {
	source, ok := s.router.(interface {
		EffortPins(context.Context) (map[string]string, error)
	})
	if !ok {
		return nil
	}
	pins, err := source.EffortPins(ctx)
	if err != nil {
		if s.log != nil {
			s.log.Warn("model effort pins unavailable; the client selection applies", "code", "effort_pins_unavailable")
		}
		return nil
	}
	return pins
}
