package river

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/core/routing"
	"time"
)

// DiscoverModels 有界读取账号目录，未知模型不自动开放未经验证的适配能力。
func (c *Client) DiscoverModels(ctx context.Context, base string, key []byte) (models []string, result error) {
	address, err := requestURL(base, "/models")
	if err != nil {
		return nil, err
	}
	if !validKey(key) {
		return nil, proxy.Failure("invalid_credential", 503)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, proxy.Failure("models_unavailable", 502)
	}
	req.Header.Set("Authorization", "Bearer "+string(key))
	defer req.Header.Del("Authorization")
	response, err := c.http.Do(req)
	if err != nil {
		return nil, proxy.Failure("models_unavailable", 502)
	}
	defer func() {
		if err := response.Body.Close(); err != nil && result == nil {
			result = proxy.Failure("models_unavailable", 502)
		}
	}()
	if response.StatusCode != 200 {
		return nil, proxy.Failure("models_unavailable", 502)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 131073))
	defer clear(body)
	var page struct {
		Data *[]struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err != nil || len(body) > 131072 || json.Unmarshal(body, &page) != nil || page.Data == nil || len(*page.Data) > 1000 {
		return nil, proxy.Failure("invalid_model_catalog", 502)
	}
	models = []string{}
	seen := map[string]bool{}
	for _, item := range *page.Data {
		if !routing.ValidModel(item.ID) {
			return nil, proxy.Failure("invalid_model_catalog", 502)
		}
		if SupportsModel(item.ID) && !seen[item.ID] {
			models = append(models, item.ID)
			seen[item.ID] = true
		}
	}
	if len(models) == 0 {
		return nil, proxy.Failure("invalid_model_catalog", 502)
	}
	return models, nil
}

func validKey(key []byte) bool {
	if len(key) == 0 || len(key) > 8192 {
		return false
	}
	for _, char := range key {
		if char < 33 || char > 126 {
			return false
		}
	}
	return true
}
