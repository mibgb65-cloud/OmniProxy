package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"omniproxy/internal/token"
	"strings"
	"time"
)

const zoModelCacheTTL = 10 * time.Minute

type zoModelCacheEntry struct {
	models    []zoModel
	fetchedAt time.Time
}

type zoModel struct {
	ModelName string `json:"model_name"`
	Label     string `json:"label"`
	Vendor    string `json:"vendor"`
}

type zoModelsPayload struct {
	Models []zoModel `json:"models"`
}

type zoAskRequest struct {
	Input        string `json:"input"`
	Stream       bool   `json:"stream"`
	ModelName    string `json:"model_name,omitempty"`
	OutputFormat any    `json:"output_format,omitempty"`
}

type zoAskResponse struct {
	Output any `json:"output"`
}

type zoOpenAIRequest struct {
	Model     string `json:"model"`
	Messages  []any  `json:"messages"`
	Stream    bool   `json:"stream"`
	Tools     []any  `json:"tools"`
	Functions []any  `json:"functions"`
}

type zoAnthropicRequest struct {
	Model    string `json:"model"`
	System   any    `json:"system"`
	Messages []any  `json:"messages"`
	Stream   bool   `json:"stream"`
	Tools    []any  `json:"tools"`
}

type zoResponsesRequest struct {
	Model        string `json:"model"`
	Input        any    `json:"input"`
	Instructions any    `json:"instructions"`
	Stream       any    `json:"stream"`
	Tools        any    `json:"tools"`
}

type zoParsedOutput struct {
	Text      string
	ToolCalls []zoToolCall
}

type zoToolCall struct {
	Name      string
	Arguments map[string]any
}

func (s *Service) forwardZo(ctx context.Context, original *http.Request, route routeInfo, body []byte, selected token.Token) (*http.Response, error) {
	path := normalizeZoClientPath(route.Path)
	switch {
	case original.Method == http.MethodGet && path == "/v1/models":
		return s.forwardZoModels(ctx, original, route, selected)
	case original.Method == http.MethodPost && path == "/v1/chat/completions":
		return s.forwardZoOpenAIChat(ctx, original, route, body, selected)
	case original.Method == http.MethodPost && path == "/v1/messages":
		return s.forwardZoAnthropicMessages(ctx, original, route, body, selected)
	case original.Method == http.MethodPost && path == "/v1/responses":
		return s.forwardZoResponses(ctx, original, route, body, selected)
	default:
		return zoErrorResponse(original, http.StatusNotFound, fmt.Sprintf("Not found: %s %s", original.Method, route.Path), route.Protocol), nil
	}
}

func normalizeZoClientPath(path string) string {
	path = strings.TrimSpace(path)
	switch path {
	case "", "/":
		return "/"
	case "/v1/v1/messages", "/messages":
		return "/v1/messages"
	case "/chat/completions":
		return "/v1/chat/completions"
	case "/models":
		return "/v1/models"
	case "/responses":
		return "/v1/responses"
	default:
		return path
	}
}

func (s *Service) forwardZoModels(ctx context.Context, original *http.Request, route routeInfo, selected token.Token) (*http.Response, error) {
	models, status, header, body, err := s.fetchZoModels(ctx, route, selected, false)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return zoUpstreamErrorResponse(original, status, header, body, route.Protocol), nil
	}

	data := make([]map[string]any, 0, len(models))
	for _, model := range models {
		id := strings.TrimSpace(model.ModelName)
		if id == "" {
			id = strings.TrimSpace(model.Label)
		}
		if id == "" {
			continue
		}
		ownedBy := strings.TrimSpace(model.Vendor)
		if ownedBy == "" {
			ownedBy = "zo"
		}
		data = append(data, map[string]any{
			"id":       id,
			"object":   "model",
			"created":  zoTimestamp(),
			"owned_by": ownedBy,
		})
	}
	return jsonResponse(original, http.StatusOK, map[string]any{
		"object": "list",
		"data":   data,
	}, nil), nil
}

func (s *Service) forwardZoOpenAIChat(ctx context.Context, original *http.Request, route routeInfo, body []byte, selected token.Token) (*http.Response, error) {
	var reqBody zoOpenAIRequest
	if err := json.Unmarshal(body, &reqBody); err != nil {
		return zoErrorResponse(original, http.StatusBadRequest, "Invalid JSON body", "openai"), nil
	}

	models, _, _, _, _ := s.fetchZoModels(ctx, route, selected, true)
	input := buildZoInputFromOpenAI(reqBody.Messages)
	tools := reqBody.Tools
	if len(tools) == 0 {
		tools = reqBody.Functions
	}
	finalInput, outputFormat := injectZoTools(input, tools)

	zoBody := zoAskRequest{Input: finalInput, Stream: false}
	if model := mapZoModel(reqBody.Model, models); model != "" {
		zoBody.ModelName = model
	}
	if outputFormat != nil {
		zoBody.OutputFormat = outputFormat
	}

	status, header, responseBody, err := s.postZoAsk(ctx, route, selected, zoBody, original.Header)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return zoUpstreamErrorResponse(original, status, header, responseBody, "openai"), nil
	}

	var zoResp zoAskResponse
	if err := json.Unmarshal(responseBody, &zoResp); err != nil {
		return zoErrorResponse(original, http.StatusBadGateway, "Invalid Zo API response", "openai"), nil
	}
	parsed := normalizeZoParsedOutput(parseZoOutput(zoResp.Output), tools)
	if reqBody.Stream {
		return openAIStreamResponse(original, reqBody.Model, parsed, header), nil
	}
	return jsonResponse(original, http.StatusOK, openAIResponseFromZo(reqBody.Model, parsed), forwardedZoHeaders(header)), nil
}

func (s *Service) forwardZoAnthropicMessages(ctx context.Context, original *http.Request, route routeInfo, body []byte, selected token.Token) (*http.Response, error) {
	var reqBody zoAnthropicRequest
	if err := json.Unmarshal(body, &reqBody); err != nil {
		return zoErrorResponse(original, http.StatusBadRequest, "Invalid JSON body", "anthropic"), nil
	}

	models, _, _, _, _ := s.fetchZoModels(ctx, route, selected, true)
	input := buildZoInputFromAnthropic(reqBody.System, reqBody.Messages)
	finalInput, outputFormat := injectZoTools(input, reqBody.Tools)

	zoBody := zoAskRequest{Input: finalInput, Stream: false}
	if model := mapZoModel(reqBody.Model, models); model != "" {
		zoBody.ModelName = model
	}
	if outputFormat != nil {
		zoBody.OutputFormat = outputFormat
	}

	status, header, responseBody, err := s.postZoAsk(ctx, route, selected, zoBody, original.Header)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return zoUpstreamErrorResponse(original, status, header, responseBody, "anthropic"), nil
	}

	var zoResp zoAskResponse
	if err := json.Unmarshal(responseBody, &zoResp); err != nil {
		return zoErrorResponse(original, http.StatusBadGateway, "Invalid Zo API response", "anthropic"), nil
	}
	parsed := normalizeZoParsedOutput(parseZoOutput(zoResp.Output), reqBody.Tools)
	if reqBody.Stream {
		return anthropicStreamResponse(original, reqBody.Model, parsed, header), nil
	}
	return jsonResponse(original, http.StatusOK, anthropicResponseFromZo(reqBody.Model, parsed), forwardedZoHeaders(header)), nil
}

func (s *Service) forwardZoResponses(ctx context.Context, original *http.Request, route routeInfo, body []byte, selected token.Token) (*http.Response, error) {
	var reqBody zoResponsesRequest
	if err := json.Unmarshal(body, &reqBody); err != nil {
		return zoErrorResponse(original, http.StatusBadRequest, "Invalid JSON body", "openai"), nil
	}

	models, _, _, _, _ := s.fetchZoModels(ctx, route, selected, true)
	input := buildZoInputFromResponses(reqBody.Instructions, reqBody.Input)
	tools := zoToolsList(reqBody.Tools)
	finalInput, outputFormat := injectZoTools(input, tools)

	zoBody := zoAskRequest{Input: finalInput, Stream: false}
	if model := mapZoModel(reqBody.Model, models); model != "" {
		zoBody.ModelName = model
	}
	if outputFormat != nil {
		zoBody.OutputFormat = outputFormat
	}

	status, header, responseBody, err := s.postZoAsk(ctx, route, selected, zoBody, original.Header)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return zoUpstreamErrorResponse(original, status, header, responseBody, "openai"), nil
	}

	var zoResp zoAskResponse
	if err := json.Unmarshal(responseBody, &zoResp); err != nil {
		return zoErrorResponse(original, http.StatusBadGateway, "Invalid Zo API response", "openai"), nil
	}
	parsed := normalizeZoParsedOutput(parseZoOutput(zoResp.Output), tools)
	if truthyZoBool(reqBody.Stream) {
		return openAIResponsesStreamResponse(original, reqBody.Model, parsed, header), nil
	}
	return jsonResponse(original, http.StatusOK, openAIResponsesFromZo(reqBody.Model, parsed), forwardedZoHeaders(header)), nil
}

func (s *Service) fetchZoModels(ctx context.Context, route routeInfo, selected token.Token, allowCached bool) ([]zoModel, int, http.Header, []byte, error) {
	baseURL := s.router.BaseURL(route, selected)
	if baseURL == "" {
		return nil, 0, nil, nil, fmt.Errorf("%s upstream base url is not configured", route.Provider)
	}
	cacheKey := strings.TrimRight(baseURL, "/") + "|" + selected.ID
	if allowCached {
		s.zoModelsMu.Lock()
		if s.zoModelsCache != nil {
			if entry, ok := s.zoModelsCache[cacheKey]; ok && time.Since(entry.fetchedAt) < zoModelCacheTTL {
				models := append([]zoModel(nil), entry.models...)
				s.zoModelsMu.Unlock()
				return models, http.StatusOK, http.Header{}, nil, nil
			}
		}
		s.zoModelsMu.Unlock()
	}

	status, header, body, err := s.zoFetch(ctx, http.MethodGet, route, selected, "/models/available", nil, nil)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	if status != http.StatusOK {
		return nil, status, header, body, nil
	}
	var payload zoModelsPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, http.StatusBadGateway, header, body, nil
	}
	models := append([]zoModel(nil), payload.Models...)

	s.zoModelsMu.Lock()
	if s.zoModelsCache == nil {
		s.zoModelsCache = map[string]zoModelCacheEntry{}
	}
	s.zoModelsCache[cacheKey] = zoModelCacheEntry{models: models, fetchedAt: time.Now()}
	s.zoModelsMu.Unlock()

	return models, status, header, body, nil
}

func (s *Service) postZoAsk(ctx context.Context, route routeInfo, selected token.Token, body zoAskRequest, clientHeader http.Header) (int, http.Header, []byte, error) {
	return s.zoFetch(ctx, http.MethodPost, route, selected, "/zo/ask", body, clientHeader)
}

func (s *Service) zoFetch(ctx context.Context, method string, route routeInfo, selected token.Token, path string, body any, clientHeader http.Header) (int, http.Header, []byte, error) {
	target, err := joinZoURL(s.router.BaseURL(route, selected), path)
	if err != nil {
		return 0, nil, nil, err
	}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, nil, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return 0, nil, nil, err
	}
	secret, err := credentialSecret(selected)
	if err != nil {
		return 0, nil, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+secret)
	req.Header.Set("Content-Type", "application/json")
	if clientHeader != nil {
		if conversationID := strings.TrimSpace(clientHeader.Get("x-conversation-id")); conversationID != "" {
			req.Header.Set("x-conversation-id", conversationID)
		}
	}

	resp, err := s.clientForRoute(route).Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer closeBody(resp.Body)
	data, err := readProxyResponseBody(resp.Body)
	if err != nil {
		return resp.StatusCode, resp.Header, nil, err
	}
	return resp.StatusCode, resp.Header, data, nil
}
