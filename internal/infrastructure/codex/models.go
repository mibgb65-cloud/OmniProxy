package codex

import (
	"context"
	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/routing"
)

type Model struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Metadata catalog.Metadata `json:"metadata"`
}

// Models 使用当前隔离账号的官方目录，分页上限防止异常服务无限返回。
func (c *rpcClient) Models(ctx context.Context) ([]Model, error) {
	models := []Model{}
	seen := map[string]bool{}
	cursor := ""
	for range 10 {
		var page struct {
			Data []struct {
				Model         string `json:"model"`
				Name          string `json:"displayName"`
				Description   string `json:"description"`
				DefaultEffort string `json:"defaultReasoningEffort"`
				Efforts       []struct {
					Value       string `json:"reasoningEffort"`
					Description string `json:"description"`
				} `json:"supportedReasoningEfforts"`
				ContextWindow   *int64   `json:"contextWindow"`
				InputModalities []string `json:"inputModalities"`
				ToolUse         *bool    `json:"supportsToolUse"`
				IsDefault       bool     `json:"isDefault"`
			} `json:"data"`
			Next *string `json:"nextCursor"`
		}
		params := map[string]any{"includeHidden": false, "limit": 100}
		if cursor != "" {
			params["cursor"] = cursor
		}
		if err := c.call(ctx, "model/list", params, &page); err != nil {
			return nil, err
		}
		if len(page.Data) > 100 {
			return nil, ErrProtocol
		}
		for _, item := range page.Data {
			if !routing.ValidModel(item.Model) || len(item.Name) > 200 {
				return nil, ErrProtocol
			}
			if !seen[item.Model] {
				metadata := catalog.Metadata{Description: item.Description, ContextWindow: item.ContextWindow, DefaultEffort: item.DefaultEffort, InputModalities: item.InputModalities, ToolUse: item.ToolUse, IsDefault: item.IsDefault}
				for _, effort := range item.Efforts {
					metadata.ReasoningEfforts = append(metadata.ReasoningEfforts, catalog.Effort{Value: effort.Value, Description: effort.Description})
				}
				if len(metadata.ReasoningEfforts) == 0 {
					metadata.DefaultEffort = ""
				}
				if catalog.ValidateMetadata(metadata) != nil {
					return nil, ErrProtocol
				}
				models = append(models, Model{ID: item.Model, Name: item.Name, Metadata: metadata})
				seen[item.Model] = true
			}
		}
		if page.Next == nil || *page.Next == "" {
			return models, nil
		}
		if len(*page.Next) > 1024 || *page.Next == cursor {
			return nil, ErrProtocol
		}
		cursor = *page.Next
	}
	return nil, ErrProtocol
}
