package agentconfig

import (
	"encoding/json"
	"fmt"

	"omniproxy/internal/core/agentendpoint"
)

func buildGatewayAgent(client string, body, key []byte, port int, plan Plan, owned, kimiLegacy bool) ([]byte, string, error) {
	root, ok := agentendpoint.BaseURL(client, port)
	if !ok {
		return nil, "omniproxy", failure("invalid_client")
	}
	root += "/v1"
	var result []byte
	var err error
	switch client {
	case "kimi-code":
		result, err = buildKimiCode(body, key, root, plan, owned, kimiLegacy)
	case "pi":
		result, err = buildPi(body, key, root, plan, owned)
	case "factory-droid":
		result, err = buildFactoryDroid(body, key, root, plan, owned)
	case "goose":
		result, err = buildGoose(body, key, root, plan, owned)
	default:
		err = failure("invalid_client")
	}
	if len(result) > maxConfigBytes {
		clear(result)
		return nil, "", failure("unsafe_config")
	}
	return result, "omniproxy", err
}

func encodeConfig(config map[string]any) ([]byte, error) {
	body, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, failure("invalid_config")
	}
	return append(body, '\n'), nil
}

func gatewayAgentSummary(client, path, model string, count int) string {
	summary := fmt.Sprintf("配置文件：%s\n接入：本地网关 Chat API\n可用模型／路由：%d 个\n默认模型：%s\n保留其他提供商、权限和会话；应用前保存加密备份。", path, count, model)
	if isBundledClient(client) {
		summary += "\n关联文件：" + bundlePaths(client, path)[1] + "\n两个文件一起校验、备份和恢复。"
	}
	return summary
}

func (s *Service) gatewayAgentIdentity(client, path string) string {
	if !isGatewayAgent(client) {
		return ""
	}
	// 相同字节可能来自另一数据目录或协议版本；旧预览不能静默应用到不同配置上下文。
	return fmt.Sprintf("%s:%t", path, client == "kimi-code" && s.paths.KimiCodeLegacy)
}
