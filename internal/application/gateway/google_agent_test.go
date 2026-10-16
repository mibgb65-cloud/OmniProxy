package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	routes "omniproxy/internal/application/routing"
	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/antigravity"
)

func TestGoogleAgentModelsCatalogAndToolContinuation(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "google.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at,auth_method,auth_state,auth_account_id) VALUES('google-sub','builtin_google','Google','protected-ref','2026-10-17T00:00:00Z','v1','subscription_login','authenticated','1001')`)
	if err != nil {
		t.Fatal(err)
	}
	models := []string{"gemini-3.8-flash-high", "gemini-3.8-flash-low", "gemini-3.8-flash-medium", "claude-opus-4-6-thinking", "claude-sonnet-4-6", "gpt-oss-120b-medium"}
	for _, model := range models {
		if _, err := db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version) VALUES('google-sub',?,?,'now','protected-ref:1001')`, model, model); err != nil {
			t.Fatal(err)
		}
	}
	providers, accounts := provider.NewService(db), account.NewService(db, &testStore{})
	router := routes.New(db, providers, accounts)
	entries, err := router.Catalog(ctx)
	if err != nil || len(entries) != len(models) {
		t.Fatal("subscription models missing from routing catalog", len(entries), entries, err)
	}
	for _, protocol := range []string{"responses", "messages"} {
		if len(catalog.ForProtocol(entries, protocol)) != len(models) {
			t.Fatal("agent protocol missing from catalog", protocol)
		}
	}
	codex, err := catalog.CodexJSON(entries)
	if err != nil {
		t.Fatal(err)
	}
	claude, err := catalog.ClaudeEntries(catalog.ForProtocol(entries, "messages"))
	if err != nil || len(catalog.DesktopModels(claude, "")) != len(models) {
		t.Fatal("Claude menus incomplete", err)
	}
	for _, model := range models {
		if !strings.Contains(string(codex), model) {
			t.Fatal("Codex menu missing model", model)
		}
	}
	service := New(proxy.Options{}, providers, accounts, nil)
	service.SetRouter(router)
	service.SetGoogleSubscription(&googleAccess{})
	upstreamCalls := 0
	service.antigravity = antigravity.New(googleTransport(func(r *http.Request) (*http.Response, error) {
		upstreamCalls++
		data, _ := io.ReadAll(r.Body)
		if strings.Contains(string(data), "allowedCreditTypes") || strings.Contains(string(data), `"thinking"`) {
			t.Error("unexpected paid or provider-specific option")
		}
		body := `{"response":{"candidates":[{"content":{"parts":[{"functionCall":{"id":"call_demo","name":"weather","args":{"city":"Paris"}},"thoughtSignature":"opaque-test-signature"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":4,"candidatesTokenCount":1,"totalTokenCount":5}}}`
		if strings.Contains(string(data), `"model":"claude-`) {
			body = strings.Replace(body, `"finishReason":"STOP"`, `"finishReason":"OTHER"`, 1)
		}
		if strings.Contains(string(data), "functionResponse") {
			if !strings.Contains(string(data), `"thoughtSignature":"opaque-test-signature"`) {
				t.Error("agent tool signature lost on continuation")
			}
			body = `{"response":{"candidates":[{"content":{"parts":[{"text":"Sunny"}]},"finishReason":"STOP"}]}}`
		}
		contentType := "application/json"
		if r.URL.Path == "/v1internal:streamGenerateContent" {
			contentType, body = "text/event-stream", "data: "+body+"\n\n"
		}
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {contentType}}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	if _, err := service.Start(); err != nil {
		t.Fatal(err)
	}
	defer service.Stop(ctx)
	for _, model := range models {
		for _, protocol := range []string{"chat", "responses", "messages"} {
			for _, stream := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/stream=%t", model, protocol, stream), func(t *testing.T) {
					for _, followup := range []bool{false, true} {
						body, path := googleAgentRequest(model, protocol, stream, followup)
						url := fmt.Sprintf("http://127.0.0.1:%d/v1/%s", service.Info().Runtime.Port, path)
						response, err := http.Post(url, "application/json", strings.NewReader(body))
						if err != nil {
							t.Fatal(err)
						}
						data, err := io.ReadAll(response.Body)
						response.Body.Close()
						if err != nil || response.StatusCode != 200 {
							t.Fatalf("gateway failed: status %d body %s error %v", response.StatusCode, data, err)
						}
						want := "call_demo"
						if followup {
							want = "Sunny"
						}
						if !strings.Contains(string(data), want) || strings.Contains(string(data), "opaque-test-signature") && protocol != "chat" {
							t.Fatal("tool output or signature boundary incorrect")
						}
						if stream && !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
							t.Fatal("missing SSE output")
						}
					}
				})
			}
		}
	}
	if upstreamCalls != len(models)*3*2*2 {
		t.Fatal("unexpected cross-account retries", upstreamCalls)
	}
	// 未支持的图片不能被转换为成功的纯文本请求。
	before := upstreamCalls
	request := proxy.Request{Protocol: "responses", Model: models[0], Body: []byte(`{"input":[{"role":"user","content":[{"type":"input_image","image_url":"file:///secret"}]}]}`)}
	if service.forwardGoogleAgent(ctx, antigravity.Credentials{}, request, func(proxy.Response) error { t.Fatal("image reached consumer"); return nil }) == nil || upstreamCalls != before {
		t.Fatal("unsupported input reached upstream")
	}
}

func googleAgentRequest(model, protocol string, stream, followup bool) (string, string) {
	var body, path string
	switch protocol {
	case "responses":
		path = "responses"
		body = `{"input":[{"role":"user","content":"Weather?"}],"tools":[{"type":"function","name":"weather","parameters":{"type":"object"}}],"tool_choice":{"type":"function","name":"weather"}}`
		if followup {
			body = `{"input":[{"role":"user","content":"Weather?"},{"type":"function_call","call_id":"call_demo","name":"weather","arguments":"{\"city\":\"Paris\"}"},{"type":"function_call_output","call_id":"call_demo","output":"Sunny"}]}`
		}
	case "messages":
		path = "messages"
		body = `{"messages":[{"role":"user","content":"Weather?"}],"tools":[{"name":"weather","input_schema":{"type":"object"}}],"tool_choice":{"type":"tool","name":"weather"},"max_tokens":64}`
		if followup {
			body = `{"messages":[{"role":"user","content":"Weather?"},{"role":"assistant","content":[{"type":"tool_use","id":"call_demo","name":"weather","input":{"city":"Paris"}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_demo","content":"Sunny"}]}],"max_tokens":64}`
		}
	default:
		path = "chat/completions"
		body = `{"messages":[{"role":"user","content":"Weather?"}],"tools":[{"type":"function","function":{"name":"weather","parameters":{"type":"object"}}}]}`
		if followup {
			body = `{"messages":[{"role":"user","content":"Weather?"},{"role":"assistant","tool_calls":[{"id":"call_demo","type":"function","function":{"name":"weather","arguments":"{\"city\":\"Paris\"}"}}]},{"role":"tool","tool_call_id":"call_demo","content":"Sunny"}]}`
		}
	}
	var fields map[string]any
	json.Unmarshal([]byte(body), &fields)
	fields["model"], fields["stream"] = model, stream
	encoded, _ := json.Marshal(fields)
	return string(encoded), path
}
