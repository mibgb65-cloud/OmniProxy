package agentconfig

import (
	"encoding/json"

	"omniproxy/internal/core/agentendpoint"
)

const qwenCredentialRef = "OMNIPROXY_QWEN_CODE_KEY"

func configObject(parent map[string]any, key string) (map[string]any, error) {
	value, ok := parent[key].(map[string]any)
	if parent[key] != nil && !ok {
		return nil, failure("invalid_config")
	}
	if value == nil {
		value = map[string]any{}
		parent[key] = value
	}
	return value, nil
}

func buildQwenCode(body, key []byte, port int, plan Plan, owned bool) ([]byte, string, error) {
	_, config, err := openCodeConfig(body)
	if err != nil {
		return nil, "", err
	}
	// 仅新文件直接采用已验证的 v4 格式；历史设置必须先由官方客户端完成迁移，
	// 不能跳过迁移后仍把旧权限／模型字段标成新版本，也避免启动时重写破坏恢复版本。
	if len(config) == 0 {
		config["$version"] = 4
	} else if config["$version"] != float64(4) {
		return nil, "", failure("client_upgrade_required")
	}
	providers, err := configObject(config, "modelProviders")
	if err != nil {
		return nil, "", err
	}
	current, ok := providers["openai"].([]any)
	if providers["openai"] != nil && !ok {
		return nil, "", failure("invalid_config")
	}
	env, err := configObject(config, "env")
	if err != nil {
		return nil, "", err
	}
	if env[qwenCredentialRef] != nil && !owned {
		return nil, "", failure("profile_conflict")
	}
	remaining := []any{}
	for _, raw := range current {
		entry, ok := raw.(map[string]any)
		if !ok {
			return nil, "", failure("invalid_config")
		}
		if entry["envKey"] == qwenCredentialRef {
			if !owned {
				return nil, "", failure("profile_conflict")
			}
			continue
		}
		remaining = append(remaining, entry)
	}
	root, ok := agentendpoint.BaseURL("qwen-code", port)
	if !ok {
		return nil, "", failure("invalid_client")
	}
	root += "/v1"
	models := []any{}
	for _, entry := range plan.Models {
		model := map[string]any{"id": entry.ID, "name": "OmniProxy · " + entry.Name, "baseUrl": root, "envKey": qwenCredentialRef, "wireApi": "chat-completions"}
		generation := map[string]int64{}
		if entry.Metadata.ContextWindow != nil {
			generation["contextWindowSize"] = *entry.Metadata.ContextWindow
		}
		if entry.Metadata.OutputTokenLimit != nil {
			generation["maxOutputTokens"] = *entry.Metadata.OutputTokenLimit
		}
		if len(generation) > 0 {
			model["generationConfig"] = generation
		}
		models = append(models, model)
	}
	providers["openai"] = append(models, remaining...)
	env[qwenCredentialRef] = string(key)
	security, err := configObject(config, "security")
	if err != nil {
		return nil, "", err
	}
	auth, err := configObject(security, "auth")
	if err != nil {
		return nil, "", err
	}
	auth["selectedType"], auth["baseUrl"] = "openai", root
	model, err := configObject(config, "model")
	if err != nil {
		return nil, "", err
	}
	model["name"] = plan.DefaultModel
	// Qwen 用 JSON.parse 读取去注释后的文档，必须输出无尾逗号的标准 JSON。
	encoded, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, "", failure("invalid_config")
	}
	return append(encoded, '\n'), "openai", nil
}

func qwenCodeModel(body []byte) string {
	_, config, err := openCodeConfig(body)
	if err != nil {
		return ""
	}
	model, _ := config["model"].(map[string]any)
	value, _ := model["name"].(string)
	return value
}
