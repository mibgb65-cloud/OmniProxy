package stepfun

import (
	"errors"
	"omniproxy/internal/core/proxy"
	"strings"
	"testing"
)

func TestResourceErrorsUseOnlyOfficialIdentifiers(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
	}{{402, "insufficient_credit"}, {429, "project_credit_limit_exceeded"}, {429, "member_project_credit_limit_exceeded"}} {
		err := resourceFailure(tc.status, strings.NewReader(`{"error":{"code":"`+tc.code+`","message":"private response must not escape"}}`))
		var value *proxy.Error
		if !errors.As(err, &value) || value.Code != "upstream_"+tc.code || strings.Contains(err.Error(), "private") {
			t.Fatalf("incorrect classification: %v", err)
		}
	}
	if err := resourceFailure(429, strings.NewReader(`{"error":{"code":"private account-specific message"}}`)); err != nil {
		t.Fatal("private error became a public code")
	}
	if err := resourceFailure(402, strings.NewReader(`{"error":{"code":"project_credit_limit_exceeded"}}`)); err != nil {
		t.Fatal("status mismatch classified as project limit")
	}
}
