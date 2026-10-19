package wails

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestSubscriptionBeginFailureLogsStableCodeWithoutRawMessage(t *testing.T) {
	var out bytes.Buffer
	service := &AppService{log: slog.New(slog.NewTextHandler(&out, nil))}
	service.logSubscriptionBeginError(errors.New("subscription: account_unavailable"))
	service.logSubscriptionBeginError(errors.New("private-authorization-marker"))
	service.logSubscriptionBeginError(nil)
	text := out.String()
	if strings.Count(text, "subscription authorization request failed") != 2 || !strings.Contains(text, "code=account_unavailable") || !strings.Contains(text, "code=operation_failed") || strings.Contains(text, "private-authorization-marker") {
		t.Fatal("unsafe or missing begin diagnostics")
	}
}

func TestSubscriptionQuotaFailureLogsOnlySafeClassification(t *testing.T) {
	var out bytes.Buffer
	service := &AppService{log: slog.New(slog.NewTextHandler(&out, nil))}
	service.logSubscriptionQuotaError(errors.New("subscription: quota_timeout"))
	service.logSubscriptionQuotaError(errors.New("private-quota-marker"))
	service.logSubscriptionQuotaError(nil)
	text := out.String()
	if strings.Count(text, "subscription quota query failed") != 2 || !strings.Contains(text, "operation=quota") || !strings.Contains(text, "stage=prepare") || !strings.Contains(text, "code=quota_timeout") || !strings.Contains(text, "cause_code=operation_failed") || strings.Contains(text, "private-quota-marker") {
		t.Fatal("unsafe or missing quota diagnostics")
	}
}
