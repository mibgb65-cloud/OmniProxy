package antigravity

import (
	"encoding/json"
	"omniproxy/internal/core/proxy"
)

// toolChoice 把 Chat 的 tool_choice 映射为 functionCallingConfig：
// Antigravity 的 Claude 模型要求调用按 schema 校验（VALIDATED），其余用 AUTO。
func toolChoice(raw json.RawMessage, tools []chatTool, family string) (object, error) {
	if len(raw) > 0 && string(raw) != "null" {
		var choice string
		config := object{}
		if json.Unmarshal(raw, &choice) == nil {
			mode := map[string]string{"auto": "AUTO", "none": "NONE", "required": "ANY"}[choice]
			if mode == "" {
				return nil, proxy.Failure("unsupported_content", 400)
			}
			if choice == "auto" && family == familyClaude {
				mode = "VALIDATED"
			}
			config["mode"] = mode
		} else {
			var named struct {
				Type     string `json:"type"`
				Function struct {
					Name string `json:"name"`
				} `json:"function"`
			}
			if json.Unmarshal(raw, &named) != nil || named.Type != "function" || !safeString(named.Function.Name, 128) {
				return nil, proxy.Failure("unsupported_content", 400)
			}
			found := false
			for _, tool := range tools {
				var f struct {
					Name string `json:"name"`
				}
				if json.Unmarshal(tool.Function, &f) == nil && f.Name == named.Function.Name {
					found = true
				}
			}
			if !found {
				return nil, proxy.Failure("invalid_request", 400)
			}
			config["mode"], config["allowedFunctionNames"] = "ANY", []string{named.Function.Name}
		}
		return object{"functionCallingConfig": config}, nil
	}
	return nil, nil
}
