package antigravity

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strings"
	"time"
)

// Credentials 只能写入受保护存储，禁止作为绑定或日志返回。
type Credentials struct {
	OAuthConfig
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
	Subject      string    `json:"subject"`
	Email        string    `json:"email"`
	Project      string    `json:"project"`
	Plan         string    `json:"plan"`
}

// Valid 检查解密凭据的类型、格式与边界，不以套餐名推断权限。
func (v Credentials) Valid() bool {
	return v.ClientID == desktopClientID && clientPattern.MatchString(v.ClientSecret) && len(v.ClientSecret) == 35 && safeString(v.AccessToken, 16384) && safeString(v.RefreshToken, 16384) && safeString(v.Subject, 128) && safeString(v.Email, 320) && safeString(v.Project, 256) && (v.Plan == "" || safeString(v.Plan, 80)) && !v.ExpiresAt.IsZero()
}

// AuthorizationURL 使用 PKCE 与随机 state 构造固定 Google 授权地址。
func AuthorizationURL(config OAuthConfig, state, verifier, redirect string) string {
	hash := sha256.Sum256([]byte(verifier))
	query := url.Values{"client_id": {config.ClientID}, "redirect_uri": {redirect}, "response_type": {"code"}, "access_type": {"offline"}, "prompt": {"consent"}, "state": {state}, "code_challenge_method": {"S256"}, "code_challenge": {base64.RawURLEncoding.EncodeToString(hash[:])}, "scope": {strings.Join([]string{"https://www.googleapis.com/auth/cloud-platform", "https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/userinfo.profile", "https://www.googleapis.com/auth/cclog", "https://www.googleapis.com/auth/experimentsandconfigs"}, " ")}}
	return authURL + "?" + query.Encode()
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (c *Client) token(ctx context.Context, config OAuthConfig, form url.Values, cred *Credentials) error {
	form.Set("client_id", config.ClientID)
	form.Set("client_secret", config.ClientSecret)
	data := []byte(form.Encode())
	defer clear(data)
	var value tokenResponse
	if err := c.read(ctx, tokenURL, data, "", "application/x-www-form-urlencoded", &value); err != nil {
		return err
	}
	if !safeString(value.AccessToken, 16384) || value.ExpiresIn < 1 || value.ExpiresIn > 86400 {
		return Failure("invalid_response")
	}
	cred.OAuthConfig, cred.AccessToken, cred.ExpiresAt = config, value.AccessToken, time.Now().Add(time.Duration(value.ExpiresIn)*time.Second)
	if value.RefreshToken != "" {
		if !safeString(value.RefreshToken, 16384) {
			return Failure("invalid_response")
		}
		cred.RefreshToken = value.RefreshToken
	}
	if !safeString(cred.RefreshToken, 16384) {
		return Failure("reauth_required")
	}
	return nil
}

// Exchange 用一次性授权码换取独立凭证并确认身份与订阅项目。
func (c *Client) Exchange(ctx context.Context, config OAuthConfig, code, verifier, redirect string) (Credentials, error) {
	cred := Credentials{}
	if !safeString(code, 8192) {
		return cred, Failure("invalid_input")
	}
	err := c.token(ctx, config, url.Values{"code": {code}, "redirect_uri": {redirect}, "grant_type": {"authorization_code"}, "code_verifier": {verifier}}, &cred)
	if err != nil {
		return Credentials{}, err
	}
	if err = c.Identity(ctx, &cred); err != nil {
		return Credentials{}, err
	}
	if err = c.Project(ctx, &cred); err != nil {
		return Credentials{}, err
	}
	if !cred.Valid() {
		return Credentials{}, Failure("invalid_response")
	}
	return cred, nil
}

// Refresh 仅刷新同一凭证，不执行计费或推理请求。
func (c *Client) Refresh(ctx context.Context, cred *Credentials) error {
	if !cred.Valid() {
		return Failure("reauth_required")
	}
	return c.token(ctx, cred.OAuthConfig, url.Values{"refresh_token": {cred.RefreshToken}, "grant_type": {"refresh_token"}}, cred)
}

// Identity 每次显式刷新校验 Google 身份，不接受账号切换。
func (c *Client) Identity(ctx context.Context, cred *Credentials) error {
	var value struct {
		ID       string `json:"id"`
		Email    string `json:"email"`
		Verified bool   `json:"verified_email"`
	}
	if err := c.read(ctx, userURL, nil, cred.AccessToken, "application/json", &value); err != nil {
		return err
	}
	if !value.Verified || !safeString(value.ID, 128) || !safeString(value.Email, 320) {
		return Failure("invalid_response")
	}
	if cred.Subject != "" && cred.Subject != value.ID {
		return Failure("identity_changed")
	}
	cred.Subject, cred.Email = value.ID, value.Email
	return nil
}
