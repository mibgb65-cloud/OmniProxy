package subscription

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"omniproxy/internal/application/account"
)

func utf8Name(value string) bool {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > 80 {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func (s *Service) finish(job *session, state, code string, item *account.Item) {
	s.mu.Lock()
	if s.current == job {
		job.view.State = state
		job.view.ErrorCode = code
		job.view.URL = ""
		job.view.UserCode = ""
		job.view.Account = item
		s.history[job.view.ID] = job.view
		s.current = nil
		job.cancel()
		if len(s.history) > 8 {
			for id := range s.history {
				if id != job.view.ID {
					delete(s.history, id)
					break
				}
			}
		}
	}
	s.mu.Unlock()
	s.changed()
}

func (s *Service) complete(ctx context.Context, job *session) {
	defer job.cancel()
	defer func() { job.client.Close(); close(job.done) }()
	if err := job.client.Wait(ctx, job.loginID); err != nil {
		state, code := "failed", safeCode(err)
		if ctx.Err() == context.Canceled {
			state, code = "cancelled", ""
		}
		if cleanupErr := s.cleanCancelled(job); cleanupErr != nil {
			code = "cleanup_failed"
		}
		s.finish(job, state, code, nil)
		return
	}
	readCtx, cancel := context.WithTimeout(context.Background(), operationTimeout)
	// 完成通知之后强制同步官方授权缓存；不能依赖登录前的账号快照。
	identity, err := job.client.Account(readCtx, true)
	cancel()
	metadataPending := err != nil || identity == nil
	if metadataPending && job.input.AccountID != "" {
		code := "account_unavailable"
		if cleanupErr := s.cleanCancelled(job); cleanupErr != nil {
			code = "cleanup_failed"
		}
		s.finish(job, "failed", code, nil)
		return
	}
	// 身份读取超时后仍需独立保存成功授权，不能复用已经取消的读取 context。
	saveCtx, stopSave := context.WithTimeout(context.Background(), operationTimeout)
	defer stopSave()
	meta := account.SubscriptionMetadata{State: "unknown"}
	var accountIDErr error
	if !metadataPending {
		meta = account.SubscriptionMetadata{Email: identity.Email, Plan: identity.Plan, State: "authenticated"}
		meta.AccountID, accountIDErr = readAccountID(ctx, job.client)
	}
	s.operations.Lock()
	defer s.operations.Unlock()
	s.mu.Lock()
	if s.current != job || ctx.Err() != nil {
		s.mu.Unlock()
		code := ""
		if cleanupErr := s.cleanCancelled(job); cleanupErr != nil {
			code = "cleanup_failed"
		}
		s.finish(job, "cancelled", code, nil)
		return
	}
	var item account.Item
	if job.input.AccountID != "" {
		previous, readErr := s.accounts.Metadata(saveCtx, job.input.AccountID)
		if readErr != nil || previous.AuthEmail != "" && !strings.EqualFold(previous.AuthEmail, identity.Email) {
			err = failure("identity_changed")
		} else {
			item, err = s.accounts.ReplaceSubscription(saveCtx, job.input.AccountID, job.profile, meta)
		}
	} else {
		item, err = s.accounts.CreateSubscription(saveCtx, job.input.ProviderID, job.input.Name, job.profile, meta)
	}
	s.mu.Unlock()
	if err != nil && item.ID == "" {
		code := safeCode(err)
		if cleanupErr := s.cleanCancelled(job); cleanupErr != nil {
			code = "cleanup_failed"
		}
		s.finish(job, "failed", code, nil)
		return
	}
	warning := err != nil
	if job.input.AccountID != "" {
		if cleanupErr := s.cleanRetired(saveCtx, item.ID); cleanupErr != nil {
			warning = true
		}
	}
	path, _ := s.profilePath(job.profile)
	if markerErr := os.Remove(filepath.Join(path, "omniproxy-pending")); markerErr != nil && !os.IsNotExist(markerErr) {
		warning = true
	}
	s.mu.Lock()
	job.view.CleanupWarning = warning
	s.mu.Unlock()
	code := ""
	if metadataPending {
		code = "account_metadata_pending"
	} else if accountIDErr != nil {
		code = "account_id_pending"
	}
	s.finish(job, "completed", code, &item)
}

// Cancel 不会注销用户默认 Codex 登录，仅取消该隔离会话。
func (s *Service) Cancel(id string) (State, error) {
	s.mu.Lock()
	job := s.current
	if job == nil || job.view.ID != id {
		s.mu.Unlock()
		return s.Read(id)
	}
	job.cancel()
	s.mu.Unlock()
	return s.Read(id)
}

// Close 终止正在进行的授权，退出应用不会清理已保存账号的系统凭证。
func (s *Service) Close() {
	s.mu.Lock()
	s.closed = true
	job := s.current
	if job != nil && job.cancel != nil {
		job.cancel()
	}
	s.mu.Unlock()
	if job != nil {
		<-job.done
	}
}
