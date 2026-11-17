package workbuddyai

import (
	"bytes"
	"context"
	"io"
	"net/http"

	"omniproxy/internal/core/proxy"
)

// ForwardMessages 把 Anthropic Messages 请求送到构建的原生 `/v1/messages`
// （目前只有国内版的 Claude 模型走这里）。
//
// 与 Chat 的差异（社区插件 0.2.36 实测口径）：
//   - 正文按调用方发来原样转发，不 gzip、不补系统提示词、不改 max_tokens，
//     也不做 11128 的「系统提示词下移」重试——那些改写只对 Chat 端点有效；
//   - 头集与 Chat 相同（签名头、客户端头、会话 ID、SDK 指纹），只是不声明 gzip；
//   - 流式与非流式都按上游返回原样交给网关，由网关按客户端协议聚合或转发。
func (c *Client) ForwardMessages(ctx context.Context, cred Credentials, request proxy.Request, consume func(proxy.Response) error) (result error) {
	if !c.build.Messages || request.Protocol != MessagesProtocol {
		return proxy.Failure("unsupported_protocol", 400)
	}
	if !cred.Valid() || consume == nil {
		return proxy.Failure("invalid_credential", 503)
	}
	if len(request.Body) == 0 || len(request.Body) > proxy.MaxRequestBytes {
		return proxy.Failure("invalid_request", 400)
	}
	reply, err := c.sendMessages(ctx, cred, request.Body)
	if err != nil {
		return err
	}
	response := reply.response
	defer func() {
		if err := response.Body.Close(); err != nil && result == nil {
			result = proxy.Failure(CodeUpstreamUnavailable, http.StatusBadGateway)
		}
	}()
	switch response.StatusCode {
	case http.StatusUnauthorized:
		return proxy.Failure(CodeUnauthorized, http.StatusUnauthorized)
	case http.StatusTooManyRequests:
		return proxy.Failure(CodeRateLimited, http.StatusTooManyRequests)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		if reply.hasCode {
			if mapped, known := businessFailure(reply.code); known {
				return mapped
			}
		}
		response.Body = replayedBody{Reader: io.MultiReader(bytes.NewReader(reply.prefix), response.Body), Closer: response.Body}
	}
	return consume(proxy.Response{
		StatusCode:  response.StatusCode,
		ContentType: response.Header.Get("Content-Type"),
		RetryAfter:  response.Header.Get("Retry-After"),
		Body:        response.Body,
	})
}

// sendMessages 发一次原生 Messages 请求：不压缩正文，头集与 Chat 一致。
func (c *Client) sendMessages(ctx context.Context, cred Credentials, body []byte) (*chatReply, error) {
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+messagesPath, bytes.NewReader(body))
	if err != nil {
		return nil, proxy.UpstreamFailure(err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	c.applySign(httpRequest, cred)
	defer httpRequest.Header.Del("Authorization")
	applyChatHeaders(httpRequest)
	if err := applyConversationIDs(httpRequest, body); err != nil {
		return nil, err
	}
	response, err := c.http.Do(httpRequest)
	if err != nil {
		return nil, proxy.UpstreamFailure(err)
	}
	reply := &chatReply{response: response}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		reply.prefix, _ = io.ReadAll(io.LimitReader(response.Body, maxClassifiedBytes))
		if code, found := upstreamBusinessCode(reply.prefix); found {
			reply.code, reply.hasCode = code, true
		}
	}
	return reply, nil
}
