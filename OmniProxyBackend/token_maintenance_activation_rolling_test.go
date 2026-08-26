package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"omniproxy/internal/config"
	"omniproxy/internal/logs"
	"omniproxy/internal/storage"
	"omniproxy/internal/token"
	"path/filepath"
	"testing"
	"time"
)

func TestManualCodexUsageActivationActivatesRollingUnusedFiveHourWindow(t *testing.T) {
	usageChecks := 0
	activationCalls := 0
	fiveHourResetAt := time.Now().Add(5 * time.Hour).Unix()
	weeklyResetAt := time.Now().Add(6 * 24 * time.Hour).Unix()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/backend-api/wham/usage":
			usageChecks++
			resetAt := fiveHourResetAt
			if usageChecks == 1 {
				resetAt = time.Now().Add(5 * time.Hour).Unix()
			}
			_, _ = fmt.Fprintf(w, `{"plan_type":"plus","rate_limit":{"primary_window":{"used_percent":0,"reset_at":%d,"window_minutes":300},"secondary_window":{"used_percent":8,"reset_at":%d,"window_minutes":10080}}}`, resetAt, weeklyResetAt)
		case r.Method == http.MethodPost && r.URL.Path == "/backend-api/codex/responses":
			activationCalls++
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\"}}\n\ndata: [DONE]\n\n"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForMainTest(t, "manual-rolling@example.com"),
	})
	if err != nil {
		t.Fatal(err)
	}
	app := &appServer{
		cfg:    config.Config{ProxyPort: 3000, ControlPort: 3890, CodexBaseURL: upstream.URL + "/backend-api/codex", SwitchThreshold: 15, MaxRetries: 1, CodexUsageEndpoint: upstream.URL + "/backend-api/wham/usage"},
		tokens: manager,
		logs:   logs.NewRecorder(10),
	}

	result, err := app.activateCodexUsageManually(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("activateCodexUsageManually: %v", err)
	}
	if !result.Activated || result.AlreadyActive {
		t.Fatalf("unexpected result: %#v", result)
	}
	if activationCalls != 1 || usageChecks != 2 {
		t.Fatalf("activation calls = %d, usage checks = %d", activationCalls, usageChecks)
	}
	if result.Token.Usage.PrimaryActivationPending {
		t.Fatalf("five hour window still marked pending: %#v", result.Token.Usage)
	}
}
