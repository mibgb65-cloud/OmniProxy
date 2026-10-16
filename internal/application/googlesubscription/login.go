package googlesubscription

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/subscription"
	"omniproxy/internal/provider/antigravity"
)

func randomValue() (string, error) {
	var value [32]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", antigravity.Failure("operation_failed")
	}
	return base64.RawURLEncoding.EncodeToString(value[:]), nil
}

// Begin 创建一次性 loopback OAuth，会话生命周期独立于前端弹窗。
func (s *Service) Begin(ctx context.Context, input subscription.Input) (subscription.State, error) {
	s.ops.Lock()
	defer s.ops.Unlock()
	if input.Mode != "browser" || len(input.Name) > 256 || !utf8.ValidString(input.Name) || utf8.RuneCountInString(input.Name) > 80 || strings.IndexFunc(input.Name, unicode.IsControl) >= 0 || s.accounts.CheckGoogleProvider(ctx, input.ProviderID) != nil {
		return subscription.State{}, antigravity.Failure("invalid_input")
	}
	if input.AccountID != "" {
		item, err := s.accounts.Metadata(ctx, input.AccountID)
		if err != nil || item.ProviderID != input.ProviderID || item.AuthMethod != account.AuthSubscription {
			return subscription.State{}, antigravity.Failure("invalid_input")
		}
	}
	s.mu.Lock()
	busy := s.closed || s.current != nil && s.current.view.State == "waiting"
	s.mu.Unlock()
	if busy {
		return subscription.State{}, antigravity.Failure("busy")
	}
	config, err := s.config()
	if err != nil {
		return subscription.State{}, err
	}
	state, err := randomValue()
	if err != nil {
		return subscription.State{}, err
	}
	verifier, err := randomValue()
	if err != nil {
		return subscription.State{}, err
	}
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return subscription.State{}, antigravity.Failure("operation_failed")
	}
	redirect := "http://" + listener.Addr().String() + "/oauth-callback"
	waitCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	job := &session{input: input, cancel: cancel, done: make(chan struct{}), view: subscription.State{ID: "google_" + state, State: "waiting", URL: antigravity.AuthorizationURL(config, state, verifier, redirect), ExpiresAt: time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339)}}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		cancel()
		listener.Close()
		return subscription.State{}, antigravity.Failure("cancelled")
	}
	s.current = job
	view := job.view
	s.mu.Unlock()
	codes := make(chan string, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth-callback", callbackHandler(state, listener.Addr().String(), codes))
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 5 * time.Second, MaxHeaderBytes: 16 << 10}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			cancel()
		}
	}()
	go s.complete(waitCtx, job, server, config, verifier, redirect, codes)
	return view, nil
}

func callbackHandler(expected, host string, codes chan<- string) http.HandlerFunc {
	var received atomic.Bool
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		if r.Method != http.MethodGet || r.Host != host || len(r.URL.RawQuery) > 16384 {
			http.Error(w, "Invalid callback", 400)
			return
		}
		values, parseErr := url.ParseQuery(r.URL.RawQuery)
		if parseErr != nil || len(values["state"]) != 1 || subtle.ConstantTimeCompare([]byte(values.Get("state")), []byte(expected)) != 1 || len(values["code"]) > 1 || len(values["error"]) > 1 {
			http.Error(w, "Invalid state", 400)
			return
		}
		code := values.Get("code")
		if values.Get("error") != "" {
			code = ""
		}
		if len(code) > 8192 {
			http.Error(w, "Invalid code", 400)
			return
		}
		if !received.CompareAndSwap(false, true) {
			http.Error(w, "Callback already received", http.StatusConflict)
			return
		}
		select {
		case codes <- code:
		default:
			http.Error(w, "Callback already received", http.StatusConflict)
			return
		}
		_, _ = w.Write([]byte("授权结果请返回 OmniProxy 查看，可以关闭此页面。"))
	}
}

func (s *Service) complete(ctx context.Context, job *session, server *http.Server, config antigravity.OAuthConfig, verifier, redirect string, codes <-chan string) {
	defer close(job.done)
	defer server.Close()
	defer job.cancel()
	var item account.Item
	var err error
	select {
	case <-ctx.Done():
		err = antigravity.Failure("cancelled")
	case code := <-codes:
		if code == "" {
			err = antigravity.Failure("access_denied")
			break
		}
		operation, cancel := context.WithTimeout(ctx, 30*time.Second)
		cred, e := s.client.Exchange(operation, config, code, verifier, redirect)
		err = e
		if err == nil {
			s.ops.Lock()
			if operation.Err() != nil {
				err = antigravity.Failure("cancelled")
			} else {
				item, err = s.accounts.SaveGoogleSubscription(operation, job.input.ProviderID, job.input.AccountID, job.input.Name, cred)
			}
			s.ops.Unlock()
		}
		cancel()
	}
	status, code := "completed", ""
	if err != nil {
		status, code = "failed", antigravity.ErrorCode(err)
		if ctx.Err() == context.DeadlineExceeded {
			code = "login_expired"
		}
	}
	s.mu.Lock()
	job.view.State, job.view.ErrorCode, job.view.URL = status, code, ""
	if item.ID != "" {
		job.view.Account = &item
	}
	s.mu.Unlock()
	s.notify()
}
