package antigravity

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"
)

func entitlementOf(t *testing.T, r *http.Request) string {
	t.Helper()
	data, _ := io.ReadAll(r.Body)
	var body map[string]any
	if json.Unmarshal(data, &body) != nil {
		t.Fatal("unreadable fetchAvailableModels payload")
	}
	entitlement, _ := body["entitlement"].(map[string]any)
	tier, _ := entitlement["userTier"].(string)
	return tier
}

func TestModelsUsesEntitlementThenFallsBackForADay(t *testing.T) {
	var asked []string
	client := New(transportFunc(func(r *http.Request) (*http.Response, error) {
		tier := entitlementOf(t, r)
		asked = append(asked, tier)
		if tier != "" {
			return reply(http.StatusBadRequest, `{"error":{"message":"tier not allowed"}}`), nil
		}
		return reply(http.StatusOK, `{"models":{"gemini-test":{"displayName":"Gemini Test"},"gemini-off":{"displayName":"Off","disabled":true},"chat_internal":{},"gemini-image":{},"claude-sonnet-4-6":{"displayName":"Sonnet"}},"agentModelSorts":[{"groups":[{"modelIds":["gemini-test","claude-sonnet-4-6"]}]}]}`), nil
	}))
	cred := testCredentials()
	cred.Tier = "g1-pro-tier"
	models, err := client.Models(context.Background(), cred)
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 2 || models[0].ID != "claude-sonnet-4-6" || models[1].ID != "gemini-test" {
		t.Fatalf("catalog filtering changed: %+v", models)
	}
	if strings.Join(asked, ",") != "g1-pro-tier," {
		t.Fatalf("entitlement fallback order: %v", asked)
	}
	// 同一账号再次列模型：被拒的权益 24 小时内不再重复尝试。
	asked = nil
	if _, err := client.Models(context.Background(), cred); err != nil {
		t.Fatal("second catalog:", err)
	}
	if strings.Join(asked, ",") != "" {
		t.Fatalf("refused tier was tried again: %q", asked)
	}
}

func TestProjectOnboardsWithClientHeadersAndLiveVersion(t *testing.T) {
	var onboard *http.Request
	var payload map[string]any
	client := New(transportFunc(func(r *http.Request) (*http.Response, error) {
		switch {
		case strings.HasSuffix(r.URL.Path, ":loadCodeAssist"):
			return reply(http.StatusOK, `{"currentTier":{"id":"free-tier"},"paidTier":{"id":"g1-ultra-tier"},"allowedTiers":[{"id":"free-tier","isDefault":true}]}`), nil
		case strings.HasSuffix(r.URL.Path, ":onboardUser"):
			onboard = r
			data, _ := io.ReadAll(r.Body)
			json.Unmarshal(data, &payload)
			return reply(http.StatusOK, `{"done":true,"response":{"cloudaicompanionProject":{"id":"proj-123"}}}`), nil
		}
		return reply(http.StatusInternalServerError, `{}`), nil
	}))
	cred := testCredentials()
	cred.Project, cred.Tier = "", ""
	if err := client.Project(context.Background(), &cred); err != nil {
		t.Fatal(err)
	}
	if cred.Project != "proj-123" || cred.Tier != "g1-ultra-tier" || cred.Plan != "g1-ultra-tier" {
		t.Fatalf("project or tier lost: %+v", cred)
	}
	if onboard == nil {
		t.Fatal("account was not onboarded")
	}
	if onboard.URL.Host != "daily-cloudcode-pa.googleapis.com" {
		t.Fatalf("onboarding endpoint changed: %s", onboard.URL.Host)
	}
	if onboard.Header.Get("X-Goog-Api-Client") != "gl-node/22.21.1" || !strings.Contains(onboard.Header.Get("User-Agent"), "google-api-nodejs-client/10.3.0") {
		t.Fatal("client identity headers missing on onboarding")
	}
	metadata, _ := payload["metadata"].(map[string]any)
	if payload["tier_id"] != "free-tier" || metadata["ide_version"] != protocolVersion() || metadata["ide_name"] != "antigravity" {
		t.Fatalf("onboarding payload changed: %v", payload)
	}
}

func TestProtocolVersionParsingAndUserAgentShape(t *testing.T) {
	for _, test := range []struct {
		manifest string
		want     string
	}{
		{"files: []\nversion: 2.9.1\npath: x", "2.9.1"},
		{"version: '2.10.0'\n", "2.10.0"},
		{"version: latest\n", ""},
		{"no version here\n", ""},
	} {
		if got := parseManifestVersion(test.manifest); got != test.want {
			t.Fatalf("parseManifestVersion(%q) = %q, want %q", test.manifest, got, test.want)
		}
	}
	agent := userAgent()
	if !strings.HasPrefix(agent, "antigravity/hub/") || !strings.Contains(agent, runtime.GOOS+"/"+runtime.GOARCH) || !strings.HasSuffix(agent, " OmniProxy") {
		t.Fatalf("unexpected user agent: %s", agent)
	}
}
