package deepseek

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/core/routing"
)

// DiscoverModels 使用固定官方域名，不将凭据发送到可编辑的提供商地址。
func (c *Client) DiscoverModels(ctx context.Context, key []byte) ([]string, error) {
	if len(key) == 0 || len(key) > 8192 {
		return nil, proxy.Failure("credential_unavailable", 503)
	}
	for _, ch := range key {
		if ch < 33 || ch > 126 {
			return nil, proxy.Failure("credential_unavailable", 503)
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, DefaultBaseURL+"/models", nil)
	if err != nil {
		return nil, proxy.Failure("models_unavailable", 502)
	}
	req.Header.Set("Authorization", "Bearer "+string(key))
	defer req.Header.Del("Authorization")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, proxy.Failure("models_unavailable", 502)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, proxy.Failure("models_unavailable", 502)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, (128<<10)+1))
	defer clear(body)
	if err != nil || len(body) > 128<<10 {
		return nil, proxy.Failure("invalid_model_catalog", 502)
	}
	var result struct {
		Data *[]struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &result) != nil || result.Data == nil || len(*result.Data) > 1000 {
		return nil, proxy.Failure("invalid_model_catalog", 502)
	}
	models := []string{}
	seen := map[string]bool{}
	for _, model := range *result.Data {
		if !routing.ValidModel(model.ID) {
			return nil, proxy.Failure("invalid_model_catalog", 502)
		}
		if !seen[model.ID] {
			models = append(models, model.ID)
			seen[model.ID] = true
		}
	}
	return models, nil
}
