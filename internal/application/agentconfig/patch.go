package agentconfig

import (
	"encoding/json"
	"github.com/pelletier/go-toml/v2"
	"omniproxy/internal/core/agentendpoint"
	catalog "omniproxy/internal/core/modelcatalog"
)

func build(client string, body, key []byte, port int, model string) ([]byte, string, error) {
	root, ok := agentendpoint.BaseURL(client, port)
	if !ok {
		return nil, "", failure("invalid_client")
	}
	if client == "codex" {
		return buildCodex(body, key, root+"/v1", model)
	}
	config := map[string]any{}
	if len(body) > 0 && json.Unmarshal(body, &config) != nil || config == nil {
		return nil, "", failure("invalid_config")
	}
	if client == "claude-desktop" {
		// 独立配置库条目由激活流程选择；禁止登录选择器，避免重启后回到官方登录。
		config["disableDeploymentModeChooser"] = true
		config["inferenceProvider"] = "gateway"
		config["inferenceGatewayBaseUrl"] = root
		config["inferenceGatewayApiKey"] = string(key)
		config["inferenceGatewayAuthScheme"] = "bearer"
		config["inferenceModels"] = []any{map[string]any{"name": catalog.DesktopAlias(model), "labelOverride": model}}
	} else {
		env, ok := config["env"].(map[string]any)
		if config["env"] != nil && !ok {
			return nil, "", failure("invalid_config")
		}
		if env == nil {
			env = map[string]any{}
		}
		env["ANTHROPIC_BASE_URL"] = root
		env["ANTHROPIC_AUTH_TOKEN"] = string(key)
		env["ANTHROPIC_MODEL"] = model
		for _, k := range []string{"ANTHROPIC_DEFAULT_OPUS_MODEL", "ANTHROPIC_DEFAULT_SONNET_MODEL", "ANTHROPIC_DEFAULT_HAIKU_MODEL"} {
			env[k] = model
		}
		delete(env, "ANTHROPIC_API_KEY")
		delete(env, "CLAUDE_CODE_OAUTH_TOKEN")
		config["env"] = env
	}
	result, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, "", failure("invalid_config")
	}
	return append(result, '\n'), "", nil
}
func buildCodex(body, key []byte, baseURL, model string) ([]byte, string, error) {
	config := map[string]any{}
	if len(body) > 0 && toml.Unmarshal(body, &config) != nil {
		return nil, "", failure("invalid_config")
	}
	provider := "openai"
	if v, ok := config["model_provider"]; ok {
		var valid bool
		provider, valid = v.(string)
		if !valid || len(provider) > 128 || provider == "" {
			return nil, "", failure("invalid_config")
		}
	}
	tables, _ := config["model_providers"].(map[string]any)
	entry, _ := tables[provider].(map[string]any)
	if entry["auth"] != nil {
		return nil, provider, failure("auth_conflict")
	}
	// 活跃 profile 的提供商覆盖需人工处理，不能擅自改变身份。
	if p, ok := config["profile"].(string); ok && p != "" {
		return nil, provider, failure("profile_conflict")
	}
	updates := map[string]string{"model": quote(model), "web_search": quote("disabled")}
	prefix := "model_providers." + provider + "."
	if provider == "openai" {
		if entry != nil {
			return nil, provider, failure("invalid_config")
		}
		// 新版 Codex 禁止覆盖内置 openai 表。本地入口使用独立固定前缀，
		// 官方 auth.json 与其登录令牌原样保留，网关按前缀归属而非接入 Key。
		updates["openai_base_url"] = quote(baseURL)
		updates["features.responses_websockets"] = "false"
		updates["features.responses_websockets_v2"] = "false"
		updates["features.enable_request_compression"] = "false"
		result, err := patchTOML(body, provider, updates)
		return result, provider, err
	}
	updates[prefix+"base_url"] = quote(baseURL)
	if entry["name"] == nil {
		updates[prefix+"name"] = quote(provider)
	}
	updates[prefix+"wire_api"] = quote("responses")
	updates[prefix+"requires_openai_auth"] = "false"
	updates[prefix+"supports_websockets"] = "false"
	updates[prefix+"experimental_bearer_token"] = quote(string(key))
	updates[prefix+"env_key"] = ""
	result, err := patchTOML(body, provider, updates)
	return result, provider, err
}
func quote(value string) string { b, _ := json.Marshal(value); return string(b) }
