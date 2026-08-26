package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"omniproxy/internal/config"
	"omniproxy/internal/history"
	"omniproxy/internal/logs"
	"omniproxy/internal/storage"
	"omniproxy/internal/token"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAddCodexTokenRefreshesUsage(t *testing.T) {
	var validationCalled bool
	var activationCalled bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			activationCalled = true
		}
		validationCalled = true
		if got := r.Header.Get("Authorization"); got != "Bearer codex-access-token" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"plan_type": "team",
			"rate_limit": {
				"primary_window": {"used_percent": 23, "reset_at": 1893456000},
				"secondary_window": {"used_percent": 41, "reset_at": 1894060800}
			}
		}`))
	}))
	defer upstream.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	app := &appServer{
		cfg: config.Config{
			ProxyPort:              3000,
			ControlPort:            3890,
			UpstreamBaseURL:        "https://api.openai.com",
			CodexAutoActivateUsage: true,
			SwitchThreshold:        15,
			MaxRetries:             1,
			CodexUsageEndpoint:     upstream.URL,
		},
		tokens: manager,
		logs:   logs.NewRecorder(10),
	}
	payload, err := json.Marshal(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForMainTest(t, "coder@example.com"),
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/tokens", strings.NewReader(string(payload)))
	res := httptest.NewRecorder()
	app.routes().ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d body=%s", res.Code, res.Body.String())
	}
	if !validationCalled {
		t.Fatal("expected codex usage validation to be called after add")
	}
	if activationCalled {
		t.Fatal("expected existing Codex usage windows not to trigger activation")
	}

	items := manager.List()
	if len(items) != 1 {
		t.Fatalf("expected 1 token, got %d", len(items))
	}
	if items[0].Remaining != 77 || items[0].Usage.PrimaryRemainingPercent != 77 || items[0].Usage.SecondaryRemainingPercent != 59 {
		t.Fatalf("expected usage to be refreshed after add, got remaining=%d usage=%#v", items[0].Remaining, items[0].Usage)
	}
}

func TestAddCodexTokenAutoActivatesMissingUsageWindows(t *testing.T) {
	usageChecks := 0
	activationCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/backend-api/wham/usage":
			usageChecks++
			w.Header().Set("Content-Type", "application/json")
			if usageChecks == 1 {
				_, _ = w.Write([]byte(`{"plan_type":"plus","rate_limit":{}}`))
				return
			}
			_, _ = w.Write([]byte(`{
				"plan_type":"plus",
				"rate_limit":{
					"primary_window":{"used_percent":1,"reset_at":1893456000,"window_minutes":300},
					"secondary_window":{"used_percent":1,"reset_at":1894060800,"window_minutes":10080}
				}
			}`))
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
	app := &appServer{
		cfg: config.Config{
			ProxyPort:              3000,
			ControlPort:            3890,
			CodexBaseURL:           upstream.URL + "/backend-api/codex",
			CodexAutoActivateUsage: true,
			SwitchThreshold:        15,
			MaxRetries:             1,
			CodexUsageEndpoint:     upstream.URL + "/backend-api/wham/usage",
		},
		tokens: manager,
		logs:   logs.NewRecorder(10),
	}

	created, err := app.createToken(context.Background(), token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForMainTest(t, "activate@example.com"),
	})
	if err != nil {
		t.Fatalf("createToken: %v", err)
	}
	if activationCalls != 1 || usageChecks != 2 {
		t.Fatalf("activation calls = %d, usage checks = %d", activationCalls, usageChecks)
	}
	updated, err := manager.Get(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Usage.PrimaryResetAt == 0 || updated.Usage.SecondaryResetAt == 0 {
		t.Fatalf("expected both usage windows after activation, got %#v", updated.Usage)
	}
}

func TestAddCodexTokenDoesNotAutoActivateWhenDisabled(t *testing.T) {
	usageChecks := 0
	activationCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			usageChecks++
			_, _ = w.Write([]byte(`{"plan_type":"plus","rate_limit":{}}`))
		case http.MethodPost:
			activationCalls++
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer upstream.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	app := &appServer{
		cfg: config.Config{
			ProxyPort:          3000,
			ControlPort:        3890,
			CodexBaseURL:       upstream.URL + "/backend-api/codex",
			SwitchThreshold:    15,
			MaxRetries:         1,
			CodexUsageEndpoint: upstream.URL + "/backend-api/wham/usage",
		},
		tokens: manager,
		logs:   logs.NewRecorder(10),
	}

	if _, err := app.createToken(context.Background(), token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForMainTest(t, "disabled@example.com"),
	}); err != nil {
		t.Fatalf("createToken: %v", err)
	}
	if usageChecks != 1 || activationCalls != 0 {
		t.Fatalf("usage checks = %d, activation calls = %d", usageChecks, activationCalls)
	}
}

func TestManualCodexUsageActivationBypassesDisabledSettingAndSkipsActiveWindows(t *testing.T) {
	usageChecks := 0
	activationCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/backend-api/wham/usage":
			usageChecks++
			_, _ = w.Write([]byte(`{
				"plan_type":"plus",
				"rate_limit":{
					"primary_window":{"used_percent":1,"reset_at":1893456000,"window_minutes":300},
					"secondary_window":{"used_percent":1,"reset_at":1894060800,"window_minutes":10080}
				}
			}`))
		case r.Method == http.MethodPost:
			activationCalls++
			w.WriteHeader(http.StatusNoContent)
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
		TokenValue:     codexAuthJSONForMainTest(t, "manual-active@example.com"),
	})
	if err != nil {
		t.Fatal(err)
	}
	app := &appServer{
		cfg: config.Config{
			ProxyPort:          3000,
			ControlPort:        3890,
			CodexBaseURL:       upstream.URL + "/backend-api/codex",
			SwitchThreshold:    15,
			MaxRetries:         1,
			CodexUsageEndpoint: upstream.URL + "/backend-api/wham/usage",
		},
		tokens: manager,
		logs:   logs.NewRecorder(10),
	}
	result, err := app.activateCodexUsageManually(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("activateCodexUsageManually: %v", err)
	}
	if result.Activated || !result.AlreadyActive {
		t.Fatalf("unexpected result: %#v", result)
	}
	if activationCalls != 0 || usageChecks != 1 {
		t.Fatalf("activation calls = %d, usage checks = %d", activationCalls, usageChecks)
	}
}

func TestManualCodexUsageActivationActivatesMissingWindowsWhenSettingDisabled(t *testing.T) {
	usageChecks := 0
	activationCalls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/backend-api/wham/usage":
			usageChecks++
			if usageChecks == 1 {
				_, _ = w.Write([]byte(`{"plan_type":"plus","rate_limit":{}}`))
				return
			}
			_, _ = w.Write([]byte(`{
				"plan_type":"plus",
				"rate_limit":{
					"primary_window":{"used_percent":1,"reset_at":1893456000,"window_minutes":300},
					"secondary_window":{"used_percent":1,"reset_at":1894060800,"window_minutes":10080}
				}
			}`))
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
		TokenValue:     codexAuthJSONForMainTest(t, "manual-missing@example.com"),
	})
	if err != nil {
		t.Fatal(err)
	}
	app := &appServer{
		cfg: config.Config{
			ProxyPort:          3000,
			ControlPort:        3890,
			CodexBaseURL:       upstream.URL + "/backend-api/codex",
			SwitchThreshold:    15,
			MaxRetries:         1,
			CodexUsageEndpoint: upstream.URL + "/backend-api/wham/usage",
		},
		tokens: manager,
		logs:   logs.NewRecorder(10),
	}
	historyRecorder, err := history.NewRecorder(storage.NewJSONStore[[]history.Entry](filepath.Join(t.TempDir(), "history.json")), 100)
	if err != nil {
		t.Fatal(err)
	}
	app.history = historyRecorder

	result, err := app.activateCodexUsageManually(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("activateCodexUsageManually: %v", err)
	}
	if !result.Activated || result.AlreadyActive || result.Token.Usage.PrimaryResetAt == 0 || result.Token.Usage.SecondaryResetAt == 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if activationCalls != 1 || usageChecks != 2 {
		t.Fatalf("activation calls = %d, usage checks = %d", activationCalls, usageChecks)
	}
	entries := historyRecorder.List(history.Filter{Search: "codex usage activation", Limit: 10})
	if len(entries) != 2 || !strings.Contains(entries[1].Message, "started") || !strings.Contains(entries[0].Message, "completed") {
		t.Fatalf("expected persisted start and completion history, got %#v", entries)
	}
}

func TestTokenRefreshEndpointForcesCodexRefresh(t *testing.T) {
	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForMainTestWithRefreshToken(t, "coder@example.com", "account-123", "old-access-token", "old-refresh-token"),
	})
	if err != nil {
		t.Fatal(err)
	}

	newIDToken := codexIDTokenForMainTest(t, "coder@example.com")
	refreshBody, err := json.Marshal(map[string]any{
		"access_token":  "new-access-token",
		"id_token":      newIDToken,
		"refresh_token": "new-refresh-token",
		"expires_in":    3600,
	})
	if err != nil {
		t.Fatal(err)
	}
	originalTransport := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://auth.openai.com/oauth/token" {
			t.Fatalf("unexpected refresh endpoint: %s", req.URL.String())
		}
		if req.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", req.Method)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		values, err := url.ParseQuery(string(body))
		if err != nil {
			t.Fatal(err)
		}
		if values.Get("refresh_token") != "old-refresh-token" {
			t.Fatalf("unexpected refresh_token: %q", values.Get("refresh_token"))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(string(refreshBody))),
		}, nil
	})
	defer func() {
		http.DefaultTransport = originalTransport
	}()

	app := &appServer{
		cfg: config.Config{
			ProxyPort:       3000,
			ControlPort:     3890,
			UpstreamBaseURL: "https://api.openai.com",
			SwitchThreshold: 15,
			MaxRetries:      1,
		},
		tokens: manager,
		logs:   logs.NewRecorder(10),
	}

	req := httptest.NewRequest(http.MethodPost, "/api/tokens/"+item.ID+"/refresh", nil)
	res := httptest.NewRecorder()
	app.routes().ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", res.Code, res.Body.String())
	}
	updated, err := manager.Get(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(updated.TokenValue, "new-access-token") || !strings.Contains(updated.TokenValue, "new-refresh-token") {
		t.Fatalf("expected stored auth token to be refreshed, got %s", updated.TokenValue)
	}
}

func TestImportAPIKeysParsesCommentsAndGeneratesNames(t *testing.T) {
	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Add(token.UpsertRequest{Name: "existing", Provider: token.ProviderOpenAI, TokenValue: "sk-existing-token-value"}); err != nil {
		t.Fatal(err)
	}
	app := &appServer{
		cfg: config.Config{
			ProxyPort:       3000,
			ControlPort:     3890,
			UpstreamBaseURL: "https://api.openai.com",
			SwitchThreshold: 15,
			MaxRetries:      1,
		},
		tokens: manager,
		logs:   logs.NewRecorder(10),
	}

	result, err := app.importAPIKeys(apiKeyBatchImportRequest{
		Provider: token.ProviderOpenAI,
		TokenText: strings.Join([]string{
			"# comment only",
			"sk-aa0aeaf480484648a8a93d672d76334d  # balance: 10.14 CNY",
			"sk-aa0aeff80f84d05a13fa2bebd27159c  # balance: 0.24 USD",
			"sk-aa0aeaf480484648a8a93d672d76334d",
			"sk-existing-token-value # already saved",
			"tiny",
		}, "\n"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.CreatedCount != 2 {
		t.Fatalf("expected 2 imported keys, got %#v", result)
	}
	if len(result.Skipped) != 3 {
		t.Fatalf("expected duplicate, existing, and invalid lines to be skipped, got %#v", result.Skipped)
	}

	items := manager.List()
	if len(items) != 3 {
		t.Fatalf("expected 3 total tokens, got %d", len(items))
	}
	byName := map[string]token.Token{}
	for _, item := range items {
		byName[item.Name] = item
	}
	if byName["sk-aa0ae"].TokenValue != "sk-aa0aeaf480484648a8a93d672d76334d" ||
		byName["sk-aa0ae-2"].TokenValue != "sk-aa0aeff80f84d05a13fa2bebd27159c" {
		t.Fatalf("unexpected generated tokens: %#v", items)
	}
}

func TestStartupRefreshesCodexUsage(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"plan_type": "team",
			"rate_limit": {
				"primary_window": {"used_percent": 35, "reset_at": 1777299888},
				"secondary_window": {"used_percent": 12, "reset_at": 1777798105}
			}
		}`))
	}))
	defer upstream.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	item, err := manager.Add(token.UpsertRequest{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     codexAuthJSONForMainTest(t, "startup@example.com"),
	})
	if err != nil {
		t.Fatal(err)
	}
	app := &appServer{
		cfg: config.Config{
			ProxyPort:          3000,
			ControlPort:        3890,
			UpstreamBaseURL:    "https://api.openai.com",
			SwitchThreshold:    15,
			MaxRetries:         1,
			CodexUsageEndpoint: upstream.URL,
		},
		tokens: manager,
		logs:   logs.NewRecorder(10),
	}

	app.refreshCodexUsageOnStartup(context.Background())

	updated, err := manager.Get(item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Remaining != 65 || updated.Usage.PrimaryRemainingPercent != 65 || updated.Usage.SecondaryRemainingPercent != 88 {
		t.Fatalf("expected startup usage refresh, got remaining=%d usage=%#v", updated.Remaining, updated.Usage)
	}
}

func TestCurrentTokenQuotaRefreshUsesLatestProxyUsage(t *testing.T) {
	var seenAuth []string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = append(seenAuth, r.Header.Get("Authorization"))
		switch r.Header.Get("Authorization") {
		case "Bearer sk-current-token":
			w.Header().Set("x-ratelimit-remaining-tokens", "64")
		case "Bearer sk-older-token":
			w.Header().Set("x-ratelimit-remaining-tokens", "12")
		default:
			t.Fatalf("unexpected auth header: %q", r.Header.Get("Authorization"))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer upstream.Close()

	manager, err := token.NewManager(storage.NewJSONStore[[]token.Token](filepath.Join(t.TempDir(), "tokens.json")), 15)
	if err != nil {
		t.Fatal(err)
	}
	older, err := manager.Add(token.UpsertRequest{Name: "older", Provider: token.ProviderOpenAI, TokenValue: "sk-older-token"})
	if err != nil {
		t.Fatal(err)
	}
	current, err := manager.Add(token.UpsertRequest{Name: "current", Provider: token.ProviderOpenAI, TokenValue: "sk-current-token"})
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.RecordProxyUsage(older.ID, token.TokenConsumption{TotalTokens: 1}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	if err := manager.RecordProxyUsage(current.ID, token.TokenConsumption{TotalTokens: 1}); err != nil {
		t.Fatal(err)
	}
	historyRecorder, err := history.NewRecorder(storage.NewJSONStore[[]history.Entry](filepath.Join(t.TempDir(), "history.json")), 100)
	if err != nil {
		t.Fatal(err)
	}

	app := &appServer{
		cfg: config.Config{
			ProxyPort:       3000,
			ControlPort:     3890,
			OpenAIBaseURL:   upstream.URL,
			SwitchThreshold: 15,
			MaxRetries:      1,
		},
		tokens:  manager,
		logs:    logs.NewRecorder(10),
		history: historyRecorder,
	}

	app.refreshCurrentTokenUsage(context.Background())

	if len(seenAuth) != 1 || seenAuth[0] != "Bearer sk-current-token" {
		t.Fatalf("expected only current token to be refreshed, got %#v", seenAuth)
	}
	updatedCurrent, err := manager.Get(current.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updatedCurrent.Remaining != 64 || updatedCurrent.Usage.APIRemaining != 64 {
		t.Fatalf("expected current token quota to refresh, got remaining=%d usage=%#v", updatedCurrent.Remaining, updatedCurrent.Usage)
	}
	updatedOlder, err := manager.Get(older.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updatedOlder.Remaining != 100 || updatedOlder.Usage.APIRemaining != 0 {
		t.Fatalf("older token should not be refreshed, got remaining=%d usage=%#v", updatedOlder.Remaining, updatedOlder.Usage)
	}
	entries := historyRecorder.List(history.Filter{Limit: 10})
	if len(entries) != 1 {
		t.Fatalf("expected quota refresh history entry, got %#v", entries)
	}
	if entries[0].Path != "/maintenance/current-token-quota-refresh" || entries[0].Protocol != "quota-refresh" || entries[0].TokenName != "current" || entries[0].Status != http.StatusOK {
		t.Fatalf("unexpected quota refresh history entry: %#v", entries[0])
	}
	if !strings.Contains(entries[0].Message, "remaining=64%") {
		t.Fatalf("expected remaining quota in history message, got %q", entries[0].Message)
	}
}

func TestCurrentQuotaRefreshCandidateSkipsValidationOnlyUsage(t *testing.T) {
	now := time.Now()
	validationOnly := token.Token{
		ID:         "validation-only",
		TokenValue: "sk-validation-only",
		LastUsedAt: &now,
		Status:     token.StatusActive,
	}
	if selected, ok := currentQuotaRefreshCandidate([]token.Token{validationOnly}, now); ok {
		t.Fatalf("expected validation-only token to be skipped, got %#v", selected)
	}
}

func TestCurrentQuotaRefreshCandidateSkipsBackoff(t *testing.T) {
	now := time.Now()
	nextCheckAt := now.Add(time.Minute)
	usedAt := now.Add(-time.Second)
	item := token.Token{
		ID:         "backoff",
		TokenValue: "sk-backoff-token",
		Status:     token.StatusActive,
		Health: token.HealthInfo{
			NextCheckAt: &nextCheckAt,
		},
		Stats: token.TokenStats{
			UpdatedAt: &usedAt,
		},
	}
	if selected, ok := currentQuotaRefreshCandidate([]token.Token{item}, now); ok {
		t.Fatalf("expected backoff token to be skipped, got %#v", selected)
	}
}
