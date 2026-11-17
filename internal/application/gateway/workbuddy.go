package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"sort"

	"omniproxy/internal/core/agentprotocol"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/workbuddyai"
)

// WorkBuddySubscriptionAccess 只允许在后端回调内使用登录令牌。
type WorkBuddySubscriptionAccess interface {
	Use(context.Context, string, func(workbuddyai.Credentials) error) error
	UseAt(context.Context, string, string, func(workbuddyai.Credentials) error) error
}

// SetWorkBuddySubscription 安装 WorkBuddy 订阅通道（国际版与国内版共用）。
func (s *Service) SetWorkBuddySubscription(service WorkBuddySubscriptionAccess) {
	s.workbuddyAuth = service
}

// workbuddyClient 返回该构建的客户端；未注册时返回 nil。
func (s *Service) workbuddyClient(build workbuddyai.Build) *workbuddyai.Client {
	s.workbuddyMu.Lock()
	defer s.workbuddyMu.Unlock()
	if client, ok := s.workbuddy[build.Kind]; ok {
		return client
	}
	client := workbuddyai.NewForBuild(build, nil)
	s.workbuddy[build.Kind] = client
	return client
}

// forwardWorkBuddy 借出登录令牌后转发。
//
// 国际版上游只接受流式 Chat：Responses／Messages 与「非流式的 Chat 客户端」都由本进程
// 聚合后按原协议回包。国内版为 Claude 模型额外提供原生 Anthropic Messages，调用方用
// Messages 协议访问这类模型时原样转发，不做 Chat 侧的系统提示词改写。
func (s *Service) forwardWorkBuddy(ctx context.Context, target target, request proxy.Request, consume func(proxy.Response) error) error {
	build, ok := workbuddyai.BuildOf(target.kind)
	if !ok || s.workbuddyAuth == nil {
		return proxy.Failure("adapter_unavailable", 503)
	}
	client := s.workbuddyClient(build)
	if client == nil {
		return proxy.Failure("adapter_unavailable", 503)
	}
	native := request.Protocol == workbuddyai.MessagesProtocol && build.MessagesNative(request.Model)
	return s.useWorkBuddyToken(ctx, target, func(cred workbuddyai.Credentials) error {
		if native {
			return s.forwardWorkBuddyMessages(ctx, client, cred, request, consume)
		}
		return s.forwardWorkBuddyChat(ctx, client, cred, request, consume)
	})
}

// forwardWorkBuddyMessages 把 Messages 请求原样送到国内版原生端点；流式与非流式都由
// 上游决定，网关只观察响应。
func (s *Service) forwardWorkBuddyMessages(ctx context.Context, client *workbuddyai.Client, cred workbuddyai.Credentials, request proxy.Request, consume func(proxy.Response) error) error {
	upstream := request
	upstream.Protocol = workbuddyai.MessagesProtocol
	return client.ForwardMessages(ctx, cred, upstream, func(response proxy.Response) error {
		return consume(proxy.ObserveResponse(ctx, request.Trace, response, workbuddyai.MessagesProtocol))
	})
}

func (s *Service) forwardWorkBuddyChat(ctx context.Context, client *workbuddyai.Client, cred workbuddyai.Credentials, request proxy.Request, consume func(proxy.Response) error) error {
	protocol := request.Protocol
	if protocol == "" {
		protocol = "chat"
	}
	upstream := request
	var kinds map[string]string
	body := request.Body
	if protocol != "chat" {
		converted, mapping, err := agentprotocol.ToChat(request.Body, protocol)
		if err != nil {
			return protocolContentFailure(err)
		}
		body, kinds = converted, mapping
	}
	streamBody, err := chatStreamBody(body)
	if err != nil {
		return err
	}
	upstream.Protocol = "chat"
	upstream.Body = streamBody
	upstream.Stream = true
	return client.Forward(ctx, cred, upstream, func(response proxy.Response) error {
		response = proxy.ObserveResponse(ctx, request.Trace, response, "chat")
		if response.StatusCode != 200 {
			return consume(response)
		}
		if protocol == "chat" && request.Stream {
			return consume(response)
		}
		merged, err := mergeChatStream(response.Body)
		if err != nil {
			return err
		}
		result, err := agentprotocol.ReadChat(bytes.NewReader(merged), kinds)
		if err != nil {
			return proxy.Failure("invalid_upstream_response", 502)
		}
		reader, contentType := agentprotocol.Encode(result, protocol, request.Stream)
		return consume(proxy.Response{StatusCode: 200, ContentType: contentType, Body: reader})
	})
}

// chatStreamBody 把 Chat 正文改成上游唯一接受的流式请求，并索取用量。
func chatStreamBody(body []byte) ([]byte, error) {
	if len(body) == 0 || len(body) > proxy.MaxRequestBytes {
		return nil, proxy.Failure("invalid_request", 400)
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil || fields == nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	fields["stream"] = json.RawMessage("true")
	options := map[string]json.RawMessage{}
	if raw, exists := fields["stream_options"]; exists && string(raw) != "null" {
		if json.Unmarshal(raw, &options) != nil || options == nil {
			return nil, proxy.Failure("invalid_request", 400)
		}
	}
	options["include_usage"] = json.RawMessage("true")
	encoded, err := json.Marshal(options)
	if err != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	fields["stream_options"] = encoded
	merged, err := json.Marshal(fields)
	if err != nil || len(merged) > proxy.MaxRequestBytes {
		return nil, proxy.Failure("request_too_large", 413)
	}
	return merged, nil
}

type chatStreamCall struct {
	id, name string
	args     bytes.Buffer
}

// mergeChatStream 把 Chat 的 SSE 分片聚合成一份完整快照，交给 agentprotocol.ReadChat。
// 上游已返回整份 JSON 时原样保留，避免把非流式回包误判成空流。
func mergeChatStream(reader io.Reader) ([]byte, error) {
	raw, err := io.ReadAll(io.LimitReader(reader, (8<<20)+1))
	if err != nil || len(raw) > 8<<20 {
		return nil, proxy.Failure("invalid_upstream_response", 502)
	}
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, proxy.Failure("invalid_upstream_response", 502)
	}
	if trimmed[0] == '{' {
		return trimmed, nil
	}
	var (
		id, model       string
		content         bytes.Buffer
		finish          string
		promptTokens    int
		completionToken int
	)
	calls := map[int]*chatStreamCall{}
	order := []int{}
	for _, line := range bytes.Split(trimmed, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if !bytes.HasPrefix(line, []byte("data:")) {
			continue
		}
		payload := bytes.TrimSpace(line[len("data:"):])
		if len(payload) == 0 || bytes.Equal(payload, []byte("[DONE]")) {
			continue
		}
		var envelope struct {
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
			ID      string `json:"id"`
			Model   string `json:"model"`
			Choices []struct {
				Finish string `json:"finish_reason"`
				Delta  struct {
					Content string `json:"content"`
					Calls   []struct {
						Index    int    `json:"index"`
						ID       string `json:"id"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
			Usage *struct {
				Prompt     int `json:"prompt_tokens"`
				Completion int `json:"completion_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal(payload, &envelope) != nil {
			return nil, proxy.Failure("invalid_upstream_response", 502)
		}
		if envelope.Error != nil {
			return nil, proxy.Failure("upstream_stream_failed", 502)
		}
		if envelope.ID != "" {
			id = envelope.ID
		}
		if envelope.Model != "" {
			model = envelope.Model
		}
		if envelope.Usage != nil {
			promptTokens, completionToken = envelope.Usage.Prompt, envelope.Usage.Completion
		}
		for _, choice := range envelope.Choices {
			if choice.Finish != "" {
				finish = choice.Finish
			}
			if choice.Delta.Content != "" {
				if content.Len()+len(choice.Delta.Content) > 8<<20 {
					return nil, proxy.Failure("invalid_upstream_response", 502)
				}
				content.WriteString(choice.Delta.Content)
			}
			for _, delta := range choice.Delta.Calls {
				call, ok := calls[delta.Index]
				if !ok {
					call = &chatStreamCall{}
					calls[delta.Index] = call
					order = append(order, delta.Index)
				}
				if delta.ID != "" {
					call.id = delta.ID
				}
				if delta.Function.Name != "" {
					call.name = delta.Function.Name
				}
				if delta.Function.Arguments != "" {
					if call.args.Len()+len(delta.Function.Arguments) > 2<<20 {
						return nil, proxy.Failure("invalid_upstream_response", 502)
					}
					call.args.WriteString(delta.Function.Arguments)
				}
			}
		}
	}
	if finish == "" && content.Len() == 0 && len(calls) == 0 && promptTokens == 0 {
		return nil, proxy.Failure("invalid_upstream_response", 502)
	}
	sort.Ints(order)
	type wireCall struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}
	message := struct {
		Role    string     `json:"role"`
		Content string     `json:"content"`
		Calls   []wireCall `json:"tool_calls,omitempty"`
	}{Role: "assistant", Content: content.String()}
	for _, index := range order {
		call := calls[index]
		if call.id == "" || call.name == "" {
			continue
		}
		item := wireCall{ID: call.id, Type: "function"}
		item.Function.Name, item.Function.Arguments = call.name, call.args.String()
		message.Calls = append(message.Calls, item)
	}
	body := struct {
		ID      string `json:"id"`
		Model   string `json:"model"`
		Choices []struct {
			Index   int    `json:"index"`
			Message any    `json:"message"`
			Finish  string `json:"finish_reason"`
		} `json:"choices"`
		Usage struct {
			Prompt     int `json:"prompt_tokens"`
			Completion int `json:"completion_tokens"`
		} `json:"usage"`
	}{ID: id, Model: model}
	body.Choices = append(body.Choices, struct {
		Index   int    `json:"index"`
		Message any    `json:"message"`
		Finish  string `json:"finish_reason"`
	}{Index: 0, Message: message, Finish: finish})
	body.Usage.Prompt, body.Usage.Completion = promptTokens, completionToken
	return json.Marshal(body)
}
