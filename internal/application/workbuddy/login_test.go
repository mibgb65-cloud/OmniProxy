package workbuddy

import (
	"context"
	"errors"
	"testing"
	"time"

	"omniproxy/internal/application/account"
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

// 账号层失败必须换算成各自的码：保存失败可重试，版本冲突要刷新，非法输入要重开。
func TestErrorCodeMapsAccountLayerFailures(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name string
		err  error
		want string
	}{
		{"save", account.ErrWorkBuddySave, "save_failed"},
		{"conflict", account.ErrConflict, "configuration_changed"},
		{"invalid", account.ErrInvalidInput, "invalid_input"},
		{"joined save", errors.Join(account.ErrWorkBuddySave, errors.New("cleanup failed")), "save_failed"},
	}
	for _, item := range cases {
		if code := errorCode(item.err, ctx); code != item.want {
			t.Fatalf("%s code = %q, want %q", item.name, code, item.want)
		}
	}
}

// 登录在后台线程结束，失败必须交给诊断钩子，且只带阶段与稳定码。
func TestReportSendsStageAndCode(t *testing.T) {
	service := &Service{}
	var message string
	var args []any
	service.SetWarn(func(text string, values ...any) {
		message, args = text, values
	})
	service.report("save", "save_failed")
	if message == "" || len(args) != 6 || args[3] != "save" || args[5] != "save_failed" {
		t.Fatalf("report = %q %v", message, args)
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
