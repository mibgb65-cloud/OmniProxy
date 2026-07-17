package main

import (
	"context"
	"crypto/rand"
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

const (
	codexOAuthCallbackPort = 1455
	codexOAuthTimeout      = 5 * time.Minute
)

type codexOAuthLoginStartResponse struct {
	LoginID string `json:"loginId"`
	AuthURL string `json:"authUrl"`
}

type codexOAuthLoginCompleteRequest struct {
	LoginID string `json:"loginId"`
}

type codexOAuthCallbackResult struct {
	code string
	err  error
}

type codexOAuthSession struct {
	id           string
	authURL      string
	state        string
	verifier     string
	redirectURI  string
	expiresAt    time.Time
	callback     chan codexOAuthCallbackResult
	callbackOnce sync.Once
	server       *http.Server
	listener     net.Listener
}

func (a *appServer) startCodexOAuthLogin() (codexOAuthLoginStartResponse, error) {
	a.codexOAuthMu.Lock()
	defer a.codexOAuthMu.Unlock()

	if existing := a.codexOAuthSession; existing != nil && time.Now().Before(existing.expiresAt) {
		return codexOAuthLoginStartResponse{LoginID: existing.id, AuthURL: existing.authURL}, nil
	}

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", codexOAuthCallbackPort))
	if err != nil {
		return codexOAuthLoginStartResponse{}, fmt.Errorf("无法启动 Codex 登录回调端口 %d：%w", codexOAuthCallbackPort, err)
	}

	verifier, err := codexOAuthRandomToken()
	if err != nil {
		_ = listener.Close()
		return codexOAuthLoginStartResponse{}, err
	}
	state, err := codexOAuthRandomToken()
	if err != nil {
		_ = listener.Close()
		return codexOAuthLoginStartResponse{}, err
	}
	loginID, err := codexOAuthRandomToken()
	if err != nil {
		_ = listener.Close()
		return codexOAuthLoginStartResponse{}, err
	}
	challengeBytes := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(challengeBytes[:])
	redirectURI := fmt.Sprintf("http://localhost:%d/auth/callback", codexOAuthCallbackPort)
	authURL := proxy.CodexOAuthAuthorizationURL(redirectURI, challenge, state)

	session := &codexOAuthSession{
		id:          loginID,
		authURL:     authURL,
		state:       state,
		verifier:    verifier,
		redirectURI: redirectURI,
		expiresAt:   time.Now().Add(codexOAuthTimeout),
		callback:    make(chan codexOAuthCallbackResult, 1),
		listener:    listener,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		a.handleCodexOAuthCallback(session, w, r)
	})
	session.server = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       5 * time.Second,
	}
	a.codexOAuthSession = session

	go func() {
		if err := session.server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) && a.logs != nil {
			a.logs.Add(logs.Entry{Level: logs.LevelWarn, Message: "Codex OAuth callback server stopped unexpectedly"})
		}
	}()
	go a.expireCodexOAuthSession(loginID, session.expiresAt)

	if a.logs != nil {
		a.logs.Add(logs.Entry{Level: logs.LevelInfo, Message: "Codex browser login started"})
	}
	return codexOAuthLoginStartResponse{LoginID: loginID, AuthURL: authURL}, nil
}

func (a *appServer) completeCodexOAuthLogin(ctx context.Context, loginID string) (tokenResponse, error) {
	loginID = strings.TrimSpace(loginID)
	a.codexOAuthMu.Lock()
	session := a.codexOAuthSession
	if session == nil || session.id != loginID {
		a.codexOAuthMu.Unlock()
		return tokenResponse{}, errors.New("Codex 登录会话不存在或已失效")
	}
	expiresAt := session.expiresAt
	a.codexOAuthMu.Unlock()

	var callback codexOAuthCallbackResult
	wait := time.Until(expiresAt)
	if wait <= 0 {
		a.finishCodexOAuthSession(loginID)
		return tokenResponse{}, errors.New("Codex 浏览器登录已超时，请重试")
	}
	select {
	case callback = <-session.callback:
	case <-time.After(wait):
		a.finishCodexOAuthSession(loginID)
		return tokenResponse{}, errors.New("Codex 浏览器登录已超时，请重试")
	case <-ctx.Done():
		a.finishCodexOAuthSession(loginID)
		return tokenResponse{}, ctx.Err()
	}
	if callback.err != nil {
		a.finishCodexOAuthSession(loginID)
		return tokenResponse{}, callback.err
	}

	a.mu.Lock()
	cfg := a.cfg
	a.mu.Unlock()
	validator, err := proxy.NewValidator(cfg)
	if err != nil {
		a.finishCodexOAuthSession(loginID)
		return tokenResponse{}, err
	}
	oauthTokens, err := validator.ExchangeCodexAuthorizationCode(ctx, callback.code, session.verifier, session.redirectURI)
	a.finishCodexOAuthSession(loginID)
	if err != nil {
		return tokenResponse{}, err
	}

	raw, err := codexOAuthAuthJSON(oauthTokens, time.Now())
	if err != nil {
		return tokenResponse{}, err
	}
	result, err := a.upsertCodexOAuthToken(ctx, raw)
	if err != nil {
		return tokenResponse{}, err
	}
	if a.logs != nil {
		a.logs.Add(logs.Entry{Level: logs.LevelInfo, TokenName: result.Name, Message: "Codex browser login completed"})
	}
	return result, nil
}

func (a *appServer) handleCodexOAuthCallback(session *codexOAuthSession, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	query := r.URL.Query()
	if query.Get("state") != session.state {
		session.callbackOnce.Do(func() {
			session.callback <- codexOAuthCallbackResult{err: errors.New("Codex 登录状态校验失败")}
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
			session.callback <- codexOAuthCallbackResult{err: fmt.Errorf("Codex 授权失败：%s", message)}
		})
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(codexOAuthResultPage(false, "Codex 授权未完成，请返回 OmniProxy 重试。")))
		return
	}
	code := strings.TrimSpace(query.Get("code"))
	if code == "" {
		session.callbackOnce.Do(func() {
			session.callback <- codexOAuthCallbackResult{err: errors.New("Codex 登录回调缺少授权码")}
		})
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(codexOAuthResultPage(false, "登录回调缺少授权码，请返回 OmniProxy 重试。")))
		return
	}
	session.callbackOnce.Do(func() {
		session.callback <- codexOAuthCallbackResult{code: code}
	})
	_, _ = w.Write([]byte(codexOAuthResultPage(true, "授权完成，可以关闭此页面并返回 OmniProxy。")))
}

func (a *appServer) expireCodexOAuthSession(loginID string, expiresAt time.Time) {
	timer := time.NewTimer(time.Until(expiresAt))
	defer timer.Stop()
	<-timer.C
	a.codexOAuthMu.Lock()
	session := a.codexOAuthSession
	if session == nil || session.id != loginID {
		a.codexOAuthMu.Unlock()
		return
	}
	a.codexOAuthSession = nil
	a.codexOAuthMu.Unlock()
	session.callbackOnce.Do(func() {
		session.callback <- codexOAuthCallbackResult{err: errors.New("Codex 浏览器登录已超时，请重试")}
	})
	_ = session.server.Close()
}

func (a *appServer) finishCodexOAuthSession(loginID string) {
	a.codexOAuthMu.Lock()
	session := a.codexOAuthSession
	if session != nil && session.id == loginID {
		a.codexOAuthSession = nil
	}
	a.codexOAuthMu.Unlock()
	if session != nil && session.id == loginID {
		_ = session.server.Close()
	}
}

func codexOAuthRandomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("生成 Codex 登录安全参数失败: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func codexOAuthAuthJSON(oauthTokens proxy.CodexOAuthTokens, now time.Time) (string, error) {
	payload := map[string]any{
		"type":         "codex",
		"access_token": oauthTokens.AccessToken,
		"id_token":     oauthTokens.IDToken,
		"last_refresh": now.UTC().Format(time.RFC3339),
	}
	if oauthTokens.RefreshToken != "" {
		payload["refresh_token"] = oauthTokens.RefreshToken
	}
	if oauthTokens.ExpiresIn > 0 {
		payload["expired"] = now.UTC().Add(time.Duration(oauthTokens.ExpiresIn) * time.Second).Format(time.RFC3339)
	}
	raw, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	fields, ok := token.ExtractCodexAuthFields(string(raw))
	if ok {
		if fields.Email != "" {
			payload["email"] = fields.Email
		}
		if fields.AccountID != "" {
			payload["account_id"] = fields.AccountID
		}
		raw, err = json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return "", err
		}
	}
	return string(raw), nil
}

func (a *appServer) upsertCodexOAuthToken(ctx context.Context, raw string) (tokenResponse, error) {
	fields, ok := token.ExtractCodexAuthFields(raw)
	if !ok || !fields.HasSupportedToken() {
		return tokenResponse{}, errors.New("Codex 登录结果缺少可用凭证")
	}
	request := token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     raw,
	}
	for _, item := range a.tokens.List() {
		if item.Provider != token.ProviderOpenAI || item.CredentialType != token.CredentialTypeCodexAuthJSON {
			continue
		}
		existing, existingOK := token.ExtractCodexAuthFields(item.TokenValue)
		if !existingOK || !sameCodexOAuthIdentity(existing, fields) {
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

func sameCodexOAuthIdentity(left token.CodexAuthFields, right token.CodexAuthFields) bool {
	leftAccountID := strings.TrimSpace(left.AccountID)
	rightAccountID := strings.TrimSpace(right.AccountID)
	if leftAccountID != "" && rightAccountID != "" {
		return leftAccountID == rightAccountID
	}
	leftEmail := strings.TrimSpace(left.Email)
	rightEmail := strings.TrimSpace(right.Email)
	return leftEmail != "" && rightEmail != "" && strings.EqualFold(leftEmail, rightEmail)
}

func codexOAuthResultPage(success bool, message string) string {
	title := "授权失败"
	accent := "#dc2626"
	if success {
		title = "授权成功"
		accent = "#16a34a"
	}
	return fmt.Sprintf(`<!doctype html><html lang="zh-CN"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title></head><body style="margin:0;min-height:100vh;display:grid;place-items:center;background:#f5f7fb;font-family:system-ui,sans-serif;color:#172033"><main style="width:min(420px,calc(100%% - 40px));padding:32px;border:1px solid #dfe4ec;border-radius:24px;background:#fff;text-align:center;box-shadow:0 18px 50px rgba(15,23,42,.08)"><div style="width:48px;height:48px;margin:0 auto 18px;border-radius:50%%;background:%s;color:#fff;display:grid;place-items:center;font-size:24px">%s</div><h1 style="margin:0 0 10px;font-size:24px">%s</h1><p style="margin:0;color:#667085;line-height:1.7">%s</p></main></body></html>`, title, accent, map[bool]string{true: "✓", false: "!"}[success], title, message)
}
