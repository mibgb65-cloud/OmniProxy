package agentconfig

import (
	"encoding/json"
	"fmt"
	"strings"

	catalog "omniproxy/internal/core/modelcatalog"
)

// antigravityGatewayBase 是桌面端读取的网关根地址。桌面端自己追加
// /v1/chat/completions 与 /v1/models?format=antigravity，不能用路径前缀。
func antigravityGatewayBase(port int) string {
	return fmt.Sprintf("http://127.0.0.1:%d", port)
}

// antigravityCatalogURL 是桌面端 models.listSource 的地址。
func antigravityCatalogURL(port int) string {
	return antigravityGatewayBase(port) + "/v1/models?format=antigravity"
}

// buildAntigravity 把两个版本的设置文件一起写成桌面端的 Enterprise Gateway。
// 用户原有 gateway 的其他字段（例如自定义请求头）保留；内联 models.list 会遮住
// 远端目录，连接期间删除，断开时由备份恢复。
func (s *Service) buildAntigravity(body, key []byte, plan Plan) ([]byte, string, error) {
	bundle, err := decodeBundle(body)
	if err != nil {
		return nil, "", err
	}
	defer bundle.clear()
	base := antigravityGatewayBase(s.port)
	for index := range bundle.Files {
		config, err := antigravityFileConfig(bundle.Files[index].Body, base, string(key), plan.DefaultModel, s.port)
		if err != nil {
			return nil, "", err
		}
		bundle.Files[index].Body = config
		bundle.Files[index].Exists = true
	}
	encoded, err := encodeBundle(bundle)
	if err != nil {
		return nil, "", err
	}
	return encoded, "omniproxy", nil
}

// antigravityFileConfig 只改网关与模型选择，其他设置原样保留。
func antigravityFileConfig(body []byte, base, key, model string, port int) ([]byte, error) {
	config := map[string]any{}
	if len(body) > 0 && strings.TrimSpace(string(body)) != "" {
		if json.Unmarshal(body, &config) != nil || config == nil {
			return nil, failure("invalid_config")
		}
	}
	gateway, ok := config["gateway"].(map[string]any)
	if config["gateway"] != nil && !ok {
		return nil, failure("invalid_config")
	}
	if gateway == nil {
		gateway = map[string]any{}
	}
	gateway["enabled"] = true
	gateway["url"] = base
	gateway["apiKey"] = key
	gateway["wireProtocol"] = "openai"
	config["gateway"] = gateway

	models, ok := config["models"].(map[string]any)
	if config["models"] != nil && !ok {
		return nil, failure("invalid_config")
	}
	if models == nil {
		models = map[string]any{}
	}
	models["defaultModelId"] = model
	models["listSource"] = antigravityCatalogURL(port)
	delete(models, "list")
	config["models"] = models

	encoded, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, failure("invalid_config")
	}
	return append(encoded, '\n'), nil
}

// antigravityConfiguredModel 读取当前设置文件里的默认模型，供预览沿用。
func antigravityConfiguredModel(body []byte) string {
	bundle, err := decodeBundle(body)
	if err != nil || len(bundle.Files) == 0 {
		return ""
	}
	defer bundle.clear()
	var config struct {
		Models struct {
			DefaultModelID string `json:"defaultModelId"`
		} `json:"models"`
	}
	if json.Unmarshal(bundle.Files[0].Body, &config) != nil {
		return ""
	}
	return config.Models.DefaultModelID
}

// antigravityEntries 只发布 Google 订阅账号当前可用的 Gemini 文本模型。
// 桌面端与网关的模型 ID 必须一致；别名沿用它实际指向的模型判断。
func antigravityEntries(entries []catalog.Entry) []catalog.Entry {
	result := make([]catalog.Entry, 0, len(entries))
	for _, entry := range entries {
		model := entry.ID
		if entry.TargetModel != "" {
			model = entry.TargetModel
		}
		if !strings.HasPrefix(model, "gemini-") || entry.ProviderKind != "google" {
			continue
		}
		result = append(result, entry)
	}
	return result
}
