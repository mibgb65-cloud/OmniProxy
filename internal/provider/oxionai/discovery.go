package oxionai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/core/routing"
)

// Model 是官方 /v1/models 中可进入网关目录的一条记录。
type Model struct {
	ID       string
	Name     string
	Metadata catalog.Metadata
}

// DiscoverModels 有界读取账号目录。只接受 type=chat 的模型：官方另有嵌入类模型，
// 网关不代理 /embeddings，不能把它们放进对话目录。
func (c *Client) DiscoverModels(ctx context.Context, base string, key []byte) (models []Model, result error) {
	address, err := endpointURL(base, "/models")
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
	req.Header.Set("Accept", "application/json")
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
			ID            string   `json:"id"`
			Name          string   `json:"name"`
			Type          string   `json:"type"`
			ContextWindow int64    `json:"context_window"`
			Capabilities  []string `json:"capabilities"`
		} `json:"data"`
	}
	if json.Unmarshal(body, &page) != nil || page.Data == nil || len(*page.Data) > maxCatalogModels {
		return nil, proxy.Failure("invalid_model_catalog", 502)
	}
	models = []Model{}
	seen := map[string]bool{}
	for _, item := range *page.Data {
		if item.Type != "" && item.Type != "chat" {
			continue
		}
		if !routing.ValidModel(item.ID) || seen[item.ID] || len(models) >= maxCatalogModels {
			if !routing.ValidModel(item.ID) {
				return nil, proxy.Failure("invalid_model_catalog", 502)
			}
			continue
		}
		seen[item.ID] = true
		models = append(models, Model{ID: item.ID, Name: displayName(item.ID, item.Name), Metadata: metadata(item.ContextWindow, item.Capabilities)})
	}
	if len(models) == 0 {
		return nil, proxy.Failure("invalid_model_catalog", 502)
	}
	return models, nil
}

// displayName 保留官方名称，但受目录名称边界约束。
func displayName(id, name string) string {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 200 {
		return id
	}
	return name
}

// metadata 只映射官方明确公布的能力：unknown 保持空值，不假设支持工具或图片。
func metadata(window int64, capabilities []string) catalog.Metadata {
	result := catalog.Metadata{}
	if window > 0 && window <= 10000000 {
		result.ContextWindow = &window
	}
	if len(capabilities) > 0 {
		modalities := []string{"text"}
		tools := false
		for _, capability := range capabilities {
			if capability == "vision" {
				modalities = append(modalities, "image")
			}
			if capability == "tools" {
				tools = true
			}
		}
		result.InputModalities = modalities
		result.ToolUse = &tools
	}
	return result
}
