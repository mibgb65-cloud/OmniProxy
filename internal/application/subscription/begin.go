package subscription

import (
	"context"
	"os"
	"path/filepath"
	"time"
)

// Begin 启动官方授权，返回浏览器地址或一次性设备码。
func (s *Service) Begin(ctx context.Context, input Input) (State, error) {
	if err := s.operations.Lock(ctx); err != nil {
		return State{}, failure("operation_cancelled")
	}
	defer s.operations.Unlock()
	if input.Mode != "browser" && input.Mode != "device" || len(input.Name) > 256 || !utf8Name(input.Name) {
		return State{}, failure("invalid_input")
	}
	if err := s.accounts.CheckSubscriptionProvider(ctx, input.ProviderID); err != nil {
		return State{}, failure("invalid_provider")
	}
	if err := s.recoverPending(ctx); err != nil {
		return State{}, err
	}
	profile, err := randomID()
	if err != nil {
		return State{}, err
	}
	if input.AccountID != "" {
		item, _, err := s.accounts.SubscriptionProfile(ctx, input.AccountID)
		if err != nil || item.ProviderID != input.ProviderID {
			return State{}, failure("invalid_account")
		}
	}
	id, err := randomID()
	if err != nil {
		return State{}, err
	}
	waitCtx, cancel := context.WithTimeout(context.Background(), loginTimeout)
	job := &session{input: input, profile: profile, cancel: cancel, done: make(chan struct{}), view: State{ID: id, State: "starting", ExpiresAt: time.Now().Add(loginTimeout).UTC().Format(time.RFC3339)}}
	s.mu.Lock()
	if s.closed || s.current != nil {
		s.mu.Unlock()
		cancel()
		return State{}, failure("busy")
	}
	s.current = job
	s.mu.Unlock()
	initialCtx, stopInitial := context.WithTimeout(waitCtx, operationTimeout)
	defer stopInitial()
	stopCaller := context.AfterFunc(ctx, stopInitial)
	defer stopCaller()
	path, err := s.profilePath(profile)
	if err != nil {
		close(job.done)
		s.finish(job, "failed", "invalid_input", nil)
		return State{}, err
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		close(job.done)
		s.finish(job, "failed", "initialization_failed", nil)
		return State{}, failure("initialization_failed")
	}
	if err := os.WriteFile(filepath.Join(path, "omniproxy-pending"), nil, 0600); err != nil {
		close(job.done)
		s.finish(job, "failed", "initialization_failed", nil)
		return State{}, failure("initialization_failed")
	}
	client, err := s.open(initialCtx, profile)
	if err != nil {
		close(job.done)
		s.finish(job, "failed", safeCode(err), nil)
		return State{}, err
	}
	job.client = client
	login, err := client.Begin(initialCtx, input.Mode)
	if err != nil {
		code := safeCode(err)
		if s.cleanCancelled(job) != nil {
			code = "cleanup_failed"
		}
		client.Close()
		close(job.done)
		s.finish(job, "failed", code, nil)
		return State{}, failure(code)
	}
	job.loginID = login.ID
	s.mu.Lock()
	if s.closed || waitCtx.Err() != nil {
		s.mu.Unlock()
		cleanupErr := s.cleanCancelled(job)
		client.Close()
		code := "operation_cancelled"
		if cleanupErr != nil {
			code = "cleanup_failed"
		}
		close(job.done)
		s.finish(job, "cancelled", code, nil)
		return State{}, failure(code)
	}
	job.view.State = "waiting"
	job.view.URL = login.AuthURL
	if input.Mode == "device" {
		job.view.URL = login.VerificationURL
	}
	job.view.UserCode = login.UserCode
	view := job.view
	s.mu.Unlock()
	go s.complete(waitCtx, job)
	s.changed()
	return view, nil
}
