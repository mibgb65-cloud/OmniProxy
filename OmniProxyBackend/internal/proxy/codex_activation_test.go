package proxy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"omniproxy/internal/config"
	"omniproxy/internal/token"
)

func TestCodexUsageNeedsActivation(t *testing.T) {
	now := time.Unix(1000, 0)
	tests := []struct {
		name  string
		usage token.UsageInfo
		want  bool
	}{
		{
			name:  "paid account without windows",
			usage: token.UsageInfo{PlanType: "plus"},
			want:  true,
		},
		{
			name: "paid account without five hour window",
			usage: token.UsageInfo{
				PlanType:         "team",
				SecondaryResetAt: 1200,
			},
			want: true,
		},
		{
			name: "paid account without weekly window",
			usage: token.UsageInfo{
				PlanType:       "pro",
				PrimaryResetAt: 1100,
			},
			want: true,
		},
		{
			name: "paid account with both windows",
			usage: token.UsageInfo{
				PlanType:         "plus",
				PrimaryResetAt:   1100,
				SecondaryResetAt: 1200,
			},
			want: false,
		},
		{
			name: "free account with weekly window",
			usage: token.UsageInfo{
				PlanType:         "free",
				SecondaryResetAt: 1200,
			},
			want: false,
		},
		{
			name:  "free account without weekly window",
			usage: token.UsageInfo{PlanType: "free"},
			want:  true,
		},
		{
			name: "paid account with expired five hour window",
			usage: token.UsageInfo{
				PlanType:         "plus",
				PrimaryResetAt:   999,
				SecondaryResetAt: 1200,
			},
			want: true,
		},
		{
			name: "free account with expired weekly window",
			usage: token.UsageInfo{
				PlanType:         "free",
				SecondaryResetAt: 1000,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CodexUsageNeedsActivationAt(tt.usage, now); got != tt.want {
				t.Fatalf("CodexUsageNeedsActivation(%#v) = %v, want %v", tt.usage, got, tt.want)
			}
		})
	}
}

func TestCodexUsageNeedsActivationDetectsRollingUnusedWindows(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	previous := token.UsageInfo{
		PlanType:                  "plus",
		PrimaryResetAt:            now.Add(4*time.Hour + 59*time.Minute).Unix(),
		SecondaryUsedPercentExact: 8,
		SecondaryResetAt:          now.Add(6 * 24 * time.Hour).Unix(),
		PrimaryActivationPending:  true,
	}
	current := token.UsageInfo{
		PlanType:                  "plus",
		PrimaryResetAt:            now.Add(5 * time.Hour).Unix(),
		SecondaryUsedPercentExact: 8,
		SecondaryResetAt:          previous.SecondaryResetAt,
	}

	primary, secondary := CodexUsageActivationPendingAt(current, previous, now)
	if !primary || secondary {
		t.Fatalf("pending windows = primary:%v secondary:%v", primary, secondary)
	}
}

func TestCodexUsageNeedsActivationKeepsStableUnusedWindowActive(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	resetAt := now.Add(4 * time.Hour).Unix()
	previous := token.UsageInfo{
		PlanType:         "plus",
		PrimaryResetAt:   resetAt,
		SecondaryResetAt: now.Add(6 * 24 * time.Hour).Unix(),
	}
	current := previous

	primary, secondary := CodexUsageActivationPendingAt(current, previous, now)
	if primary || secondary {
		t.Fatalf("pending windows = primary:%v secondary:%v", primary, secondary)
	}
}

func TestCodexUsageNeedsActivationDetectsRollingFreeWeeklyWindow(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	current := token.UsageInfo{
		PlanType:         "free",
		SecondaryResetAt: now.Add(7 * 24 * time.Hour).Unix(),
	}

	primary, secondary := CodexUsageActivationPendingAt(current, token.UsageInfo{}, now)
	if primary || !secondary {
		t.Fatalf("pending windows = primary:%v secondary:%v", primary, secondary)
	}
}

func TestValidatorActivatesCodexUsageWithMinimalResponse(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != http.MethodPost || r.URL.Path != "/backend-api/codex/responses" {
			t.Errorf("unexpected activation request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer access-secret" {
			t.Errorf("Authorization = %q", got)
		}
		if got := r.Header.Get("ChatGPT-Account-Id"); got != "acct-123" {
			t.Errorf("ChatGPT-Account-Id = %q", got)
		}
		if got := r.Header.Get("OpenAI-Beta"); got != "responses=experimental" {
			t.Errorf("OpenAI-Beta = %q", got)
		}
		if got := r.Header.Get("Originator"); got != "codex_cli_rs" {
			t.Errorf("Originator = %q", got)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode activation body: %v", err)
		}
		if payload["model"] != "gpt-5.6-luna" || payload["stream"] != true || payload["store"] != false {
			t.Errorf("unexpected activation body: %#v", payload)
		}
		input, ok := payload["input"].([]any)
		if !ok || len(input) != 1 {
			t.Errorf("unexpected activation input: %#v", payload["input"])
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\"}}\n\ndata: [DONE]\n\n"))
	}))
	defer server.Close()

	cfg := config.Default()
	cfg.CodexBaseURL = server.URL + "/backend-api/codex"
	validator, err := NewValidator(cfg)
	if err != nil {
		t.Fatalf("NewValidator: %v", err)
	}
	selected := token.Token{
		Provider:       token.ProviderOpenAI,
		CredentialType: token.CredentialTypeCodexAuthJSON,
		TokenValue:     `{"tokens":{"access_token":"access-secret","account_id":"acct-123"}}`,
	}

	if err := validator.ActivateCodexUsage(context.Background(), selected); err != nil {
		t.Fatalf("ActivateCodexUsage: %v", err)
	}
	if calls != 1 {
		t.Fatalf("activation calls = %d", calls)
	}
}
