package kimi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/core/routing"
	"time"
)

// Model 保存经验证的账号目录及官方上下文；未核对协议的模型不纳入路由。
type Model struct {
	ID       string
	Metadata catalog.Metadata
}

// DiscoverModels 有界发现当前账号的模型，失败不使用别的区域。
func (c *Client) DiscoverModels(ctx context.Context, o Options, key []byte) (models []Model, result error) {
	if o.Mode() != "payg" {
		return nil, proxy.Failure("models_unavailable", 503)
	}
	if !ValidAPIKey(key) {
		return nil, proxy.Failure("invalid_credential", 503)
	}
	base, err := o.BaseURL()
	if err != nil {
		return nil, proxy.Failure("invalid_configuration", 503)
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/models", nil)
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
			ID      string `json:"id"`
			Context *int64 `json:"context_length"`
		} `json:"data"`
	}
	if err != nil || len(body) > 131072 || json.Unmarshal(body, &page) != nil || page.Data == nil || len(*page.Data) > 1000 {
		return nil, proxy.Failure("invalid_model_catalog", 502)
	}
	models = []Model{}
	seen := map[string]bool{}
	for _, item := range *page.Data {
		if !routing.ValidModel(item.ID) || item.Context != nil && (*item.Context < 1 || *item.Context > 10000000) {
			return nil, proxy.Failure("invalid_model_catalog", 502)
		}
		if !SupportsModel(item.ID) || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		m := ModelMetadata(item.ID)
		if item.Context != nil {
			m.ContextWindow = item.Context
		}
		models = append(models, Model{ID: item.ID, Metadata: m})
	}
	if len(models) == 0 {
		return nil, proxy.Failure("invalid_model_catalog", 502)
	}
	return models, nil
}
