package google

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	core "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/core/routing"
)

type Model struct {
	ID       string
	Name     string
	Metadata core.Metadata
}

// Models 保存输入／输出的独立限制，不把两者相加或猜测总上下文窗口。
func (c *Client) nativeModels(ctx context.Context, base string, key []byte) ([]Model, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	if !validKey(key) {
		return nil, proxy.Failure("invalid_credential", 503)
	}
	address, err := endpoint(base, "/models")
	if err != nil {
		return nil, err
	}
	models := []Model{}
	seen := map[string]bool{}
	cursor := ""
	for range 10 {
		u, err := url.Parse(address)
		if err != nil {
			return nil, proxy.Failure("invalid_configuration", 503)
		}
		query := u.Query()
		query.Set("pageSize", "100")
		if cursor != "" {
			query.Set("pageToken", cursor)
		}
		u.RawQuery = query.Encode()
		page, err := c.modelPage(ctx, u.String(), key)
		if err != nil {
			return nil, err
		}
		for _, item := range page.Models {
			id := strings.TrimPrefix(item.Name, "models/")
			if !strings.HasPrefix(item.Name, "models/") || !routing.ValidModel(id) || strings.Contains(id, "/") || len(item.DisplayName) > 200 || len(item.Description) > 2000 || len(item.Methods) > 16 {
				return nil, proxy.Failure("invalid_upstream_response", 502)
			}
			generates := false
			for _, method := range item.Methods {
				if method == "generateContent" {
					generates = true
				}
			}
			if !generates || seen[id] {
				continue
			}
			metadata := core.Metadata{Description: item.Description, InputTokenLimit: item.Input, OutputTokenLimit: item.Output, InputModalities: []string{"text"}}
			if core.ValidateMetadata(metadata) != nil {
				return nil, proxy.Failure("invalid_upstream_response", 502)
			}
			models = append(models, Model{ID: id, Name: item.DisplayName, Metadata: metadata})
			seen[id] = true
		}
		if page.Next == "" {
			return models, nil
		}
		if len(page.Next) > 1024 || page.Next == cursor {
			return nil, proxy.Failure("invalid_upstream_response", 502)
		}
		cursor = page.Next
	}
	return nil, proxy.Failure("invalid_upstream_response", 502)
}

type modelPage struct {
	Models []struct {
		Name        string   `json:"name"`
		DisplayName string   `json:"displayName"`
		Description string   `json:"description"`
		Input       *int64   `json:"inputTokenLimit"`
		Output      *int64   `json:"outputTokenLimit"`
		Methods     []string `json:"supportedGenerationMethods"`
	} `json:"models"`
	Next string `json:"nextPageToken"`
}

func (c *Client) modelPage(ctx context.Context, address string, key []byte) (page modelPage, resultError error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return page, proxy.Failure("invalid_configuration", 503)
	}
	req.Header.Set("x-goog-api-key", string(key))
	defer req.Header.Del("x-goog-api-key")
	response, err := c.http.Do(req)
	if err != nil {
		return page, proxy.Failure("upstream_unavailable", 502)
	}
	defer func() {
		if err := response.Body.Close(); err != nil && resultError == nil {
			resultError = proxy.Failure("invalid_upstream_response", 502)
		}
	}()
	if response.StatusCode != 200 {
		return page, proxy.Failure("model_discovery_failed", 502)
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 || json.Unmarshal(raw, &page) != nil || page.Models == nil || len(page.Models) > 100 {
		return page, proxy.Failure("invalid_upstream_response", 502)
	}
	return page, nil
}
