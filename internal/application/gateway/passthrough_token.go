package gateway

import (
	"context"
	"net/http"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/codex"
)

// codexAuthAccess 只在回调内交出网关自己保管的 auth.json，并可回写续期结果。
type codexAuthAccess interface {
	UseCodexAuthAt(context.Context, string, string, func([]byte) error) error
	ReplaceCodexAuthTokensAt(context.Context, string, string, []byte, []byte) error
}

// passthroughAttempt 借出一次令牌并填好上游鉴权头；令牌只活在这一份头部里。
// 两种凭据来源都收敛到同一个结果：Bearer 访问令牌 + 官方账号标识。
func (s *Service) passthroughAttempt(ctx context.Context, t target, model, base string) (proxy.PassthroughAttempt, error) {
	if t.authMethod == account.AuthCodexFile {
		return s.attemptFromCodexAuth(ctx, t, model, base)
	}
	var attempt proxy.PassthroughAttempt
	err := s.useToken(ctx, t, func(key []byte, officialID string) error {
		attempt = newPassthroughAttempt(t, base, model, officialID, string(key))
		return nil
	})
	if err != nil {
		return proxy.PassthroughAttempt{}, err
	}
	return attempt, nil
}

func newPassthroughAttempt(t target, base, model, accountID, accessToken string) proxy.PassthroughAttempt {
	header := make(http.Header, 2)
	header.Set("Authorization", "Bearer "+accessToken)
	header.Set("ChatGPT-Account-ID", accountID)
	return proxy.PassthroughAttempt{Header: header, BaseURL: base, AccountID: t.accountID, Model: model, Snapshot: t.snapshot}
}

// attemptFromCodexAuth 走网关自己保管的 auth.json：需要时先自行续期，再把
// 新令牌写回保护存储（v1 的 UpdateTokenValue 语义）。
func (s *Service) attemptFromCodexAuth(ctx context.Context, t target, model, base string) (proxy.PassthroughAttempt, error) {
	stage := proxy.BeginSocketStage(ctx, "credential_queue")
	err := s.codexCredentials.LockAccount(ctx, t.accountID)
	stage(err)
	if err != nil {
		return proxy.PassthroughAttempt{}, err
	}
	defer s.codexCredentials.UnlockAccount(t.accountID)
	access, ok := s.accounts.(codexAuthAccess)
	if !ok {
		return proxy.PassthroughAttempt{}, proxy.Failure("invalid_configuration", 503)
	}
	var attempt proxy.PassthroughAttempt
	err = access.UseCodexAuthAt(ctx, t.accountID, t.updatedAt, func(raw []byte) error {
		refreshed, changed, err := s.refreshCodexAuth(ctx, raw)
		if err != nil {
			return err
		}
		if changed {
			defer clear(refreshed)
		}
		if changed {
			if err := access.ReplaceCodexAuthTokensAt(ctx, t.accountID, t.updatedAt, raw, refreshed); err != nil {
				s.warn("codex token refresh not persisted", "code", "refresh_persist_failed")
				return proxy.Failure("route_configuration_changed", 503)
			}
		}
		auth, err := codex.ParseAuthFile(refreshed)
		if err != nil {
			return proxy.Failure("invalid_configuration", 503)
		}
		attempt = newPassthroughAttempt(t, base, model, auth.AccountID, auth.AccessToken)
		return nil
	})
	if err != nil {
		discardAttemptHeaders(attempt.Header)
		return proxy.PassthroughAttempt{}, err
	}
	return attempt, nil
}

func discardAttemptHeaders(header http.Header) {
	for key, values := range header {
		clear(values)
		delete(header, key)
	}
}

// refreshCodexAuth 用池账号自己的 refresh_token 续期；失败时继续用现有令牌，
// 让上游的明确拒绝去触发换号，而不是把请求卡在续期上。
func (s *Service) refreshCodexAuth(ctx context.Context, raw []byte) ([]byte, bool, error) {
	now := time.Now()
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	if !codex.AuthNeedsRefresh(string(raw), now) {
		return raw, false, nil
	}
	refreshCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	updated, changed, err := codex.RefreshAuthJSON(refreshCtx, nil, string(raw), now)
	if err != nil || !changed {
		if ctx.Err() != nil {
			return nil, false, ctx.Err()
		}
		if refreshCtx.Err() != nil {
			return nil, false, proxy.Failure("credential_prepare_timeout", 504)
		}
		s.warn("codex token refresh failed", "code", "refresh_failed")
		// 已知需要续期的材料不能当作健康凭据继续发送。
		return nil, false, proxy.Failure("credential_unavailable", 503)
	}
	return []byte(updated), true, nil
}

func (s *Service) warn(message, key, value string) {
	if s.log != nil {
		s.log.Warn(message, "module", "gateway", key, value)
	}
}

// codexLoginMethod 判定账号是否用 Codex 登录凭据接入网关。
func codexLoginMethod(method string) bool {
	return method == account.AuthSubscription || method == account.AuthCodexFile
}
