package openai

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

const maxCatalogBytes = 128 << 10
const maxCatalogModels = 1000

// DiscoverModels 有界读取账号可见模型；只保留可进入对话／Codex 目录的型号。
func (c *Client) DiscoverModels(ctx context.Context, baseURL string, key []byte) ([]string, error) {
	if len(key) == 0 || len(key) > 8192 {
		return nil, proxy.Failure("credential_unavailable", 503)
	}
	for _, char := range key {
		if char < 33 || char > 126 {
			return nil, proxy.Failure("credential_unavailable", 503)
		}
	}
	endpoint, err := endpointURL(baseURL, "/models")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, proxy.Failure("models_unavailable", 502)
	}
	req.Header.Set("Authorization", "Bearer "+string(key))
	req.Header.Set("Accept", "application/json")
	defer req.Header.Del("Authorization")
	response, err := c.http.Do(req)
	if err != nil {
		return nil, proxy.Failure("models_unavailable", 502)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, proxy.Failure("models_unavailable", 502)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxCatalogBytes+1))
	defer clear(body)
	if err != nil || len(body) > maxCatalogBytes {
		return nil, proxy.Failure("invalid_model_catalog", 502)
	}
	var page struct {
		Data *[]struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &page) != nil || page.Data == nil || len(*page.Data) > maxCatalogModels {
		return nil, proxy.Failure("invalid_model_catalog", 502)
	}
	models := []string{}
	seen := map[string]bool{}
	for _, item := range *page.Data {
		if !routing.ValidModel(item.ID) {
			return nil, proxy.Failure("invalid_model_catalog", 502)
		}
		if !isChatModel(item.ID) || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		models = append(models, item.ID)
	}
	if len(models) == 0 {
		return nil, proxy.Failure("invalid_model_catalog", 502)
	}
	return models, nil
}

// isChatModel 只排除已明确的非对话家族；未知的新对话型号仍保留，避免按名称猜能力。
func isChatModel(id string) bool {
	lower := strings.ToLower(strings.TrimSpace(id))
	chat := strings.HasPrefix(lower, "gpt-") ||
		strings.HasPrefix(lower, "chatgpt-") ||
		strings.HasPrefix(lower, "codex-") ||
		len(lower) > 1 && lower[0] == 'o' && lower[1] >= '1' && lower[1] <= '9'
	if !chat {
		return false
	}
	for _, blocked := range []string{
		"embedding", "tts", "whisper", "dall-e", "image", "audio",
		"realtime", "transcribe", "moderation", "sora", "search",
	} {
		if strings.Contains(lower, blocked) {
			return false
		}
	}
	return true
}

func clear(value []byte) {
	for index := range value {
		value[index] = 0
	}
}
