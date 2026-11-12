package agentconfig

import (
	"context"
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

// restoreAntigravity 只撤销本应用写入的 gateway 与 models 字段。
//
// 桌面端会在连接期间持续改写 userSettings、sidecars 等自己的字段；整文件摘要
// 变化不能成为拒绝断开的理由，否则用户会被锁在网关模式。识别网关块时只认
// 本机地址与 openai 协议，用户自行改过的块不覆盖。
func (s *Service) restoreAntigravity(ctx context.Context, path, ref, currentHash string) ([]Item, error) {
	data, err := s.store.Get(ref)
	if err != nil {
		return nil, failure("backup_unavailable")
	}
	defer clear(data)
	var original backup
	if json.Unmarshal(data, &original) != nil {
		return nil, failure("backup_unavailable")
	}
	defer clear(original.Body)
	originalBundle, err := decodeBundle(original.Body)
	if err != nil {
		return nil, err
	}
	defer originalBundle.clear()
	current, _, err := s.readClient("antigravity", path)
	if err != nil {
		return nil, err
	}
	defer clear(current)
	currentBundle, err := decodeBundle(current)
	if err != nil {
		return nil, err
	}
	defer currentBundle.clear()
	base, catalogURL := antigravityGatewayBase(s.port), antigravityCatalogURL(s.port)
	for index := range currentBundle.Files {
		body, fileExists, err := mergeAntigravityRestore(currentBundle.Files[index].Body, originalBundle.Files[index], base, catalogURL)
		if err != nil {
			return nil, err
		}
		currentBundle.Files[index] = backup{Body: body, Exists: fileExists}
	}
	encoded, err := encodeBundle(currentBundle)
	if err != nil {
		return nil, err
	}
	defer clear(encoded)
	if err := s.writeClient("antigravity", path, encoded, currentHash); err != nil {
		return nil, err
	}
	if err := s.deleteRestoreRecords(ctx, "antigravity", false); err != nil {
		return nil, err
	}
	if err := s.store.Delete(ref); err != nil {
		return nil, failure("backup_failed")
	}
	if err := s.SetEnabled(ctx, "antigravity", false); err != nil {
		return nil, err
	}
	return s.Inspect(ctx)
}

// mergeAntigravityRestore 把当前文件里的受管字段换回原值；其他字段原样保留。
func mergeAntigravityRestore(current []byte, original backup, base, catalogURL string) ([]byte, bool, error) {
	currentConfig := map[string]any{}
	if len(current) > 0 && strings.TrimSpace(string(current)) != "" {
		if json.Unmarshal(current, &currentConfig) != nil || currentConfig == nil {
			return nil, false, failure("invalid_config")
		}
	}
	originalConfig := map[string]any{}
	if original.Exists && len(original.Body) > 0 {
		if json.Unmarshal(original.Body, &originalConfig) != nil {
			return nil, false, failure("backup_unavailable")
		}
	}
	if gateway, ok := currentConfig["gateway"].(map[string]any); ok {
		if gateway["enabled"] == true && gateway["url"] == base && gateway["wireProtocol"] == "openai" {
			if value, exists := originalConfig["gateway"]; exists {
				currentConfig["gateway"] = value
			} else {
				delete(currentConfig, "gateway")
			}
		}
	}
	if models, ok := currentConfig["models"].(map[string]any); ok {
		originalModels, _ := originalConfig["models"].(map[string]any)
		if models["listSource"] == catalogURL {
			if value, exists := originalModels["listSource"]; exists {
				models["listSource"] = value
			} else {
				delete(models, "listSource")
			}
		}
		if value, exists := originalModels["defaultModelId"]; exists {
			models["defaultModelId"] = value
		} else {
			delete(models, "defaultModelId")
		}
		// 连接时删掉的内联列表按原样放回；连接期间用户新增的列表不覆盖。
		if _, exists := models["list"]; !exists {
			if value, exists := originalModels["list"]; exists {
				models["list"] = value
			}
		}
		if len(models) == 0 {
			delete(currentConfig, "models")
		} else {
			currentConfig["models"] = models
		}
	}
	if len(currentConfig) == 0 && !original.Exists {
		return []byte{}, false, nil
	}
	encoded, err := json.MarshalIndent(currentConfig, "", "  ")
	if err != nil {
		return nil, false, failure("invalid_config")
	}
	return append(encoded, '\n'), true, nil
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

// antigravityDefault 优先官方默认模型，其次 Gemini，最后目录首项。目录按 ID
// 排序后 Claude 会排在 Gemini 前面；桌面端默认模型仍应优先落在订阅主模型上。
func antigravityDefault(entries []catalog.Entry) string {
	for _, entry := range entries {
		if entry.Metadata.IsDefault {
			return entry.ID
		}
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.ID, "gemini-") {
			return entry.ID
		}
	}
	if len(entries) > 0 {
		return entries[0].ID
	}
	return ""
}

// antigravityEntries 发布 Google 订阅账号当前可用的全部文本模型，包括
// Gemini、Claude 与 GPT-OSS；不引入 Google 之外的提供商。
func antigravityEntries(entries []catalog.Entry) []catalog.Entry {
	result := make([]catalog.Entry, 0, len(entries))
	for _, entry := range entries {
		if entry.ProviderKind != "google" {
			continue
		}
		result = append(result, entry)
	}
	return result
}
