package stepfun

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"omniproxy/internal/core/routing"
	"time"
)

func (c *Client) read(ctx context.Context, o Options, key []byte, path string) ([]byte, error) {
	if !validKey(key) {
		return nil, failure("credential_unavailable")
	}
	base, err := o.BaseURL()
	if err != nil {
		return nil, failure("invalid_configuration")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return nil, failure("network_failed")
	}
	req.Header.Set("Authorization", "Bearer "+string(key))
	defer req.Header.Del("Authorization")
	response, err := c.http.Do(req)
	if err != nil {
		return nil, failure("network_failed")
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, (128<<10)+1))
	closeErr := response.Body.Close()
	if err != nil || closeErr != nil || len(body) > 128<<10 {
		clear(body)
		return nil, failure("invalid_response")
	}
	if response.StatusCode != http.StatusOK {
		clear(body)
		code := "query_failed"
		switch response.StatusCode {
		case 401:
			code = "unauthorized"
		case 403:
			code = "forbidden"
		case 429:
			code = "rate_limited"
		case 500, 502, 503, 504:
			code = "service_unavailable"
		}
		return nil, failure(code)
	}
	return body, nil
}

// DiscoverModels 只查询普通通道，不能用普通目录冒充订阅授权。
func (c *Client) DiscoverModels(ctx context.Context, o Options, key []byte) ([]string, error) {
	if o.BillingMode != "payg" {
		return nil, failure("unsupported")
	}
	body, err := c.read(ctx, o, key, "/models")
	if err != nil {
		return nil, err
	}
	defer clear(body)
	var page struct {
		Data *[]struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &page) != nil || page.Data == nil || len(*page.Data) > 1000 {
		return nil, failure("invalid_response")
	}
	ids := []string{}
	seen := map[string]bool{}
	for _, item := range *page.Data {
		if !routing.ValidModel(item.ID) {
			return nil, failure("invalid_response")
		}
		if Supports(o, item.ID, "chat") && !seen[item.ID] {
			ids = append(ids, item.ID)
			seen[item.ID] = true
		}
	}
	if len(ids) == 0 {
		return nil, failure("invalid_response")
	}
	return ids, nil
}
