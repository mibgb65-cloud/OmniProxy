package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"omniproxy/internal/core/netproxy"
)

// 与 v1 相同的官方 OAuth 参数：先用授权码换令牌，再用 refresh_token 续期。
const (
	oauthClientID       = "app_EMoamEEZ73f0CkXaXp7hrann"
	oauthRefreshScope   = "openid profile email"
	oauthUserAgent      = "codex-cli/0.91.0"
	accessRefreshMargin = 30 * time.Minute
	maxAuthJSONBytes    = 128 << 10
	maxOAuthResponse    = 1 << 20
)

// oauthTokenURL 是令牌端点；测试可以覆盖它，生产恒为官方地址。
var oauthTokenURL = "https://auth.openai.com/oauth/token"

type refreshResponse struct {
	AccessToken  string `json:"access_token"`
	IDToken      string `json:"id_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// AuthNeedsRefresh 只解析官方保存的 auth.json，判断访问令牌是否即将过期。
// 解析失败报告需要续期，让调用方看到真实错误而不是静默继续用坏令牌。
func AuthNeedsRefresh(raw string, now time.Time) bool {
	auth, tokens, _, err := parseAuth(raw)
	if err != nil {
		return true
	}
	if expiresSoon(mapValue(tokens, "access_token"), now) {
		return true
	}
	return storedExpirySoon(auth, now) || storedExpirySoon(tokens, now)
}

// RefreshAuthJSON 用 refresh_token 换新访问令牌，其余字段原样保留。
func RefreshAuthJSON(ctx context.Context, client *http.Client, raw string, now time.Time) (string, bool, error) {
	if client == nil {
		// 续期同样属于提供商出站流量：默认客户端不能绕过代理策略。
		client = &http.Client{Transport: netproxy.Wrap(nil)}
	}
	auth, tokens, topLevel, err := parseAuth(raw)
	if err != nil {
		return "", false, err
	}
	refreshToken := mapValue(tokens, "refresh_token")
	if refreshToken == "" {
		return "", false, ErrProtocol
	}
	values := url.Values{}
	values.Set("grant_type", "refresh_token")
	values.Set("client_id", authClientID(auth, tokens))
	values.Set("refresh_token", refreshToken)
	values.Set("scope", oauthRefreshScope)

	resp, err := postForm(ctx, client, oauthTokenURL, values)
	if err != nil {
		return "", false, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxOAuthResponse))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false, ErrProtocol
	}
	var refresh refreshResponse
	if json.Unmarshal(body, &refresh) != nil || !validTokenValue(refresh.AccessToken) {
		return "", false, ErrProtocol
	}
	tokens["access_token"] = refresh.AccessToken
	if validTokenValue(refresh.IDToken) {
		tokens["id_token"] = refresh.IDToken
	}
	if validTokenValue(refresh.RefreshToken) {
		tokens["refresh_token"] = refresh.RefreshToken
	}
	if !topLevel {
		auth["tokens"] = tokens
	}
	if refresh.ExpiresIn > 0 && refresh.ExpiresIn <= 86400*365 {
		auth["last_refresh"] = now.UTC().Format(time.RFC3339Nano)
	}
	updated, err := json.Marshal(auth)
	if err != nil || len(updated) > maxAuthJSONBytes {
		return "", false, ErrProtocol
	}
	defer clear(updated)
	return string(updated), true, nil
}

func postForm(ctx context.Context, client *http.Client, endpoint string, values url.Values) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, ErrProtocol
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", oauthUserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Join(ErrUnavailable, err)
	}
	return resp, nil
}
