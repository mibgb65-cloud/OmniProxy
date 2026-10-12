package codex

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
)

// Login 只含官方授权位置和一次性用户码，不含令牌。
type Login struct {
	Type            string `json:"type"`
	ID              string `json:"loginId"`
	AuthURL         string `json:"authUrl"`
	VerificationURL string `json:"verificationUrl"`
	UserCode        string `json:"userCode"`
}

// Account 是官方接口提供的身份与套餐元数据，不表示实际模型权限已经验证。
type Account struct {
	Type  string `json:"type"`
	Email string `json:"email"`
	Plan  string `json:"planType"`
}

// ValidAuthorizationURL 限定官方 HTTPS 授权域名，拒绝用户信息、片段及异常长度。
func ValidAuthorizationURL(value string) bool {
	if len(value) < 1 || len(value) > 8192 || strings.ContainsAny(value, "\r\n\x00") {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil && parsed.Scheme == "https" && parsed.User == nil && parsed.Fragment == "" && parsed.Port() == "" && (parsed.Host == "auth.openai.com" || parsed.Host == "chatgpt.com")
}

func (c *rpcClient) Begin(ctx context.Context, mode string) (Login, error) {
	if mode != "browser" && mode != "device" {
		return Login{}, ErrProtocol
	}
	kind := "chatgpt"
	if mode == "device" {
		kind = "chatgptDeviceCode"
	}
	var result Login
	if err := c.call(ctx, "account/login/start", map[string]string{"type": kind}, &result); err != nil {
		return result, err
	}
	location := result.AuthURL
	if mode == "device" {
		location = result.VerificationURL
	}
	if result.Type != kind || result.ID == "" || len(result.ID) > 128 || !ValidAuthorizationURL(location) || len(result.UserCode) > 128 || strings.ContainsAny(result.ID+result.UserCode, "\r\n\x00") || mode == "device" && result.UserCode == "" {
		return Login{}, ErrProtocol
	}
	if mode == "browser" {
		compatible, err := compatibleBrowserURL(result.AuthURL)
		if err != nil {
			return Login{}, err
		}
		result.AuthURL = compatible
	}
	return result, nil
}

func (c *rpcClient) Wait(ctx context.Context, id string) error {
	for {
		select {
		case event := <-c.notices:
			var result struct {
				ID      string `json:"loginId"`
				Success bool   `json:"success"`
			}
			if json.Unmarshal(event.params, &result) != nil {
				return ErrProtocol
			}
			if result.ID == id {
				if result.Success {
					return nil
				}
				return errors.New("subscription: login_failed")
			}
		case <-ctx.Done():
			return errors.New("subscription: login_expired")
		case <-c.done:
			return ErrProtocol
		}
	}
}

func (c *rpcClient) Account(ctx context.Context, refresh bool) (*Account, error) {
	var result struct {
		Account *Account `json:"account"`
	}
	if err := c.call(ctx, "account/read", map[string]bool{"refreshToken": refresh}, &result); err != nil {
		return nil, err
	}
	if result.Account != nil && (result.Account.Type != "chatgpt" || len(result.Account.Email) > 320 || len(result.Account.Plan) > 80) {
		return nil, ErrProtocol
	}
	return result.Account, nil
}
func (c *rpcClient) Cancel(ctx context.Context, id string) error {
	var result json.RawMessage
	return c.call(ctx, "account/login/cancel", map[string]string{"loginId": id}, &result)
}
func (c *rpcClient) Logout(ctx context.Context) error {
	var result json.RawMessage
	return c.call(ctx, "account/logout", map[string]any{}, &result)
}
