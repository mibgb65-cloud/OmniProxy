package gateway

import (
	"context"
	"encoding/json"
	"omniproxy/internal/core/agentprotocol"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/river"
)

func (s *Service) forwardRiver(ctx context.Context, target target, request proxy.Request, consume func(proxy.Response) error) error {
	upstream := request
	var kinds map[string]string
	if request.Protocol == "messages" {
		body, toolKinds, err := agentprotocol.ToChat(request.Body, "messages")
		if err != nil {
			return protocolContentFailure(err)
		}
		var fields map[string]json.RawMessage
		if json.Unmarshal(body, &fields) != nil {
			return proxy.Failure("invalid_request", 400)
		}
		// 既有转换器的 thinking 是 DeepSeek 专有字段，River 使用自己的默认推理策略。
		delete(fields, "thinking")
		effort, err := riverMessagesEffort(request.Body, request.Model)
		if err != nil {
			return err
		}
		if effort != nil {
			fields["reasoning_effort"] = effort
		} else if _, exists := fields["reasoning_effort"]; !exists {
			// 该模型已确认有档位枚举、但用户未显式选择时，使用 River 公布的默认档位，
			// 避免客户端在原生 effort 控件下保留的旧选择或空值改变默认行为。
			if defaultEffort := river.ModelMetadata(request.Model).DefaultEffort; defaultEffort != "" {
				fields["reasoning_effort"], _ = json.Marshal(defaultEffort)
			}
		}
		upstream.Body, err = json.Marshal(fields)
		if err != nil {
			return proxy.Failure("invalid_request", 400)
		}
		upstream.Protocol, upstream.Stream, kinds = "chat", false, toolKinds
		upstream.ConvertedFromMessages = true
	}
	return s.useAPIKey(ctx, target, func(key []byte) error {
		return s.river.Forward(ctx, target.baseURL, key, upstream, func(response proxy.Response) error {
			response = proxy.ObserveResponse(ctx, request.Trace, response, upstream.Protocol)
			if request.Protocol != "messages" || response.StatusCode != 200 {
				return consume(response)
			}
			result, err := agentprotocol.ReadChat(response.Body, kinds)
			if err != nil {
				return proxy.Failure("invalid_upstream_response", 502)
			}
			reader, contentType := agentprotocol.Encode(result, request.Protocol, request.Stream)
			return consume(proxy.Response{StatusCode: 200, ContentType: contentType, Body: reader})
		})
	})
}
