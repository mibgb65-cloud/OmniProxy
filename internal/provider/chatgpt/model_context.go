package chatgpt

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"omniproxy/internal/core/proxy"
	"omniproxy/internal/core/routing"
)

const maxModelCatalogBytes = 2 << 20

// ModelContextWindows 读取同一订阅身份的原生目录；model/list 的公开投影不包含窗口容量。
// 只采用当前默认 context_window，不能把可选 max_context_window 当作已启用窗口。
func (c *Client) ModelContextWindows(ctx context.Context, key []byte, accountID, version string) (map[string]int64, error) {
	if len(key) == 0 || len(key) > 16384 || strings.ContainsAny(string(key), "\r\n") ||
		accountID == "" || len(accountID) > 128 || !routing.ValidModel(accountID) ||
		!validCatalogVersion(version) {
		return nil, proxy.Failure("models_unavailable", 502)
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	address := "https://chatgpt.com/backend-api/codex/models?client_version=" + url.QueryEscape(version)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, proxy.Failure("models_unavailable", 502)
	}
	req.Header.Set("Authorization", "Bearer "+string(key))
	req.Header.Set("ChatGPT-Account-ID", accountID)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Originator", "codex_cli_rs")
	defer req.Header.Del("Authorization")
	defer req.Header.Del("ChatGPT-Account-ID")
	response, err := c.http.Do(req)
	if err != nil {
		return nil, proxy.Failure("models_unavailable", 502)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, proxy.Failure("models_unavailable", 502)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxModelCatalogBytes+1))
	defer clear(body)
	if err != nil || len(body) > maxModelCatalogBytes {
		return nil, proxy.Failure("invalid_model_catalog", 502)
	}
	return parseModelContextWindows(body)
}

func validCatalogVersion(value string) bool {
	if len(value) == 0 || len(value) > 64 || value[0] < '0' || value[0] > '9' {
		return false
	}
	for _, char := range value {
		if !(char >= '0' && char <= '9' || char >= 'a' && char <= 'z' || char == '.' || char == '-') {
			return false
		}
	}
	return true
}

func parseModelContextWindows(body []byte) (map[string]int64, error) {
	var page struct {
		Models *[]struct {
			ID     string `json:"slug"`
			Window *int64 `json:"context_window"`
		} `json:"models"`
	}
	if json.Unmarshal(body, &page) != nil || page.Models == nil || len(*page.Models) == 0 || len(*page.Models) > 1000 {
		return nil, proxy.Failure("invalid_model_catalog", 502)
	}
	windows, seen := map[string]int64{}, map[string]bool{}
	for _, model := range *page.Models {
		if !routing.ValidModel(model.ID) || seen[model.ID] {
			return nil, proxy.Failure("invalid_model_catalog", 502)
		}
		seen[model.ID] = true
		if model.Window == nil {
			continue
		}
		if *model.Window < 1 || *model.Window > 10000000 {
			return nil, proxy.Failure("invalid_model_catalog", 502)
		}
		windows[model.ID] = *model.Window
	}
	return windows, nil
}
