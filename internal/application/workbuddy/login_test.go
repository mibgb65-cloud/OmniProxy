package workbuddy

import (
	"context"
	"errors"
	"testing"
	"time"

	"omniproxy/internal/provider/workbuddyai"
)

// 登录失败只暴露稳定错误码：超时与取消优先，其次是适配器的码，未知一律 operation_failed。
func TestErrorCodeMapsContextBeforeUpstream(t *testing.T) {
	expired, cancelExpired := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancelExpired()
	if code := errorCode(workbuddyai.Failure("sign_in_expired"), expired); code != "login_expired" {
		t.Fatalf("deadline code = %q", code)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if code := errorCode(errors.New("boom"), cancelled); code != "cancelled" {
		t.Fatalf("cancel code = %q", code)
	}
	if code := errorCode(workbuddyai.Failure("sign_in_expired"), context.Background()); code != "sign_in_expired" {
		t.Fatalf("stable code = %q", code)
	}
	if code := errorCode(errors.New("private upstream text"), context.Background()); code != "operation_failed" {
		t.Fatalf("unknown code = %q", code)
	}
}

// 登录令牌与身份字段必须一起保存；缺失身份会让重复登录认不出同一账号。
func TestCredentialsFromKeepsIdentityAndTokens(t *testing.T) {
	expires := time.Now().Add(time.Hour).UTC()
	token := workbuddyai.Token{
		AccessToken:      "access",
		RefreshToken:     "refresh",
		ExpiresAt:        expires,
		RefreshExpiresAt: expires.Add(24 * time.Hour),
		Domain:           "example.com",
		TokenType:        "Bearer",
	}
	identity := workbuddyai.Identity{UID: "uid-1", EnterpriseID: "ent-1", Name: "Tester", Plan: "Pro"}
	cred := credentialsFrom(token, identity)
	if !cred.Valid() || !cred.SignedIn() {
		t.Fatal("credentials from a token are not valid")
	}
	if cred.UID != "uid-1" || cred.EnterpriseID != "ent-1" || cred.AccountName != "Tester" || cred.Plan != "Pro" {
		t.Fatalf("identity lost: %+v", cred)
	}
	if cred.AccessToken != "access" || cred.RefreshToken != "refresh" || !cred.ExpiresAt.Equal(expires) {
		t.Fatalf("tokens lost: %+v", cred)
	}
}
