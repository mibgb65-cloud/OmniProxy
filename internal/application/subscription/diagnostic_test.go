package subscription

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestDiagnosticContainsOnlyLatestStateAndErrorCode(t *testing.T) {
	s := &Service{history: map[string]State{
		"private-session-old": {State: "cancelled", ExpiresAt: "2026-10-08T10:00:00Z", URL: "private-location"},
		"private-session-new": {State: "failed", ErrorCode: "protocol_failed", ExpiresAt: "2026-10-08T11:00:00Z", UserCode: "private-code"},
	}}
	info := s.DiagnosticState()
	if info.State != "failed" || info.Code != "protocol_failed" {
		t.Fatal("wrong latest diagnostic")
	}
	s.current = &session{view: State{State: "waiting", URL: "private-current-location", ID: "private-current-session"}}
	info = s.DiagnosticState()
	if info.State != "waiting" || info.Code != "" {
		t.Fatal("current session diagnostic not preferred")
	}
	data, err := json.Marshal(info)
	if err != nil || strings.Contains(string(data), "private") {
		t.Fatal("diagnostic exposed authorization data")
	}
}

func TestDiagnosticErrorCodeDoesNotExposeRawMessages(t *testing.T) {
	for _, code := range []string{"account_unavailable", "initialization_failed", "request_failed", "protocol_failed", "login_failed", "operation_cancelled"} {
		if got := ErrorCode(errors.New("subscription: " + code)); got != code {
			t.Fatal("known failure lost", got)
		}
	}
	if ErrorCode(nil) != "" || ErrorCode(errors.New("private-authorization-value")) != "operation_failed" || ErrorCode(errors.New("private prefix: subscription: account_unavailable")) != "operation_failed" {
		t.Fatal("dynamic error accepted")
	}
}
