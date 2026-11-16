package workbuddy

import (
	"context"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/subscription"
	"omniproxy/internal/provider/workbuddyai"
)

// Begin 发起一次浏览器登录；会话生命周期独立于前端弹窗，关闭弹窗不会中断。
func (s *Service) Begin(ctx context.Context, input subscription.Input) (subscription.State, error) {
	s.ops.Lock()
	defer s.ops.Unlock()
	if input.Mode != "browser" || len(input.Name) > 256 || !utf8.ValidString(input.Name) || utf8.RuneCountInString(input.Name) > 80 || strings.IndexFunc(input.Name, unicode.IsControl) >= 0 || s.accounts.CheckWorkBuddyProvider(ctx, input.ProviderID) != nil {
		return subscription.State{}, workbuddyai.Failure("invalid_input")
	}
	if input.AccountID != "" {
		item, err := s.accounts.Metadata(ctx, input.AccountID)
		if err != nil || item.ProviderID != input.ProviderID || item.AuthMethod != account.AuthWorkBuddyLogin {
			return subscription.State{}, workbuddyai.Failure("invalid_input")
		}
	}
	s.mu.Lock()
	busy := s.closed || s.current != nil && subscription.InProgress(s.current.view.State)
	s.mu.Unlock()
	if busy {
		return subscription.State{}, workbuddyai.Failure("busy")
	}
	startCtx, cancelStart := context.WithTimeout(ctx, operationTimeout)
	signIn, err := s.client.BeginSignIn(startCtx)
	cancelStart()
	if err != nil {
		return subscription.State{}, err
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), loginTimeout)
	job := &session{
		input:  input,
		cancel: cancel,
		done:   make(chan struct{}),
		view: subscription.State{
			ID:        "workbuddy_" + signIn.State,
			State:     "waiting",
			URL:       signIn.AuthURL,
			ExpiresAt: time.Now().Add(loginTimeout).UTC().Format(time.RFC3339),
		},
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		cancel()
		return subscription.State{}, workbuddyai.Failure("cancelled")
	}
	s.current = job
	view := job.view
	s.mu.Unlock()
	go s.complete(waitCtx, job, signIn.State)
	return view, nil
}

// complete 等待用户完成浏览器登录，核对远端身份后保存受保护令牌。
func (s *Service) complete(ctx context.Context, job *session, state string) {
	defer close(job.done)
	defer job.cancel()
	var item account.Item
	var err error
	token, err := s.client.WaitForToken(ctx, state)
	if err == nil {
		s.progress(job, "exchanging")
		operation, cancel := context.WithTimeout(ctx, operationTimeout)
		var identity workbuddyai.Identity
		identity, err = s.client.ReadIdentity(operation, token)
		cancel()
		if err == nil {
			cred := credentialsFrom(token, identity)
			if job.input.AccountID != "" {
				var current account.Item
				current, err = s.accounts.Metadata(ctx, job.input.AccountID)
				if err == nil && current.AuthAccountID != cred.UID {
					err = workbuddyai.Failure("identity_changed")
				}
			}
			if err == nil {
				s.progress(job, "saving")
				s.ops.Lock()
				if ctx.Err() != nil {
					err = workbuddyai.Failure("cancelled")
				} else {
					item, err = s.accounts.SaveWorkBuddyAccount(ctx, job.input.ProviderID, job.input.AccountID, job.input.Name, cred)
				}
				s.ops.Unlock()
			}
		}
	}
	status, code := "completed", ""
	if err != nil {
		status, code = "failed", errorCode(err, ctx)
	}
	s.mu.Lock()
	job.view.State, job.view.ErrorCode, job.view.URL = status, code, ""
	if item.ID != "" {
		job.view.Account = &item
	}
	s.mu.Unlock()
	s.notify()
}

// errorCode 只返回稳定错误码，不把上游原文带到界面。
func errorCode(err error, ctx context.Context) string {
	if ctx.Err() == context.DeadlineExceeded {
		return "login_expired"
	}
	if ctx.Err() == context.Canceled {
		return "cancelled"
	}
	if code := workbuddyai.ErrorCode(err); code != "" {
		return code
	}
	return "operation_failed"
}

func credentialsFrom(token workbuddyai.Token, identity workbuddyai.Identity) workbuddyai.Credentials {
	return workbuddyai.Credentials{
		AccessToken:      token.AccessToken,
		RefreshToken:     token.RefreshToken,
		ExpiresAt:        token.ExpiresAt,
		RefreshExpiresAt: token.RefreshExpiresAt,
		Domain:           token.Domain,
		TokenType:        token.TokenType,
		UID:              identity.UID,
		EnterpriseID:     identity.EnterpriseID,
		AccountName:      identity.Name,
		Plan:             identity.Plan,
	}
}

func (s *Service) progress(job *session, state string) {
	s.mu.Lock()
	job.view.State, job.view.URL = state, ""
	s.mu.Unlock()
	s.notify()
}
