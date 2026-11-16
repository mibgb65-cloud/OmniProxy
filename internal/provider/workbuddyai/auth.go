package workbuddyai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"omniproxy/internal/core/proxy"
)

// 登录轮询与超时；测试里可以缩短。
var (
	// pollInterval 是登录轮询间隔。
	pollInterval = time.Second
	// signInWindow 是登录轮询的总窗口；调用方的 ctx 是主要约束，这里只做兜底。
	signInWindow = 5 * time.Minute
)

// SignInState 是一次浏览器登录的 state 与授权页地址。
type SignInState struct {
	State   string
	AuthURL string
}

// Token 是登录或续期返回的令牌；时间字段未知时保持零值。
type Token struct {
	AccessToken      string
	RefreshToken     string
	ExpiresAt        time.Time
	RefreshExpiresAt time.Time
	Domain           string
	TokenType        string
}

// Identity 是登录后读取到的账号身份；上游缺失的字段保持空，不报错。
type Identity struct {
	UID          string
	EnterpriseID string
	Name         string
	Plan         string
}

// BeginSignIn 发起一次浏览器登录，返回 state 与已补上客户端参数的授权页地址。
func (c *Client) BeginSignIn(ctx context.Context) (SignInState, error) {
	address := c.base + signInStatePath + "?platform=" + url.QueryEscape(pluginPlatform)
	data, err := c.call(ctx, http.MethodPost, address, nil, func(request *http.Request) {
		// 登录接口明确要求不带任何身份头；四个头缺一个上游就按需要登录处理。
		for _, name := range []string{"X-No-Authorization", "X-No-User-Id", "X-No-Enterprise-Id", "X-No-Department-Info"} {
			request.Header.Set(name, "true")
		}
	})
	if err != nil {
		return SignInState{}, err
	}
	var value struct {
		State   string `json:"state"`
		AuthURL string `json:"authUrl"`
	}
	if json.Unmarshal(data, &value) != nil || !safeASCII(value.State, maxStateBytes, false) || value.AuthURL == "" {
		return SignInState{}, Failure(CodeInvalidResponse)
	}
	authURL, err := signInURL(value.AuthURL)
	if err != nil {
		return SignInState{}, err
	}
	return SignInState{State: value.State, AuthURL: authURL}, nil
}

// signInURL 校验授权页是 HTTPS，并按桌面端形态补 version 与随机 loginSessionId。
func signInURL(raw string) (string, error) {
	if len(raw) > maxURLBytes {
		return "", Failure(CodeInvalidResponse)
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil {
		return "", Failure(CodeInvalidResponse)
	}
	session, err := randomID()
	if err != nil {
		return "", proxy.UpstreamFailure(err)
	}
	query := parsed.Query()
	query.Set("version", AppVersion)
	query.Set("loginSessionId", session)
	parsed.RawQuery = query.Encode()
	parsed.Fragment = ""
	return parsed.String(), nil
}

// WaitForToken 轮询登录是否完成：每秒一次，最多 5 分钟（ctx 是主要约束）。
// 业务码 11217 表示用户还没在浏览器里完成登录，继续轮询。
func (c *Client) WaitForToken(ctx context.Context, state string) (Token, error) {
	if !safeASCII(state, maxStateBytes, false) {
		return Token{}, proxy.Failure("invalid_request", 400)
	}
	deadline := time.NewTimer(signInWindow)
	defer deadline.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return Token{}, err
		}
		token, done, err := c.pollToken(ctx, state)
		if err != nil {
			return Token{}, err
		}
		if done {
			c.rememberState(state)
			return token, nil
		}
		select {
		case <-ctx.Done():
			return Token{}, ctx.Err()
		case <-deadline.C:
			return Token{}, signInFailure(CodeSignInExpired)
		case <-time.After(pollInterval):
		}
	}
}

// pollToken 发一次轮询；done 为 false 表示上游还没完成登录。
func (c *Client) pollToken(ctx context.Context, state string) (Token, bool, error) {
	address := c.base + signInTokenPath + "?state=" + url.QueryEscape(state)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return Token{}, false, proxy.UpstreamFailure(err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", loginUserAgent)
	response, err := c.http.Do(request)
	if err != nil {
		// 轮询被取消时返回 ctx 错误，让登录编排按「取消」而不是「上游故障」处理。
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Token{}, false, ctxErr
		}
		return Token{}, false, proxy.UpstreamFailure(err)
	}
	defer func() { _ = response.Body.Close() }()
	data, err := readBounded(response.Body, maxSignInBytes)
	if err != nil {
		return Token{}, false, err
	}
	// 408/429 是暂时状态：插件也把它们当成「还没完成」继续等。
	if response.StatusCode == http.StatusRequestTimeout || response.StatusCode == http.StatusTooManyRequests {
		return Token{}, false, nil
	}
	if failure := businessStatus(response.StatusCode, data); failure != nil {
		return Token{}, false, failure
	}
	var envelope struct {
		Code json.RawMessage `json:"code"`
		Data json.RawMessage `json:"data"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return Token{}, false, Failure(CodeInvalidResponse)
	}
	if code, found := rawCode(envelope.Code); found && code != 0 {
		if code == 11217 {
			return Token{}, false, nil
		}
		if mapped, known := businessFailure(code); known {
			return Token{}, false, mapped
		}
		return Token{}, false, Failure(CodeUpstreamUnavailable)
	}
	payload := json.RawMessage(data)
	if len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		payload = envelope.Data
	}
	fields := map[string]json.RawMessage{}
	if json.Unmarshal(payload, &fields) != nil {
		return Token{}, false, Failure(CodeInvalidResponse)
	}
	if len(fields) == 0 {
		return Token{}, false, nil
	}
	token, err := parseToken(payload)
	if err != nil {
		return Token{}, false, err
	}
	return token, true, nil
}

// ReadIdentity 读取登录后的账号身份；字段缺失保持空，不视为失败。
//
// /v2/plugin/login/account 按 state 选中用户刚在浏览器里选的账号（个人与企业账号
// 共用 uid），而本方法的契约签名只带 Token，所以使用 WaitForToken 刚记下的 state：
// 只在登录窗口内附加，过期后按 Bearer 的当前账号读取。
func (c *Client) ReadIdentity(ctx context.Context, token Token) (Identity, error) {
	if !safeASCII(token.AccessToken, maxTokenBytes, false) {
		return Identity{}, Failure(CodeCredentialUnavailable)
	}
	address := c.base + accountPath
	if state := c.recentState(); state != "" {
		address += "?state=" + url.QueryEscape(state)
	}
	data, err := c.call(ctx, http.MethodGet, address, nil, func(request *http.Request) {
		request.Header.Set("Authorization", "Bearer "+token.AccessToken)
		request.Header.Set("X-Domain", c.domainOf(Credentials{Domain: token.Domain}))
		request.Header.Set("X-No-User-Id", "true")
		request.Header.Set("X-No-Enterprise-Id", "true")
	})
	if err != nil {
		return Identity{}, err
	}
	fields := map[string]json.RawMessage{}
	if json.Unmarshal(data, &fields) != nil {
		return Identity{}, Failure(CodeInvalidResponse)
	}
	identity := Identity{
		UID:          stringField(fields, "uid"),
		EnterpriseID: stringField(fields, "enterpriseId"),
		Name:         stringField(fields, "nickname", "name"),
		Plan:         stringField(fields, "plan", "planName", "planCode"),
	}
	if !safeASCII(identity.UID, maxUIDBytes, true) || !safeASCII(identity.EnterpriseID, maxEnterpriseBytes, true) ||
		!safeText(identity.Name, maxAccountNameBytes, true) || !safeText(identity.Plan, maxPlanBytes, true) {
		return Identity{}, Failure(CodeInvalidResponse)
	}
	return identity, nil
}

// Refresh 用 refresh token 换新令牌；单飞与缓存由调用方负责，本客户端无状态
// （登录窗口内的 state 除外）。
//
// 续期属于业务请求：除了 X-Refresh-Token／X-Auth-Refresh-Source，同样带完整签名头
// （凭据本身有效时）；凭据已损坏时只带域名，避免发出非法 Authorization。
func (c *Client) Refresh(ctx context.Context, current Credentials) (Token, error) {
	if !safeASCII(current.RefreshToken, maxTokenBytes, false) {
		return Token{}, signInFailure(CodeSignInExpired)
	}
	sign := current.Valid()
	address := c.base + refreshPath
	data, err := c.call(ctx, http.MethodPost, address, nil, func(request *http.Request) {
		if sign {
			c.applySign(request, current)
		} else {
			request.Header.Set("X-Domain", c.domainOf(current))
		}
		request.Header.Set("X-Refresh-Token", current.RefreshToken)
		request.Header.Set("X-Auth-Refresh-Source", "plugin")
	})
	if err != nil {
		return Token{}, refreshFailure(err)
	}
	return parseToken(data)
}

// refreshFailure 把续期失败换算成登录态语义：鉴权与明确拒绝都要求重新登录，
// 网络与限流保持原错误，让调用方按普通失败处理。
func refreshFailure(err error) error {
	var boundary *proxy.Error
	if errors.As(err, &boundary) {
		switch boundary.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return signInFailure(CodeSignInExpired)
		case http.StatusBadRequest, http.StatusNotFound, http.StatusGone:
			return signInFailure(CodeRefreshRejected)
		}
		return err
	}
	var failure *Error
	if errors.As(err, &failure) {
		// HTTP 200 带非 0 业务码：上游明确拒绝了这次续期。
		return signInFailure(CodeRefreshRejected)
	}
	return err
}

// call 发一次登录类请求并返回外壳里的 data；Authorization 在返回前删除。
func (c *Client) call(ctx context.Context, method, address string, body []byte, prepare func(*http.Request)) (json.RawMessage, error) {
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	var request *http.Request
	var err error
	if reader == nil {
		request, err = http.NewRequestWithContext(ctx, method, address, nil)
	} else {
		request, err = http.NewRequestWithContext(ctx, method, address, reader)
	}
	if err != nil {
		return nil, proxy.UpstreamFailure(err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", loginUserAgent)
	if prepare != nil {
		prepare(request)
	}
	defer request.Header.Del("Authorization")
	response, err := c.http.Do(request)
	if err != nil {
		return nil, proxy.UpstreamFailure(err)
	}
	defer func() { _ = response.Body.Close() }()
	data, err := readBounded(response.Body, maxSignInBytes)
	if err != nil {
		return nil, err
	}
	if failure := businessStatus(response.StatusCode, data); failure != nil {
		return nil, failure
	}
	return decodeBusiness(data)
}

// rememberState 记录登录轮询使用的 state，供随后的 ReadIdentity 使用。
func (c *Client) rememberState(state string) {
	c.stateMu.Lock()
	c.state, c.stateAt = state, time.Now()
	c.stateMu.Unlock()
}

// recentState 只在登录窗口内返回 state，窗口外不再把它带给身份接口。
func (c *Client) recentState() string {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	if c.state == "" || time.Since(c.stateAt) > signInWindow {
		return ""
	}
	return c.state
}

// stringField 按名称顺序取第一个非空字符串字段。
func stringField(fields map[string]json.RawMessage, names ...string) string {
	for _, name := range names {
		raw, exists := fields[name]
		if !exists {
			continue
		}
		var value string
		if json.Unmarshal(raw, &value) == nil && value != "" {
			return value
		}
	}
	return ""
}

// tokenEnvelope 是登录与续期共用的令牌字段；时间既可能是毫秒时间戳，也可能是
// expiresIn 秒数。
type tokenEnvelope struct {
	AccessToken      string          `json:"accessToken"`
	RefreshToken     string          `json:"refreshToken"`
	ExpiresAt        json.RawMessage `json:"expiresAt"`
	RefreshExpiresAt json.RawMessage `json:"refreshExpiresAt"`
	ExpiresIn        json.RawMessage `json:"expiresIn"`
	RefreshExpiresIn json.RawMessage `json:"refreshExpiresIn"`
	Domain           string          `json:"domain"`
	TokenType        string          `json:"tokenType"`
}

// parseToken 解析并校验令牌字段；access token 缺失按无法识别处理。
func parseToken(data json.RawMessage) (Token, error) {
	var raw tokenEnvelope
	if len(data) == 0 || json.Unmarshal(data, &raw) != nil {
		return Token{}, Failure(CodeInvalidResponse)
	}
	if !safeASCII(raw.AccessToken, maxTokenBytes, false) ||
		!safeASCII(raw.RefreshToken, maxTokenBytes, true) ||
		!safeASCII(raw.Domain, maxDomainBytes, true) ||
		!safeASCII(raw.TokenType, maxTokenTypeBytes, true) {
		return Token{}, Failure(CodeInvalidResponse)
	}
	now := time.Now()
	return Token{
		AccessToken:      raw.AccessToken,
		RefreshToken:     raw.RefreshToken,
		ExpiresAt:        expiryTime(raw.ExpiresAt, raw.ExpiresIn, now),
		RefreshExpiresAt: expiryTime(raw.RefreshExpiresAt, raw.RefreshExpiresIn, now),
		Domain:           raw.Domain,
		TokenType:        raw.TokenType,
	}, nil
}

// expiryTime 兼容毫秒/秒时间戳、数字字符串、RFC3339 与 expiresIn 秒。
func expiryTime(at, in json.RawMessage, now time.Time) time.Time {
	if expiry, found := parseEpoch(at); found {
		return expiry
	}
	if seconds, found := parseSeconds(in); found && seconds > 0 && seconds <= 10*365*24*3600 {
		return now.Add(time.Duration(seconds) * time.Second).UTC()
	}
	return time.Time{}
}

// parseEpoch 解析绝对时间：上游用毫秒，也兼容秒与 RFC3339。
func parseEpoch(raw json.RawMessage) (time.Time, bool) {
	if len(raw) == 0 {
		return time.Time{}, false
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return time.Time{}, false
	}
	switch typed := value.(type) {
	case float64:
		return epochValue(typed)
	case string:
		if number, err := strconv.ParseFloat(strings.TrimSpace(typed), 64); err == nil {
			return epochValue(number)
		}
		if parsed, err := time.Parse(time.RFC3339Nano, typed); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}

func epochValue(value float64) (time.Time, bool) {
	if math.IsNaN(value) || math.IsInf(value, 0) || value <= 0 {
		return time.Time{}, false
	}
	switch {
	case value >= 1e12:
		return time.UnixMilli(int64(value)).UTC(), true
	case value >= 1e9:
		return time.Unix(int64(value), 0).UTC(), true
	}
	return time.Time{}, false
}

// parseSeconds 解析 expiresIn 秒数（数字或数字字符串）。
func parseSeconds(raw json.RawMessage) (int64, bool) {
	if len(raw) == 0 {
		return 0, false
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		if math.IsNaN(typed) || math.IsInf(typed, 0) || typed != math.Trunc(typed) {
			return 0, false
		}
		return int64(typed), true
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	}
	return 0, false
}
