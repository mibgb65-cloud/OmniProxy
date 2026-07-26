package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"omniproxy/internal/token"
	"strings"
	"time"
)

const (
	claudeOAuthClientID          = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	claudeOAuthAuthorizeEndpoint = "https://claude.com/cai/oauth/authorize"
	claudeOAuthTokenEndpoint     = "https://platform.claude.com/v1/oauth/token"
	claudeOAuthLoginScope        = "org:create_api_key user:profile user:inference user:sessions:claude_code user:mcp_servers user:file_upload"
	claudeOAuthRefreshScope      = "user:profile user:inference user:sessions:claude_code user:mcp_servers user:file_upload"
	claudeAccessRefreshMargin    = 30 * time.Minute
)

type ClaudeOAuthTokens struct {
	AccessToken    string
	RefreshToken   string
	ExpiresIn      int
	Scope          string
	Email          string
	AccountID      string
	OrganizationID string
}

type claudeOAuthRefreshResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	TokenType    string      `json:"token_type"`
	ExpiresIn    int         `json:"expires_in"`
	ExpiresAt    json.Number `json:"expires_at"`
	Scope        string      `json:"scope"`
	Account      struct {
		UUID         string `json:"uuid"`
		EmailAddress string `json:"email_address"`
	} `json:"account"`
	Organization struct {
		UUID string `json:"uuid"`
	} `json:"organization"`
}

type claudeOAuthCredential struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	HasExpiresAt bool
}

var httpPostJSON = func(ctx context.Context, client *http.Client, endpoint string, payload any) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	return client.Do(req)
}

func ClaudeOAuthAuthorizationURL(redirectURI string, codeChallenge string, state string) string {
	values := url.Values{}
	values.Set("code", "true")
	values.Set("client_id", claudeOAuthClientID)
	values.Set("response_type", "code")
	values.Set("redirect_uri", strings.TrimSpace(redirectURI))
	values.Set("scope", claudeOAuthLoginScope)
	values.Set("code_challenge", strings.TrimSpace(codeChallenge))
	values.Set("code_challenge_method", "S256")
	values.Set("state", strings.TrimSpace(state))
	return claudeOAuthAuthorizeEndpoint + "?" + values.Encode()
}

func (v *Validator) ExchangeClaudeAuthorizationCode(ctx context.Context, code string, state string, codeVerifier string, redirectURI string) (ClaudeOAuthTokens, error) {
	payload := map[string]string{
		"grant_type":    "authorization_code",
		"code":          strings.TrimSpace(code),
		"redirect_uri":  strings.TrimSpace(redirectURI),
		"client_id":     claudeOAuthClientID,
		"code_verifier": strings.TrimSpace(codeVerifier),
		"state":         strings.TrimSpace(state),
	}

	client := v.clientForToken(token.Token{Provider: token.ProviderAnthropic})
	resp, err := httpPostJSON(ctx, client, claudeOAuthTokenEndpoint, payload)
	if err != nil {
		return ClaudeOAuthTokens{}, fmt.Errorf("exchange claude authorization code: %w", err)
	}
	defer closeBody(resp.Body)
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return ClaudeOAuthTokens{}, fmt.Errorf("Claude 登录令牌交换返回 %d：%s", resp.StatusCode, codexRefreshErrorMessage(body, resp.Status))
	}

	var result claudeOAuthRefreshResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return ClaudeOAuthTokens{}, fmt.Errorf("decode claude authorization response: %w", err)
	}
	if strings.TrimSpace(result.AccessToken) == "" {
		return ClaudeOAuthTokens{}, errors.New("Claude 登录响应缺少 access_token")
	}
	return ClaudeOAuthTokens{
		AccessToken:    strings.TrimSpace(result.AccessToken),
		RefreshToken:   strings.TrimSpace(result.RefreshToken),
		ExpiresIn:      result.ExpiresIn,
		Scope:          strings.TrimSpace(result.Scope),
		Email:          strings.TrimSpace(result.Account.EmailAddress),
		AccountID:      strings.TrimSpace(result.Account.UUID),
		OrganizationID: strings.TrimSpace(result.Organization.UUID),
	}, nil
}

// ClaudeOAuthNeedsRefresh reports whether the stored OAuth JSON is expired or
// close enough to expiry to be worth refreshing, without making a request.
func ClaudeOAuthNeedsRefresh(raw string, now time.Time) bool {
	if now.IsZero() {
		now = time.Now()
	}
	_, credential, err := parseClaudeOAuth(raw)
	if err != nil {
		return true
	}
	return claudeOAuthExpiredOrExpiring(credential, now)
}

func RefreshClaudeOAuthJSON(ctx context.Context, client *http.Client, raw string, force bool, now time.Time) (string, bool, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if now.IsZero() {
		now = time.Now()
	}

	auth, credential, err := parseClaudeOAuth(raw)
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(credential.RefreshToken) == "" {
		if force || claudeOAuthExpiredOrExpiring(credential, now) {
			return "", false, errors.New("claude OAuth JSON does not contain refresh_token")
		}
		return raw, false, nil
	}
	if !force && !claudeOAuthExpiredOrExpiring(credential, now) {
		return raw, false, nil
	}

	payload := map[string]string{
		"grant_type":    "refresh_token",
		"client_id":     claudeOAuthClientID,
		"refresh_token": credential.RefreshToken,
		"scope":         claudeOAuthRefreshScope,
	}
	resp, err := httpPostJSON(ctx, client, claudeOAuthTokenEndpoint, payload)
	if err != nil {
		return "", false, fmt.Errorf("refresh claude OAuth token: %w", err)
	}
	defer closeBody(resp.Body)

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*1024))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", false, fmt.Errorf("refresh claude OAuth token returned %d: %s", resp.StatusCode, codexRefreshErrorMessage(body, resp.Status))
	}

	var refresh claudeOAuthRefreshResponse
	if err := json.Unmarshal(body, &refresh); err != nil {
		return "", false, fmt.Errorf("decode claude OAuth refresh response: %w", err)
	}
	if strings.TrimSpace(refresh.AccessToken) == "" {
		return "", false, errors.New("claude OAuth refresh response did not include access_token")
	}

	expiresAt := claudeRefreshExpiresAt(refresh, now)
	applyClaudeOAuthRefresh(auth, refresh, expiresAt, now)

	updated, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return "", false, err
	}
	return string(updated), true, nil
}

func parseClaudeOAuth(raw string) (map[string]any, claudeOAuthCredential, error) {
	var auth map[string]any
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&auth); err != nil {
		return nil, claudeOAuthCredential{}, fmt.Errorf("claude OAuth JSON must be valid JSON: %w", err)
	}

	credential := claudeOAuthCredential{
		AccessToken:  firstStringAny(auth, "access_token", "accessToken"),
		RefreshToken: firstStringAny(auth, "refresh_token", "refreshToken"),
	}
	if expiresAt, ok := firstTimeAny(auth, "expired", "expires_at", "expiresAt"); ok {
		credential.ExpiresAt = expiresAt
		credential.HasExpiresAt = true
	}

	for _, key := range []string{"claudeAiOauth", "claude"} {
		nested, ok := anyMap(auth[key])
		if !ok {
			continue
		}
		if credential.AccessToken == "" {
			credential.AccessToken = firstStringAny(nested, "access_token", "accessToken")
		}
		if credential.RefreshToken == "" {
			credential.RefreshToken = firstStringAny(nested, "refresh_token", "refreshToken")
		}
		if !credential.HasExpiresAt {
			if expiresAt, ok := firstTimeAny(nested, "expired", "expires_at", "expiresAt"); ok {
				credential.ExpiresAt = expiresAt
				credential.HasExpiresAt = true
			}
		}
	}

	if credential.AccessToken == "" && credential.RefreshToken == "" {
		return nil, claudeOAuthCredential{}, errors.New("claude OAuth JSON must contain access_token or refresh_token")
	}
	return auth, credential, nil
}

func claudeOAuthExpiredOrExpiring(credential claudeOAuthCredential, now time.Time) bool {
	if strings.TrimSpace(credential.AccessToken) == "" {
		return true
	}
	if !credential.HasExpiresAt {
		return false
	}
	return !credential.ExpiresAt.After(now.Add(claudeAccessRefreshMargin))
}

func claudeRefreshExpiresAt(refresh claudeOAuthRefreshResponse, now time.Time) time.Time {
	if refresh.ExpiresAt != "" {
		if parsed, ok := timeFromAny(refresh.ExpiresAt); ok {
			return parsed
		}
	}
	if refresh.ExpiresIn > 0 {
		return now.Add(time.Duration(refresh.ExpiresIn) * time.Second)
	}
	return now.Add(time.Hour)
}

func applyClaudeOAuthRefresh(auth map[string]any, refresh claudeOAuthRefreshResponse, expiresAt time.Time, now time.Time) {
	accessToken := strings.TrimSpace(refresh.AccessToken)
	refreshToken := strings.TrimSpace(refresh.RefreshToken)
	if refreshToken == "" {
		refreshToken = firstStringAny(auth, "refresh_token", "refreshToken")
	}

	if _, ok := auth["access_token"]; ok || firstStringAny(auth, "accessToken") == "" {
		auth["access_token"] = accessToken
	}
	if _, ok := auth["accessToken"]; ok {
		auth["accessToken"] = accessToken
	}
	if refreshToken != "" {
		if _, ok := auth["refresh_token"]; ok || firstStringAny(auth, "refreshToken") == "" {
			auth["refresh_token"] = refreshToken
		}
		if _, ok := auth["refreshToken"]; ok {
			auth["refreshToken"] = refreshToken
		}
	}
	if _, ok := auth["expired"]; ok {
		auth["expired"] = expiresAt.Format(time.RFC3339)
	} else if _, ok := auth["expires_at"]; ok {
		auth["expires_at"] = expiresAt.Format(time.RFC3339)
	} else if _, ok := auth["expiresAt"]; ok {
		auth["expiresAt"] = expiresAt.UnixMilli()
	} else {
		auth["expired"] = expiresAt.Format(time.RFC3339)
	}
	auth["last_refresh"] = now.Format(time.RFC3339)

	for _, key := range []string{"claudeAiOauth", "claude"} {
		nested, ok := anyMap(auth[key])
		if !ok {
			continue
		}
		if _, ok := nested["accessToken"]; ok {
			nested["accessToken"] = accessToken
		}
		if _, ok := nested["access_token"]; ok {
			nested["access_token"] = accessToken
		}
		if refreshToken != "" {
			if _, ok := nested["refreshToken"]; ok {
				nested["refreshToken"] = refreshToken
			}
			if _, ok := nested["refresh_token"]; ok {
				nested["refresh_token"] = refreshToken
			}
		}
		if _, ok := nested["expiresAt"]; ok {
			nested["expiresAt"] = expiresAt.UnixMilli()
		}
		if _, ok := nested["expires_at"]; ok {
			nested["expires_at"] = expiresAt.Format(time.RFC3339)
		}
		if _, ok := nested["expired"]; ok {
			nested["expired"] = expiresAt.Format(time.RFC3339)
		}
	}
}

func firstStringAny(payload map[string]any, names ...string) string {
	for _, name := range names {
		if text, ok := payload[name].(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func firstTimeAny(payload map[string]any, names ...string) (time.Time, bool) {
	for _, name := range names {
		if value, ok := payload[name]; ok {
			if parsed, ok := timeFromAny(value); ok {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
}

func timeFromAny(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case json.Number:
		if parsed, err := typed.Int64(); err == nil {
			return unixTimeFromNumber(parsed)
		}
	case float64:
		return unixTimeFromNumber(int64(typed))
	case int64:
		return unixTimeFromNumber(typed)
	case int:
		return unixTimeFromNumber(int64(typed))
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return time.Time{}, false
		}
		if parsed, err := time.Parse(time.RFC3339Nano, text); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func unixTimeFromNumber(value int64) (time.Time, bool) {
	if value <= 0 {
		return time.Time{}, false
	}
	if value > 1_000_000_000_000 {
		value /= 1000
	}
	return time.Unix(value, 0), true
}

func anyMap(value any) (map[string]any, bool) {
	nested, ok := value.(map[string]any)
	return nested, ok
}
