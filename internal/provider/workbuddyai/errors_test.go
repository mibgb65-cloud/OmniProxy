package workbuddyai

import (
	"errors"
	"fmt"
	"testing"
)

func TestFailureKeepsStableCodes(t *testing.T) {
	cases := []string{
		CodeSignInExpired,
		CodeCredentialUnavailable,
		CodeUnauthorized,
		CodeChannelRejected,
		CodeSignatureInvalid,
		CodeModelNotEntitled,
		CodeUserAgentRefused,
		CodeRateLimited,
		CodeUpstreamUnavailable,
		CodeInvalidResponse,
		CodeRefreshRejected,
	}
	for _, want := range cases {
		err := Failure(want)
		if got := ErrorCode(err); got != want {
			t.Errorf("ErrorCode = %q, want %q", got, want)
		}
		if failure, ok := err.(*Error); !ok || failure.SignIn {
			t.Errorf("%q unexpectedly marked as sign-in failure", want)
		}
	}
}

func TestErrorCodeIgnoresForeignErrors(t *testing.T) {
	if code := ErrorCode(nil); code != "" {
		t.Fatalf("nil => %q", code)
	}
	if code := ErrorCode(errors.New("plain")); code != "" {
		t.Fatalf("plain => %q", code)
	}
	wrapped := fmt.Errorf("outer: %w", Failure(CodeRateLimited))
	if code := ErrorCode(wrapped); code != CodeRateLimited {
		t.Fatalf("wrapped => %q", code)
	}
}

func TestSignInExpiredMarksOnlyLoginFailures(t *testing.T) {
	if SignInExpired(nil) || SignInExpired(Failure(CodeUnauthorized)) || SignInExpired(errors.New("plain")) {
		t.Fatal("ordinary failures marked as sign-in expiry")
	}
	for _, code := range []string{CodeSignInExpired, CodeRefreshRejected} {
		err := signInFailure(code)
		if !SignInExpired(err) {
			t.Errorf("%q not marked as sign-in expiry", code)
		}
		if ErrorCode(fmt.Errorf("wrapped: %w", err)) != code {
			t.Errorf("%q lost through wrapping", code)
		}
	}
}
