package gateway

import (
	"context"
	"encoding/json"

	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/proxy"
)

// carriedEffort 读取客户端在这条协议上选择的推理档位。
//
// Messages 用 output_config.effort，Responses 用 reasoning.effort，Chat 用 reasoning_effort。
func carriedEffort(protocol string, body []byte) string {
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return ""
	}
	switch protocol {
	case "messages":
		return nestedEffort(fields["output_config"])
	case "responses":
		return nestedEffort(fields["reasoning"])
	default:
		var level string
		if json.Unmarshal(fields["reasoning_effort"], &level) != nil {
			return ""
		}
		return level
	}
}

func nestedEffort(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var container map[string]json.RawMessage
	if json.Unmarshal(raw, &container) != nil {
		return ""
	}
	var level string
	if json.Unmarshal(container["effort"], &level) != nil {
		return ""
	}
	return level
}

// applyEffortVariant 把客户端档位换成目录中实际存在的上游变体模型 ID。
//
// Google 系上游用独立型号表达 thinking level（“…-low”）。目录已把这些变体折叠成
// 一行模型 + 档位枚举，因此请求必须换回具体型号，并移除已经表达的档位字段：
// 订阅适配器的生成参数是白名单校验，多余的 reasoning_effort 会被直接拒绝。
func (s *Service) applyEffortVariant(ctx context.Context, request proxy.Request) (proxy.Request, error) {
	level := carriedEffort(request.Protocol, request.Body)
	if level == "" || s.router == nil {
		return request, nil
	}
	entries, err := s.Catalog(ctx)
	if err != nil {
		// 目录不可用时不改写请求；后面的路由解析仍会给出明确失败码。
		return request, nil
	}
	for _, entry := range entries {
		if entry.ID != request.Model && entry.TargetModel != request.Model {
			continue
		}
		variant, ok := catalog.EffortModel(entry, level)
		if !ok {
			return request, nil
		}
		body, err := rewriteEffortVariant(request.Protocol, request.Body, variant)
		if err != nil {
			return request, err
		}
		request.Body, request.Model = body, variant
		return request, nil
	}
	return request, nil
}

// rewriteEffortVariant 改写模型 ID 并移除档位字段，保留同级的其他配置。
func rewriteEffortVariant(protocol string, body []byte, model string) ([]byte, error) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	encoded, _ := json.Marshal(model)
	fields["model"] = encoded
	switch protocol {
	case "messages":
		if err := stripNestedEffort(fields, "output_config"); err != nil {
			return nil, err
		}
	case "responses":
		if err := stripNestedEffort(fields, "reasoning"); err != nil {
			return nil, err
		}
	default:
		delete(fields, "reasoning_effort")
	}
	result, err := json.Marshal(fields)
	if err != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	return result, nil
}

func stripNestedEffort(fields map[string]json.RawMessage, key string) error {
	raw, exists := fields[key]
	if !exists || string(raw) == "null" {
		return nil
	}
	var container map[string]json.RawMessage
	if json.Unmarshal(raw, &container) != nil || container == nil {
		return proxy.Failure("invalid_request", 400)
	}
	delete(container, "effort")
	if len(container) == 0 {
		delete(fields, key)
		return nil
	}
	encoded, err := json.Marshal(container)
	if err != nil {
		return proxy.Failure("invalid_request", 400)
	}
	fields[key] = encoded
	return nil
}

// applyLegacyVariant 把目录折叠前的档位变体 ID 还原为基础模型 + 档位。
//
// 折叠只在下次同步模型后生效，用户已保存的选择仍可能指向 “…-low” 这种旧 ID；
// 这里只在路由判定为未知模型后调用，避免给正常请求增加目录读取。
func (s *Service) applyLegacyVariant(ctx context.Context, request proxy.Request) (proxy.Request, bool, error) {
	if s.router == nil {
		return request, false, nil
	}
	entries, err := s.Catalog(ctx)
	if err != nil {
		return request, false, nil
	}
	entry, level, ok := catalog.EffortVariantBase(entries, request.Model)
	if !ok {
		return request, false, nil
	}
	body, err := setEffortVariantLevel(request.Protocol, request.Body, entry.ID, level)
	if err != nil {
		return request, false, err
	}
	request.Body, request.Model = body, entry.ID
	return request, true, nil
}

// setEffortVariantLevel 改写模型并把该档位写回对应协议的字段。
func setEffortVariantLevel(protocol string, body []byte, model, level string) ([]byte, error) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	encodedModel, _ := json.Marshal(model)
	encodedLevel, _ := json.Marshal(level)
	fields["model"] = encodedModel
	switch protocol {
	case "messages":
		container := map[string]json.RawMessage{}
		if raw, exists := fields["output_config"]; exists && string(raw) != "null" {
			if json.Unmarshal(raw, &container) != nil {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
		container["effort"] = encodedLevel
		fields["output_config"], _ = json.Marshal(container)
	case "responses":
		container := map[string]json.RawMessage{}
		if raw, exists := fields["reasoning"]; exists && string(raw) != "null" {
			if json.Unmarshal(raw, &container) != nil {
				return nil, proxy.Failure("invalid_request", 400)
			}
		}
		container["effort"] = encodedLevel
		fields["reasoning"], _ = json.Marshal(container)
	default:
		fields["reasoning_effort"] = encodedLevel
	}
	result, err := json.Marshal(fields)
	if err != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	return result, nil
}
