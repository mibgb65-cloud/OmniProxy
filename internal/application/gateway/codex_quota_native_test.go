package gateway

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/provider/chatgpt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestCodexNativeQuotaFailoverAndLargeEvent(t *testing.T) {
	executable := os.Getenv("OMNIPROXY_CODEX_TEST_EXECUTABLE")
	if executable == "" {
		t.Skip("set OMNIPROXY_CODEX_TEST_EXECUTABLE for the isolated native client contract")
	}
	if !filepath.IsAbs(executable) {
		t.Fatal("absolute executable required")
	}
	s, _, observer, _ := subscriptionQuotaFixture(t)
	var calls atomic.Int32
	s.chatgpt = chatgpt.New(roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls.Add(1)
		body := `{"error":{"type":"usage_limit_reached","message":"private-quota-marker"}}`
		if strings.HasSuffix(r.Header.Get("Authorization"), "-b") {
			body = "data: {\"type\":\"response.created\",\"response\":{\"id\":\"one\",\"tools\":[{\"description\":\"" + strings.Repeat("x", 600<<10) + "\"}]}}\n\n" + "data: {\"type\":\"response.completed\",\"response\":{\"id\":\"one\",\"object\":\"response\",\"status\":\"completed\",\"output\":[],\"usage\":{\"input_tokens\":12,\"output_tokens\":4,\"total_tokens\":16}}}\n\n"
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	home, work := t.TempDir(), t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	provider := fmt.Sprintf(`model_providers.qa={name="Isolated quota QA",base_url="%s/v1",wire_api="responses",requires_openai_auth=false,env_key="QA_KEY",supports_websockets=true,request_max_retries=0,stream_max_retries=0}`, s.Info().Runtime.Address)
	command := exec.CommandContext(ctx, executable, "exec", "--ignore-user-config", "--ignore-rules", "--ephemeral", "--skip-git-repo-check", "--sandbox", "read-only", "--json", "-C", work, "-c", `model_provider="qa"`, "-m", "gpt-test", "-c", provider, "Reply OK without tools.")
	command.Dir = work
	for _, key := range []string{"PATH", "SystemRoot", "WINDIR", "COMSPEC", "PATHEXT"} {
		if value := os.Getenv(key); value != "" {
			command.Env = append(command.Env, key+"="+value)
		}
	}
	command.Env = append(command.Env, "CODEX_HOME="+home, "HOME="+home, "USERPROFILE="+home, "APPDATA="+home, "LOCALAPPDATA="+home, "TEMP="+home, "TMP="+home, "QA_KEY=synthetic-placeholder", "NO_PROXY=127.0.0.1,localhost", "HTTP_PROXY="+s.Info().Runtime.Address, "HTTPS_PROXY="+s.Info().Runtime.Address, "ALL_PROXY="+s.Info().Runtime.Address)
	output, err := command.CombinedOutput()
	if err != nil || !strings.Contains(string(output), `"type":"turn.completed"`) || calls.Load() != 2 {
		t.Fatalf("native quota contract failed: calls=%d error=%v output=%s", calls.Load(), err, output)
	}
	page, err := observer.Requests(ctx, activity.RequestQuery{Limit: 10})
	if err != nil || len(page.Items) != 1 || page.Items[0].State != "success" {
		t.Fatal("native logical accounting lost", page, err)
	}
	item, err := observer.Request(ctx, page.Items[0].ID)
	if err != nil || item.AttemptCount != 2 || item.AccountID != "b" || item.Attempts[0].StatusCode != 200 || item.Attempts[0].ErrorCode != "upstream_quota_exhausted" {
		t.Fatal("native failover lost wire evidence", item, err)
	}
}
