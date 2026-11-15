package gateway

import (
	"context"
	"encoding/json"

	"omniproxy/internal/core/proxy"
)

// adaptCodexEntryBody 把 Codex 入口改交给第三方服务商的正文收拾成该服务商能接受的形状。
//
// Codex 认为自己在跟 OpenAI 说话，会把 MCP 与多智能体工具包成 `type: namespace`；
// River 这类服务商只接受 `type: function`，留着会让整个请求被本地校验拒成
// `unsupported_content`。这里只移除无法表达的分组，函数工具与其它字段原样保留，
// 并把移除数量写进日志，能力差异不静默吞掉。
func (s *Service) adaptCodexEntryBody(ctx context.Context, request *proxy.Request) error {
	if !request.FromCodexEntry || s.providers == nil {
		return nil
	}
	kind, err := s.providerKind(ctx, request.ProviderID)
	if err != nil || kind == "openai" {
		return nil
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(request.Body, &fields) != nil || fields == nil {
		return nil
	}
	raw, exists := fields["tools"]
	if !exists || len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var tools []json.RawMessage
	if json.Unmarshal(raw, &tools) != nil {
		return nil
	}
	kept := make([]json.RawMessage, 0, len(tools))
	dropped := 0
	for _, tool := range tools {
		var shape struct {
			Type string `json:"type"`
		}
		if json.Unmarshal(tool, &shape) == nil && shape.Type == "namespace" {
			dropped++
			continue
		}
		kept = append(kept, tool)
	}
	if dropped == 0 {
		return nil
	}
	encoded, err := json.Marshal(kept)
	if err != nil {
		return proxy.Failure("invalid_request", 400)
	}
	fields["tools"] = encoded
	body, err := json.Marshal(fields)
	if err != nil {
		return proxy.Failure("invalid_request", 400)
	}
	request.Body = body
	if s.log != nil {
		s.log.Info("codex namespace tools dropped", "module", "gateway", "count", dropped)
	}
	return nil
}

// providerKind 只读取已保存提供商的类型，未知目标按空串返回。
func (s *Service) providerKind(ctx context.Context, id string) (string, error) {
	providers, err := s.providers.List(ctx)
	if err != nil {
		return "", err
	}
	for _, item := range providers {
		if item.ID == id {
			return item.Kind, nil
		}
	}
	return "", nil
}
