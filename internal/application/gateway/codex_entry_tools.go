package gateway

import (
	"context"
	"encoding/json"
	"strings"

	"omniproxy/internal/core/proxy"
)

// codexUnsupportedIncludes 是 Codex 会请求、但第三方服务商无法表达的 include 取值。
//
// `output.encrypted_content` 是 OpenAI 的加密输出能力（Codex 0.169 起默认请求）；
// River 这类服务商收到后会在 SSE 开头之后中断连接。加密输出本来也无法从第三方服务商
// 获得，请求它没有意义。`reasoning.encrypted_content` 实测可用，继续保留。
var codexUnsupportedIncludes = map[string]bool{
	"output.encrypted_content": true,
}

// adaptCodexEntryBody 把 Codex 入口改交给第三方服务商的正文收拾成该服务商能接受的形状。
//
// Codex 认为自己在跟 OpenAI 说话，会送出三类 River 这类服务商无法表达的输入：
//
//   - 工具分组：MCP 与多智能体工具被包成 `type: namespace`，服务商只接受
//     `type: function`，留着会让整个请求被本地校验拒成 `unsupported_content`。
//   - 孤儿工具结果：桌面端把 Companion Page 快照注入成没有 `call_id` 的
//     `function_call_output`；这类条目没有对应的工具调用，服务商收到后会在 SSE
//     开头之后直接中断连接，客户端只能看到 `error decoding response body`。
//   - 加密输出请求：`include` 里的 `output.encrypted_content` 是 OpenAI 专有能力，
//     服务商收到同样会在 SSE 开头之后中断连接。
//
// 这里只移除无法表达的条目与取值，函数工具、合法工具结果与其它字段原样保留，并把移除
// 数量写进日志，能力差异不静默吞掉。
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
	changed := false
	if kept, dropped := withoutNamespaceTools(fields["tools"]); dropped > 0 {
		encoded, err := json.Marshal(kept)
		if err != nil {
			return proxy.Failure("invalid_request", 400)
		}
		fields["tools"] = encoded
		changed = true
		if s.log != nil {
			s.log.Info("codex namespace tools dropped", "module", "gateway", "count", dropped)
		}
	}
	if kept, dropped := withoutOrphanToolOutputs(fields["input"]); dropped > 0 {
		encoded, err := json.Marshal(kept)
		if err != nil {
			return proxy.Failure("invalid_request", 400)
		}
		fields["input"] = encoded
		changed = true
		if s.log != nil {
			s.log.Info("codex orphan tool outputs dropped", "module", "gateway", "count", dropped)
		}
	}
	if kept, dropped := withoutUnsupportedIncludes(fields["include"]); len(dropped) > 0 {
		if len(kept) == 0 {
			delete(fields, "include")
		} else {
			encoded, err := json.Marshal(kept)
			if err != nil {
				return proxy.Failure("invalid_request", 400)
			}
			fields["include"] = encoded
		}
		changed = true
		if s.log != nil {
			s.log.Info("codex unsupported includes dropped", "module", "gateway", "count", len(dropped),
				"values", strings.Join(dropped, ","))
		}
	}
	if !changed {
		return nil
	}
	body, err := json.Marshal(fields)
	if err != nil {
		return proxy.Failure("invalid_request", 400)
	}
	request.Body = body
	return nil
}

// withoutNamespaceTools 移除 `type: "namespace"` 工具分组，返回保留项与移除数量。
// 缺少 tools、或 tools 不是数组时原样返回（不计数），保持既有的宽松处理。
func withoutNamespaceTools(raw json.RawMessage) ([]json.RawMessage, int) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, 0
	}
	var tools []json.RawMessage
	if json.Unmarshal(raw, &tools) != nil {
		return nil, 0
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
	return kept, dropped
}

// withoutOrphanToolOutputs 移除没有 `call_id` 的工具结果输入项，返回保留项与移除数量。
//
// 工具结果必须用 `call_id` 关联到同一段历史里的工具调用；Codex 桌面端的
// Companion Page 快照没有这个字段（rollout 里只有 name／namespace），River 这类
// 服务商收到后会在发出响应头之后中断连接。缺少 input 或 input 不是数组时不改动。
func withoutOrphanToolOutputs(raw json.RawMessage) ([]json.RawMessage, int) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, 0
	}
	var items []json.RawMessage
	if json.Unmarshal(raw, &items) != nil {
		return nil, 0
	}
	kept := make([]json.RawMessage, 0, len(items))
	dropped := 0
	for _, item := range items {
		if orphanToolOutput(item) {
			dropped++
			continue
		}
		kept = append(kept, item)
	}
	return kept, dropped
}

// orphanToolOutput 报告一个输入项是否是无法表达的工具结果：类型是工具结果，
// 但 call_id 缺失、为 null 或不是非空字符串。
func orphanToolOutput(item json.RawMessage) bool {
	var shape struct {
		Type   string          `json:"type"`
		CallID json.RawMessage `json:"call_id"`
	}
	if json.Unmarshal(item, &shape) != nil {
		return false
	}
	if shape.Type != "function_call_output" {
		return false
	}
	var callID string
	if json.Unmarshal(shape.CallID, &callID) != nil {
		return true
	}
	return strings.TrimSpace(callID) == ""
}

// withoutUnsupportedIncludes 从 `include` 数组里移除目标服务商无法表达的取值，
// 返回保留项（原始 JSON，非字符串元素原样保留）与被移除的取值。`include` 缺失、
// 为 null 或不是数组时不改动，避免替客户端解释不认识的形状。
func withoutUnsupportedIncludes(raw json.RawMessage) ([]json.RawMessage, []string) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var values []json.RawMessage
	if json.Unmarshal(raw, &values) != nil {
		return nil, nil
	}
	kept := make([]json.RawMessage, 0, len(values))
	dropped := make([]string, 0)
	for _, value := range values {
		var name string
		if json.Unmarshal(value, &name) == nil && codexUnsupportedIncludes[name] {
			dropped = append(dropped, name)
			continue
		}
		kept = append(kept, value)
	}
	return kept, dropped
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
