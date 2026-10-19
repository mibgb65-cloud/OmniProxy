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
