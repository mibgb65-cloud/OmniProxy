package main

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"omniproxy/internal/logs"
	"omniproxy/internal/proxy"
	"omniproxy/internal/token"
)

const claudeOAuthTimeout = 5 * time.Minute

type claudeOAuthLoginStartResponse struct {
	LoginID string `json:"loginId"`
	AuthURL string `json:"authUrl"`
}

type claudeOAuthLoginStatusResponse struct {
	Ready bool `json:"ready"`
}

type claudeOAuthLoginCompleteRequest struct {
	LoginID string `json:"loginId"`
}

type claudeOAuthCallbackResult struct {
	code string
	err  error
}

type claudeOAuthSession struct {
	id           string
	authURL      string
	state        string
	verifier     string
	redirectURI  string
	expiresAt    time.Time
	callback     chan claudeOAuthCallbackResult
	callbackOnce sync.Once
	server       *http.Server
	listener     net.Listener
}

func (a *appServer) startClaudeOAuthLogin(refresh bool) (claudeOAuthLoginStartResponse, error) {
	a.claudeOAuthMu.Lock()
	defer a.claudeOAuthMu.Unlock()

	if existing := a.claudeOAuthSession; existing != nil {
		if !refresh && time.Now().Before(existing.expiresAt) {
			return claudeOAuthLoginStartResponse{LoginID: existing.id, AuthURL: existing.authURL}, nil
		}
		a.claudeOAuthSession = nil
		existing.callbackOnce.Do(func() {
			if existing.callback != nil {
				existing.callback <- claudeOAuthCallbackResult{err: errors.New("Claude 登录链接已刷新")}
			}
		})
		if existing.server != nil {
			_ = existing.server.Close()
		}
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return claudeOAuthLoginStartResponse{}, fmt.Errorf("无法启动 Claude 登录回调：%w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port

	verifier, err := oauthRandomToken()
	if err != nil {
		_ = listener.Close()
		return claudeOAuthLoginStartResponse{}, err
	}
	state, err := oauthRandomToken()
	if err != nil {
		_ = listener.Close()
		return claudeOAuthLoginStartResponse{}, err
	}
	loginID, err := oauthRandomToken()
	if err != nil {
		_ = listener.Close()
		return claudeOAuthLoginStartResponse{}, err
	}
	challengeBytes := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeBytes[:])
	redirectURI := fmt.Sprintf("http://localhost:%d/callback", port)
	authURL := proxy.ClaudeOAuthAuthorizationURL(redirectURI, challenge, state)

	session := &claudeOAuthSession{
		id:          loginID,
		authURL:     authURL,
		state:       state,
		verifier:    verifier,
		redirectURI: redirectURI,
		expiresAt:   time.Now().Add(claudeOAuthTimeout),
		callback:    make(chan claudeOAuthCallbackResult, 1),
		listener:    listener,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		a.handleClaudeOAuthCallback(session, w, r)
	})
	session.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       5 * time.Second,
	}
	a.claudeOAuthSession = session

	go func() {
		if err := session.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) && a.logs != nil {
			a.logs.Add(logs.Entry{Level: logs.LevelWarn, Message: "Claude OAuth callback server stopped unexpectedly"})
		}
	}()
	go a.expireClaudeOAuthSession(loginID, session.expiresAt)

	if a.logs != nil {
		a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "Claude browser login started"})
	}
	return claudeOAuthLoginStartResponse{LoginID: loginID, AuthURL: authURL}, nil
}

func (a *appServer) claudeOAuthLoginStatus(loginID string) (claudeOAuthLoginStatusResponse, error) {
	loginID = strings.TrimSpace(loginID)
	a.claudeOAuthMu.Lock()
	defer a.claudeOAuthMu.Unlock()

	session := a.claudeOAuthSession
	if session == nil || session.id != loginID {
		return claudeOAuthLoginStatusResponse{}, errors.New("Claude 登录会话不存在或已失效")
	}
	if !time.Now().Before(session.expiresAt) {
		return claudeOAuthLoginStatusResponse{}, errors.New("Claude 浏览器登录已超时，请刷新登录链接")
	}
	return claudeOAuthLoginStatusResponse{Ready: len(session.callback) > 0}, nil
}

func (a *appServer) completeClaudeOAuthLogin(ctx context.Context, loginID string) (tokenResponse, error) {
	loginID = strings.TrimSpace(loginID)
	a.claudeOAuthMu.Lock()
	session := a.claudeOAuthSession
	if session == nil || session.id != loginID {
		a.claudeOAuthMu.Unlock()
		return tokenResponse{}, errors.New("Claude 登录会话不存在或已失效")
	}
	expiresAt := session.expiresAt
	a.claudeOAuthMu.Unlock()

	var callback claudeOAuthCallbackResult
	wait := time.Until(expiresAt)
	if wait <= 0 {
		a.finishClaudeOAuthSession(loginID)
		return tokenResponse{}, errors.New("Claude 浏览器登录已超时，请重试")
	}
	select {
	case callback = <-session.callback:
	case <-time.After(wait):
		a.finishClaudeOAuthSession(loginID)
		return tokenResponse{}, errors.New("Claude 浏览器登录已超时，请重试")
	case <-ctx.Done():
		a.finishClaudeOAuthSession(loginID)
		return tokenResponse{}, ctx.Err()
	}
	if callback.err != nil {
		a.finishClaudeOAuthSession(loginID)
		return tokenResponse{}, callback.err
	}

	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()
	validator, err := proxy.NewValidator(cfg)
	if err != nil {
		a.finishClaudeOAuthSession(loginID)
		return tokenResponse{}, err
	}
	oauthTokens, err := validator.ExchangeClaudeAuthorizationCode(
		ctx,
		callback.code,
		session.state,
		session.verifier,
		session.redirectURI,
	)
	a.finishClaudeOAuthSession(loginID)
	if err != nil {
		return tokenResponse{}, err
	}

	raw, err := claudeOAuthAuthJSON(oauthTokens, time.Now())
	if err != nil {
		return tokenResponse{}, err
	}
	result, err := a.upsertClaudeOAuthToken(ctx, raw)
	if err != nil {
		return tokenResponse{}, err
	}
	if a.logs != nil {
		a.logs.Add(logs.Entry{Level: logs.LevelInfo, TokenName: result.Name, Message: "Claude browser login completed"})
	}
	return result, nil
}

func (a *appServer) handleClaudeOAuthCallback(session *claudeOAuthSession, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	query := r.URL.Query()
	if query.Get("state") != session.state {
		session.callbackOnce.Do(func() {
			session.callback <- claudeOAuthCallbackResult{err: errors.New("Claude 登录状态校验失败")}
		})
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(codexOAuthResultPage(false, "登录校验失败，请返回 OmniProxy 重试。")))
		return
	}
	if upstreamError := strings.TrimSpace(query.Get("error")); upstreamError != "" {
		message := strings.TrimSpace(query.Get("error_description"))
		if message == "" {
			message = upstreamError
		}
		session.callbackOnce.Do(func() {
			session.callback <- claudeOAuthCallbackResult{err: fmt.Errorf("Claude 授权失败：%s", message)}
		})
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(codexOAuthResultPage(false, "Claude 授权未完成，请返回 OmniProxy 重试。")))
		return
	}
	code := strings.TrimSpace(query.Get("code"))
	if code == "" {
		session.callbackOnce.Do(func() {
			session.callback <- claudeOAuthCallbackResult{err: errors.New("Claude 登录回调缺少授权码")}
		})
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(codexOAuthResultPage(false, "登录回调缺少授权码，请返回 OmniProxy 重试。")))
		return
	}
	session.callbackOnce.Do(func() {
		session.callback <- claudeOAuthCallbackResult{code: code}
	})
	_, _ = w.Write([]byte(codexOAuthResultPage(true, "授权完成，OmniProxy 正在自动识别并导入 Claude 账号。")))
}

func (a *appServer) expireClaudeOAuthSession(loginID string, expiresAt time.Time) {
	timer := time.NewTimer(time.Until(expiresAt))
	defer timer.Stop()
	<-timer.C
	a.claudeOAuthMu.Lock()
	session := a.claudeOAuthSession
	if session == nil || session.id != loginID {
		a.claudeOAuthMu.Unlock()
		return
	}
	a.claudeOAuthSession = nil
	a.claudeOAuthMu.Unlock()
	session.callbackOnce.Do(func() {
		session.callback <- claudeOAuthCallbackResult{err: errors.New("Claude 浏览器登录已超时，请重试")}
	})
	_ = session.server.Close()
}

func (a *appServer) finishClaudeOAuthSession(loginID string) {
	a.claudeOAuthMu.Lock()
	session := a.claudeOAuthSession
	if session != nil && session.id == loginID {
		a.claudeOAuthSession = nil
	}
	a.claudeOAuthMu.Unlock()
	if session != nil && session.id == loginID {
		_ = session.server.Close()
	}
}

func claudeOAuthAuthJSON(oauthTokens proxy.ClaudeOAuthTokens, now time.Time) (string, error) {
	credential := map[string]any{
		"accessToken": oauthTokens.AccessToken,
	}
	if oauthTokens.RefreshToken != "" {
		credential["refreshToken"] = oauthTokens.RefreshToken
	}
	if oauthTokens.ExpiresIn > 0 {
		credential["expiresAt"] = now.UTC().Add(time.Duration(oauthTokens.ExpiresIn) * time.Second).UnixMilli()
	}
	if scopes := strings.Fields(oauthTokens.Scope); len(scopes) > 0 {
		credential["scopes"] = scopes
	}

	payload := map[string]any{
		"claudeAiOauth": credential,
		"last_refresh":  now.UTC().Format(time.RFC3339),
	}
	if oauthTokens.Email != "" {
		payload["email"] = oauthTokens.Email
	}
	if oauthTokens.AccountID != "" {
		payload["account_uuid"] = oauthTokens.AccountID
	}
	if oauthTokens.OrganizationID != "" {
		payload["organization_uuid"] = oauthTokens.OrganizationID
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (a *appServer) upsertClaudeOAuthToken(ctx context.Context, raw string) (tokenResponse, error) {
	fields, ok := token.ExtractClaudeOAuthFields(raw)
	if !ok {
		return tokenResponse{}, errors.New("Claude 登录结果缺少可用凭证")
	}
	name := strings.TrimSpace(fields.Email)
	if name == "" && fields.AccountID != "" {
		name = "Claude " + fields.AccountID
	}
	if name == "" {
		return tokenResponse{}, errors.New("Claude 登录结果缺少账号身份")
	}
	request := token.UpsertRequest{
		Name:           name,
		Provider:       token.ProviderAnthropic,
		CredentialType: token.CredentialTypeClaudeOAuth,
		TokenValue:     raw,
	}
	for _, item := range a.tokens.List() {
		if item.Provider != token.ProviderAnthropic || item.CredentialType != token.CredentialTypeClaudeOAuth {
			continue
		}
		existing, existingOK := token.ExtractClaudeOAuthFields(item.TokenValue)
		if !existingOK || !sameClaudeOAuthIdentity(existing, fields) {
			continue
		}
		updated, err := a.updateToken(item.ID, request)
		if err != nil {
			return tokenResponse{}, err
		}
		_, _ = a.validateToken(ctx, item.ID)
		if latest, err := a.tokens.Get(item.ID); err == nil {
			return tokenResponseFor(latest), nil
		}
		return updated, nil
	}
	return a.createToken(ctx, request)
}

func sameClaudeOAuthIdentity(left token.ClaudeOAuthFields, right token.ClaudeOAuthFields) bool {
	leftAccountID := strings.TrimSpace(left.AccountID)
	rightAccountID := strings.TrimSpace(right.AccountID)
	if leftAccountID != "" && rightAccountID != "" {
		return leftAccountID == rightAccountID
	}
	leftEmail := strings.TrimSpace(left.Email)
	rightEmail := strings.TrimSpace(right.Email)
	return leftEmail != "" && rightEmail != "" && strings.EqualFold(leftEmail, rightEmail)
}
