package gateway

import (
	"context"
	"encoding/json"
	"omniproxy/internal/core/agentprotocol"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/antigravity"
)

// 订阅模型共用文本／工具适配；跨协议等待完整结果，保留已有 Chat 的增量流。
func (s *Service) forwardGoogleAgent(ctx context.Context, cred antigravity.Credentials, request proxy.Request, consume func(proxy.Response) error) error {
	upstream := request
	var kinds map[string]string
	if request.Protocol != "chat" {
		if request.Protocol != "responses" && request.Protocol != "messages" {
			return proxy.Failure("unsupported_protocol", 400)
		}
		body, toolKinds, err := agentprotocol.ToChat(request.Body, request.Protocol)
		if err != nil {
			return proxy.Failure("unsupported_content", 400)
		}
		var fields map[string]json.RawMessage
		if json.Unmarshal(body, &fields) != nil {
			return proxy.Failure("invalid_request", 400)
		}
		// 中立转换器用于 DeepSeek 的字段不能传给 Google，思考策略由实际订阅型号决定。
		delete(fields, "thinking")
		upstream.Body, err = json.Marshal(fields)
		if err != nil {
			return proxy.Failure("invalid_request", 400)
		}
		upstream.Protocol, upstream.Stream, kinds = "chat", false, toolKinds
	}
	return s.antigravity.Forward(ctx, cred, upstream, func(response proxy.Response) error {
		response = proxy.ObserveResponse(ctx, request.Trace, response, "chat")
		if request.Protocol == "chat" || response.StatusCode != 200 {
			return consume(response)
		}
		result, err := agentprotocol.ReadChat(response.Body, kinds)
		if err != nil {
			return proxy.Failure("invalid_upstream_response", 502)
		}
		reader, contentType := agentprotocol.Encode(result, request.Protocol, request.Stream)
		return consume(proxy.Response{StatusCode: 200, ContentType: contentType, Body: reader})
	})
}
