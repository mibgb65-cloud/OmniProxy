package google

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"omniproxy/internal/core/proxy"
	"omniproxy/internal/core/routing"
)

// Models 以兼容接口目录为可调用 ID 来源，原生目录只补充明确公布的限制。
// 目录交集保留文本生成方法，但目录可见不表示特定 Key 的推理权限已验证。
func (c *Client) Models(ctx context.Context, base string, key []byte) ([]Model, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if !validKey(key) {
		return nil, proxy.Failure("invalid_credential", 503)
	}
	ids, err := c.compatibleModels(ctx, base, key)
	if err != nil {
		return nil, err
	}
	native, err := c.nativeModels(ctx, base, key)
	if err != nil {
		return nil, err
	}
	result := []Model{}
	for _, model := range native {
		if ids[model.ID] {
			result = append(result, model)
		}
	}
	return result, nil
}

func (c *Client) compatibleModels(ctx context.Context, base string, key []byte) (result map[string]bool, resultError error) {
	address, err := endpoint(base, "/openai/models")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, proxy.Failure("invalid_configuration", 503)
	}
	req.Header.Set("Authorization", "Bearer "+string(key))
	defer req.Header.Del("Authorization")
	response, err := c.http.Do(req)
	if err != nil {
		return nil, proxy.Failure("upstream_unavailable", 502)
	}
	defer func() {
		if err := response.Body.Close(); err != nil && resultError == nil {
			resultError = proxy.Failure("invalid_upstream_response", 502)
		}
	}()
	if response.StatusCode != 200 {
		return nil, proxy.Failure("model_discovery_failed", 502)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	var list struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		HasMore bool `json:"has_more"`
	}
	if err != nil || len(raw) > 1<<20 || json.Unmarshal(raw, &list) != nil || list.Data == nil || len(list.Data) > 1000 || list.HasMore {
		return nil, proxy.Failure("invalid_upstream_response", 502)
	}
	result = map[string]bool{}
	for _, model := range list.Data {
		// 官方兼容目录也可能返回 models/ 资源名前缀，Chat 使用相同资源的裸 ID。
		id := strings.TrimPrefix(model.ID, "models/")
		if !routing.ValidModel(id) || strings.Contains(id, "/") {
			return nil, proxy.Failure("invalid_upstream_response", 502)
		}
		result[id] = true
	}
	return result, nil
}
