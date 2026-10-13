package quotaactivation

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"strings"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/subscription"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/codex"
	"omniproxy/internal/provider/chatgpt"
)

// Access 将原账号模型目录与短期令牌使用限制在已有订阅服务内。
type Access interface {
	ReadQuota(context.Context, string) (subscription.Quota, error)
	Models(context.Context, string) ([]codex.Model, error)
	UseToken(context.Context, string, func([]byte, string) error) error
}

// MessageSender 复用隔离授权和固定官方端点，绝不通过路由选择另一账号。
type MessageSender struct {
	accounts  Accounts
	providers Providers
	access    Access
	client    *chatgpt.Client
}

// NewMessageSender 可注入隔离 HTTP 传输验证，正式路径使用现有官方请求适配器。
func NewMessageSender(accounts Accounts, providers Providers, access Access, client *chatgpt.Client) *MessageSender {
	if client == nil {
		client = chatgpt.New(nil)
	}
	return &MessageSender{accounts, providers, access, client}
}

// Send 发送前复查额度与身份，只请求当前账号官方目录中的模型。
func (s *MessageSender) Send(ctx context.Context, expected account.Item) error {
	fresh, err := s.access.ReadQuota(ctx, expected.ID)
	if err != nil {
		return errors.Join(ErrNotSent, failure("quota_unavailable"))
	}
	if !NeedsActivation(fresh) {
		return ErrAlreadyActive
	}
	models, err := s.access.Models(ctx, expected.ID)
	if err != nil || len(models) == 0 {
		return errors.Join(ErrNotSent, failure("models_unavailable"))
	}
	model := models[0]
	for _, item := range models {
		if item.Metadata.IsDefault {
			model = item
			break
		}
	}
	body := map[string]any{
		"model":        model.ID,
		"instructions": "Reply only OK.",
		"input":        []any{map[string]any{"role": "user", "content": []any{map[string]any{"type": "input_text", "text": "Hi"}}}},
		"tools":        []any{}, "tool_choice": "none", "store": false,
	}
	for _, effort := range []string{"none", "minimal", "low"} {
		found := false
		for _, supported := range model.Metadata.ReasoningEfforts {
			if supported.Value == effort {
				found = true
			}
		}
		if found {
			body["reasoning"] = map[string]string{"effort": effort}
			break
		}
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return errors.Join(ErrNotSent, failure("request_failed"))
	}
	defer clear(raw)
	attempted := false
	err = s.access.UseToken(ctx, expected.ID, func(token []byte, officialID string) error {
		current, err := s.accounts.Metadata(ctx, expected.ID)
		if err != nil || !current.Enabled || current.AuthState != "authenticated" || identityVersion(current) != identityVersion(expected) || officialID != expected.AuthAccountID {
			return failure("identity_changed")
		}
		providers, err := s.providers.List(ctx)
		if err != nil {
			return failure("accounts_unavailable")
		}
		allowed := false
		for _, item := range providers {
			if item.ID == expected.ProviderID && item.Kind == "openai" && item.Enabled {
				allowed = true
			}
		}
		if !allowed || ctx.Err() != nil {
			return failure("account_unavailable")
		}
		attempted = true
		return s.client.Forward(ctx, token, officialID, proxy.Request{Protocol: "responses", Model: model.ID, Body: raw, Stream: true}, consumeActivation)
	})
	if err != nil && !attempted {
		return errors.Join(ErrNotSent, err)
	}
	return err
}

const maxActivationResponseBytes = 64 << 10

func consumeActivation(response proxy.Response) error {
	if response.StatusCode != 200 {
		return failure("request_failed")
	}
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 4096), maxActivationResponseBytes)
	total := 0
	for scanner.Scan() {
		line := scanner.Text()
		total += len(line) + 1
		if total > maxActivationResponseBytes {
			return failure("response_too_large")
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			continue
		}
		var event struct {
			Type     string `json:"type"`
			Response struct {
				Status string          `json:"status"`
				Error  json.RawMessage `json:"error"`
			} `json:"response"`
		}
		if json.Unmarshal([]byte(data), &event) != nil {
			return failure("request_failed")
		}
		switch event.Type {
		case "error", "response.failed", "response.incomplete":
			return failure("request_failed")
		case "response.completed":
			if event.Response.Status != "completed" || (len(event.Response.Error) > 0 && string(event.Response.Error) != "null") {
				return failure("request_failed")
			}
			return nil
		}
	}
	return failure("request_failed")
}
