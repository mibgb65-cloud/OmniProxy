package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"omniproxy/internal/config"
	"omniproxy/internal/proxy"
	"omniproxy/internal/token"
)

type providerModelCatalogRequest struct {
	Provider string `json:"provider"`
}

type providerModelCatalogItem struct {
	ID            string `json:"id"`
	Name          string `json:"name,omitempty"`
	ContextLength int    `json:"contextLength,omitempty"`
}

type providerModelCatalogResponse struct {
	Provider  string                     `json:"provider"`
	Source    string                     `json:"source"`
	BaseURL   string                     `json:"baseUrl,omitempty"`
	TokenName string                     `json:"tokenName,omitempty"`
	FetchedAt string                     `json:"fetchedAt,omitempty"`
	Models    []providerModelCatalogItem `json:"models"`
}

func (a *appServer) providerModels(ctx context.Context, req providerModelCatalogRequest) (providerModelCatalogResponse, error) {
	provider, _, err := token.NormalizeProviderAndCredential(req.Provider, "")
	if err != nil {
		return providerModelCatalogResponse{}, err
	}
	if provider == token.ProviderOpenRouter {
		result, err := a.openRouterModels(ctx, true)
		if err != nil {
			return providerModelCatalogResponse{Provider: provider, Source: provider}, err
		}
		return providerModelCatalogResponse{
			Provider:  provider,
			Source:    result.Source,
			FetchedAt: result.FetchedAt,
			Models:    openRouterCatalogItems(result.Models),
		}, nil
	}

	a.mu.Lock()
	cfg := config.Normalize(a.cfg)
	a.mu.Unlock()
	selected, err := a.providerModelToken(provider)
	if err != nil {
		return providerModelCatalogResponse{Provider: provider, Source: provider}, err
	}
	baseURL := providerModelBaseURL(cfg, selected)
	if strings.TrimSpace(baseURL) == "" {
		return providerModelCatalogResponse{Provider: provider, Source: provider}, errors.New("provider base url is empty")
	}
	client, err := proxy.NewTokenHTTPClient(cfg, selected, 20*time.Second)
	if err != nil {
		return providerModelCatalogResponse{Provider: provider, Source: provider}, err
	}

	var models []providerModelCatalogItem
	if provider == token.ProviderGemini {
		models, err = fetchGeminiCatalogModels(ctx, client, baseURL, selected.TokenValue)
	} else {
		models, err = fetchOpenAICompatibleCatalogModels(ctx, client, provider, baseURL, selected.TokenValue)
	}
	if err != nil {
		return providerModelCatalogResponse{Provider: provider, Source: provider, BaseURL: baseURL}, err
	}
	return providerModelCatalogResponse{
		Provider:  provider,
		Source:    provider,
		BaseURL:   baseURL,
		TokenName: a.tokenDisplayName(selected),
		FetchedAt: timeString(time.Now()),
		Models:    models,
	}, nil
}

func (a *appServer) providerModelToken(provider string) (token.Token, error) {
	if a.tokens == nil {
		return token.Token{}, errors.New("token manager is not ready")
	}
	var fallback token.Token
	for _, item := range a.tokens.List() {
		if token.NormalizeProvider(item.Provider) != provider {
			continue
		}
		if strings.TrimSpace(item.TokenValue) == "" || !catalogCredentialSupported(item) {
			continue
		}
		if fallback.ID == "" {
			fallback = item
		}
		if !item.Disabled {
			return item, nil
		}
	}
	if fallback.ID != "" {
		return fallback, nil
	}
	return token.Token{}, fmt.Errorf("请先添加可用于同步模型目录的 %s 凭据", provider)
}

func catalogCredentialSupported(item token.Token) bool {
	switch item.CredentialType {
	case "", token.CredentialTypeAPIKey, token.CredentialTypeMimoTokenPlan, token.CredentialTypeCodingPlan:
		return true
	default:
		return false
	}
}

func providerModelBaseURL(cfg config.Config, selected token.Token) string {
	if strings.TrimSpace(selected.BaseURL) != "" {
		return selected.BaseURL
	}
	switch token.NormalizeProvider(selected.Provider) {
	case token.ProviderOpenAI:
		return cfg.OpenAIBaseURL
	case token.ProviderAnthropic:
		return cfg.AnthropicBaseURL
	case token.ProviderDeepSeek:
		return cfg.DeepSeekBaseURL
	case token.ProviderKimi:
		return cfg.KimiBaseURL
	case token.ProviderXiaomi:
		if selected.CredentialType == token.CredentialTypeMimoTokenPlan {
			return xiaomiModelBaseURL(cfg, selected.Region)
		}
		return cfg.XiaomiAPIBaseURL
	case token.ProviderZhipu:
		return cfg.ZhipuBaseURL
	case token.ProviderMiniMax:
		return cfg.MiniMaxBaseURL
	case token.ProviderGemini:
		return cfg.GeminiBaseURL
	case token.ProviderTokenRouter:
		return cfg.TokenRouterBaseURL
	case token.ProviderFeatherless:
		return cfg.FeatherlessBaseURL
	case token.ProviderSub2API:
		return cfg.Sub2APIBaseURL
	case token.ProviderNewAPI:
		return cfg.NewAPIBaseURL
	case token.ProviderAnyRouter:
		return cfg.AnyRouterBaseURL
	case token.ProviderZo:
		return cfg.ZoBaseURL
	case token.ProviderPrem:
		return cfg.PremBaseURL
	case token.ProviderAtria:
		return cfg.AtriaBaseURL
	case token.ProviderCustom:
		return cfg.CustomGatewayBaseURL
	default:
		return cfg.OpenAIBaseURL
	}
}

func xiaomiModelBaseURL(cfg config.Config, region string) string {
	switch strings.ToLower(strings.TrimSpace(region)) {
	case token.MimoRegionSGP:
		return cfg.XiaomiTokenPlanSGPBaseURL
	case token.MimoRegionAMS:
		return cfg.XiaomiTokenPlanAMSBaseURL
	default:
		return cfg.XiaomiTokenPlanBaseURL
	}
}

func fetchOpenAICompatibleCatalogModels(ctx context.Context, client *http.Client, provider string, baseURL string, apiKey string) ([]providerModelCatalogItem, error) {
	target, err := joinExternalURLPath(baseURL, "/models")
	if err != nil {
		return nil, err
	}
	if provider == token.ProviderFeatherless {
		parsed, err := url.Parse(target)
		if err != nil {
			return nil, err
		}
		query := parsed.Query()
		query.Set("available_on_current_plan", "true")
		query.Set("status", "active")
		query.Set("sort", "-popularity")
		query.Set("per_page", "1000")
		parsed.RawQuery = query.Encode()
		target = parsed.String()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if provider == token.ProviderAnthropic {
		req.Header.Set("X-API-Key", strings.TrimSpace(apiKey))
		req.Header.Set("Anthropic-Version", "2023-06-01")
	} else if secret := strings.TrimSpace(apiKey); secret != "" {
		req.Header.Set("Authorization", "Bearer "+secret)
	}
	return doCatalogModelsRequest(client, req)
}

func fetchGeminiCatalogModels(ctx context.Context, client *http.Client, baseURL string, apiKey string) ([]providerModelCatalogItem, error) {
	target, err := joinExternalURLPath(baseURL, "/v1beta/models")
	if err != nil {
		return nil, err
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	query := parsed.Query()
	query.Set("key", strings.TrimSpace(apiKey))
	parsed.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	return doCatalogModelsRequest(client, req)
}

func doCatalogModelsRequest(client *http.Client, req *http.Request) ([]providerModelCatalogItem, error) {
	if client == nil {
		client = http.DefaultClient
	}
	ctx, cancel := context.WithTimeout(req.Context(), 20*time.Second)
	defer cancel()
	req = req.WithContext(ctx)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("models request returned %d: %s", resp.StatusCode, limitOpenRouterError(body))
	}
	models, err := parseProviderCatalogModels(body)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(models, func(i, j int) bool {
		return strings.ToLower(models[i].ID) < strings.ToLower(models[j].ID)
	})
	return models, nil
}

func parseProviderCatalogModels(body []byte) ([]providerModelCatalogItem, error) {
	var payload map[string]any
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	if err := decoder.Decode(&payload); err != nil {
		return nil, err
	}
	rawItems := sliceField(payload["data"])
	if len(rawItems) == 0 {
		rawItems = sliceField(payload["models"])
	}
	models := make([]providerModelCatalogItem, 0, len(rawItems))
	for _, raw := range rawItems {
		item := providerCatalogItem(raw)
		if item.ID != "" {
			models = append(models, item)
		}
	}
	return models, nil
}

func providerCatalogItem(raw any) providerModelCatalogItem {
	if text := stringField(raw); text != "" {
		return providerModelCatalogItem{ID: strings.TrimPrefix(text, "models/")}
	}
	object := objectField(raw)
	id := firstStringField(object, "id", "name", "model")
	id = strings.TrimPrefix(id, "models/")
	return providerModelCatalogItem{
		ID:            id,
		Name:          firstStringField(object, "display_name", "displayName", "name", "id"),
		ContextLength: firstIntField(object, "context_length", "contextLength", "input_token_limit", "inputTokenLimit"),
	}
}

func openRouterCatalogItems(models []openRouterModelResponse) []providerModelCatalogItem {
	out := make([]providerModelCatalogItem, 0, len(models))
	for _, model := range models {
		if strings.TrimSpace(model.ID) == "" {
			continue
		}
		out = append(out, providerModelCatalogItem{
			ID:            model.ID,
			Name:          model.Name,
			ContextLength: model.ContextLength,
		})
	}
	return out
}

func sliceField(value any) []any {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	return items
}

func firstStringField(object map[string]any, keys ...string) string {
	for _, key := range keys {
		if text := stringField(object[key]); text != "" {
			return text
		}
	}
	return ""
}

func firstIntField(object map[string]any, keys ...string) int {
	for _, key := range keys {
		if value := intField(object[key]); value > 0 {
			return value
		}
	}
	return 0
}
