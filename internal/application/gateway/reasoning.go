package gateway

import (
	"encoding/json"

	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/provider/river"
)

// claudeRequestedEffort 只读取客户端原生 effort 选择，不解析其他字段也不猜测默认值。
func claudeRequestedEffort(body []byte) string {
	var fields struct {
		OutputConfig json.RawMessage `json:"output_config"`
	}
	if json.Unmarshal(body, &fields) != nil || fields.OutputConfig == nil {
		return ""
	}
	var config struct {
		Effort string `json:"effort"`
	}
	if json.Unmarshal(fields.OutputConfig, &config) != nil {
		return ""
	}
	return config.Effort
}

func withClaudeEffort(body []byte, effort string) ([]byte, error) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	config := map[string]json.RawMessage{}
	if raw, ok := fields["output_config"]; ok && (json.Unmarshal(raw, &config) != nil || config == nil) {
		return nil, proxy.Failure("invalid_request", 400)
	}
	config["effort"], _ = json.Marshal(effort)
	fields["output_config"], _ = json.Marshal(config)
	// 菜单档位是用户的明确选择，不能被客户端沿用的旧模型 thinking=disabled 覆盖。
	mode := "adaptive"
	if effort == "none" {
		mode = "disabled"
	}
	fields["thinking"], _ = json.Marshal(map[string]string{"type": mode})
	return json.Marshal(fields)
}

func riverMessagesEffort(body []byte, model string) (json.RawMessage, error) {
	var source map[string]json.RawMessage
	if json.Unmarshal(body, &source) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	raw, exists := source["output_config"]
	if !exists {
		return nil, nil
	}
	var config map[string]json.RawMessage
	if json.Unmarshal(raw, &config) != nil || config == nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	raw, exists = config["effort"]
	if !exists {
		return nil, nil
	}
	var effort string
	if json.Unmarshal(raw, &effort) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	upstream, ok := catalog.UpstreamEffort(river.ModelMetadata(model).ReasoningEfforts, effort)
	if !ok {
		return nil, proxy.Failure("unsupported_reasoning_effort", 400)
	}
	return json.Marshal(upstream)
}

// withResponsesEffort 把客户端（Codex）选择的档位改写为该模型上游接受的值。
//
// Codex 目录发布的是就近映射后的档位（minimal／xhigh），上游只接受 none／max；
// 未带档位或带的是本模型已确认的上游取值时保持原样。
func withResponsesEffort(body []byte, efforts []catalog.Effort) ([]byte, error) {
	var fields map[string]json.RawMessage
	if json.Unmarshal(body, &fields) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	raw, exists := fields["reasoning"]
	if !exists || string(raw) == "null" {
		return body, nil
	}
	var reasoning map[string]json.RawMessage
	if json.Unmarshal(raw, &reasoning) != nil || reasoning == nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	raw, exists = reasoning["effort"]
	if !exists {
		return body, nil
	}
	var level string
	if json.Unmarshal(raw, &level) != nil {
		return nil, proxy.Failure("invalid_request", 400)
	}
	upstream, ok := catalog.UpstreamEffort(efforts, level)
	if !ok {
		return nil, proxy.Failure("unsupported_reasoning_effort", 400)
	}
	reasoning["effort"], _ = json.Marshal(upstream)
	fields["reasoning"], _ = json.Marshal(reasoning)
	return json.Marshal(fields)
}
