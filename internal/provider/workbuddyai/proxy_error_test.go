package workbuddyai

import (
	"errors"
	"testing"

	"omniproxy/internal/core/proxy"
)

// 业务码拒绝只有翻译成边界错误，网关的失败分类与请求记录才能看到真实原因。
func TestProxyErrorKeepsStableCodeAndStatus(t *testing.T) {
	cases := []struct {
		code   string
		status int
	}{
		{CodeModelNotEntitled, 403},
		{CodeChannelRejected, 400},
		{CodeSignatureInvalid, 400},
		{CodeUserAgentRefused, 400},
		{CodeSignInExpired, 401},
		{CodeRefreshRejected, 401},
		{CodeRateLimited, 429},
		{CodeCredentialUnavailable, 503},
		{CodeUpstreamUnavailable, 502},
		{CodeInvalidResponse, 502},
	}
	for _, test := range cases {
		t.Run(test.code, func(t *testing.T) {
			err := ProxyError(Failure(test.code))
			var failure *proxy.Error
			if !errors.As(err, &failure) || failure.Code != test.code || failure.StatusCode != test.status {
				t.Fatalf("ProxyError(%s) = %+v", test.code, err)
			}
			if ErrorCode(err) != test.code {
				t.Fatalf("ErrorCode lost stable code: %v", err)
			}
		})
	}
}

func TestProxyErrorLeavesOtherErrorsUntouched(t *testing.T) {
	boundary := proxy.Failure("model_not_entitled", 403)
	if got := ProxyError(boundary); got != boundary {
		t.Fatalf("boundary error rewritten: %v", got)
	}
	unrelated := errors.New("workbuddy_ai: unknown")
	if got := ProxyError(unrelated); got != unrelated {
		t.Fatalf("unrelated error rewritten: %v", got)
	}
	if ProxyError(nil) != nil {
		t.Fatal("nil error must stay nil")
	}
}
