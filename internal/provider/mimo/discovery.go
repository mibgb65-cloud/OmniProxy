package mimo

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"omniproxy/internal/core/proxy"
	"omniproxy/internal/core/routing"
)

// DiscoverModels 查询账号实际目录；未核对的模型和 ASR／TTS 不进入对话路由。
func (c *Client) DiscoverModels(ctx context.Context, base string, key []byte) (models []string, result error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if !validKey(key) {
		return nil, proxy.Failure("invalid_credential", 503)
	}
	address, err := endpoint(base, "/v1/models")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, proxy.Failure("models_unavailable", 502)
	}
	req.Header.Set("Api-Key", string(key))
	defer req.Header.Del("Api-Key")
	response, err := c.http.Do(req)
	if err != nil {
		return nil, proxy.Failure("models_unavailable", 502)
	}
	defer func() {
		if err := response.Body.Close(); err != nil && result == nil {
			result = proxy.Failure("models_unavailable", 502)
		}
	}()
	if response.StatusCode != http.StatusOK {
		return nil, proxy.Failure("models_unavailable", 502)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (128<<10)+1))
	defer clear(body)
	var page struct {
		Data *[]struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err != nil || len(body) > 128<<10 || json.Unmarshal(body, &page) != nil || page.Data == nil || len(*page.Data) > 1000 {
		return nil, proxy.Failure("invalid_model_catalog", 502)
	}
	models = []string{}
	seen := map[string]bool{}
	for _, model := range *page.Data {
		if !routing.ValidModel(model.ID) {
			return nil, proxy.Failure("invalid_model_catalog", 502)
		}
		if SupportsModel(model.ID) && !seen[model.ID] {
			models = append(models, model.ID)
			seen[model.ID] = true
		}
	}
	if len(models) == 0 {
		return nil, proxy.Failure("invalid_model_catalog", 502)
	}
	return models, nil
}
