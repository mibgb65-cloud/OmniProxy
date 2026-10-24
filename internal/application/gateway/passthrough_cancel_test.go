package gateway

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"omniproxy/internal/core/proxy"
)

type cancellingSubscription struct {
	cancel context.CancelFunc
	calls  int
}

type timedOutSubscription struct{}

func (timedOutSubscription) UseToken(context.Context, string, func([]byte, string) error) error {
	return context.DeadlineExceeded
}

func (s *cancellingSubscription) UseToken(context.Context, string, func([]byte, string) error) error {
	s.calls++
	s.cancel()
	return errors.New("private-credential-error")
}

func TestPassthroughCancellationDoesNotRotateOrCorruptCredentialState(t *testing.T) {
	s, db, _, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("不应发送协议请求")
		return nil, nil
	}))
	if _, err := db.Exec(`UPDATE accounts SET provider_id='builtin_openai',auth_method='subscription_login',auth_email=id||'@example.invalid',auth_account_id='workspace-'||id,auth_state='authenticated'`); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sub := &cancellingSubscription{cancel: cancel}
	s.SetSubscription(sub)
	session, err := s.OpenPassthrough(ctx, proxy.PassthroughRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := session.Next(ctx); !errors.Is(err, context.Canceled) {
			t.Fatal("取消被误报成凭据或配置损坏", err)
		}
	}
	if sub.calls != 1 || session.(*passthroughSession).credentialFailed {
		t.Fatal("请求结束后仍遍历账号或污染了凭据状态")
	}
}

func TestPassthroughInternalPreparationTimeoutIsNotCredentialFailure(t *testing.T) {
	s, db, _, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("不应发送协议请求")
		return nil, nil
	}))
	if _, err := db.Exec(`UPDATE accounts SET provider_id='builtin_openai',auth_method='subscription_login',auth_email=id||'@example.invalid',auth_account_id='workspace-'||id,auth_state='authenticated'`); err != nil {
		t.Fatal(err)
	}
	s.SetSubscription(timedOutSubscription{})
	session, err := s.OpenPassthrough(context.Background(), proxy.PassthroughRequest{})
	if err != nil {
		t.Fatal(err)
	}
	_, err = session.Next(context.Background())
	var failure *proxy.Error
	if !errors.As(err, &failure) || failure.Code != "credential_prepare_timeout" || failure.StatusCode != 504 ||
		session.(*passthroughSession).credentialFailed {
		t.Fatal("内部准备期限被误报为凭据损坏", err)
	}
}
