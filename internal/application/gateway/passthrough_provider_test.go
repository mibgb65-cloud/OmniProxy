package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/application/provider"
	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/river"
)

// TestPassthroughAPIKeyRejectsThirdPartyDefaultBase 断言内置第三方服务商在基址为空时
// 不会被当成 OpenAI 通道：否则 River／DeepSeek 的 Key 会被发到 api.openai.com 而整池 401。
func TestPassthroughAPIKeyRejectsThirdPartyDefaultBase(t *testing.T) {
	s, _, _, _ := failoverFixture(t, roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("准备凭据不应发送上游请求")
		return nil, nil
	}))
	for _, kind := range []string{"river", "deepseek", "kimi"} {
		_, err := s.passthroughAttempt(context.Background(), target{
			accountID: "a", authMethod: account.AuthAPIKey, kind: kind,
		}, "deepseek-flash", "")
		var failure *proxy.Error
		if !errors.As(err, &failure) || failure.Code != "unsupported_provider" {
			t.Fatalf("%s: err = %v, want unsupported_provider", kind, err)
		}
	}
}

// TestCodexEntryRoutesNonOpenAIModelThroughProtocolPath 断言 Codex 选择 River 模型时不再
// 透明转发：网关回到协议入口，把公共模型 ID 与 Codex 档位改写成上游取值后发往 River。
func TestCodexEntryRoutesNonOpenAIModelThroughProtocolPath(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(strconv.FormatBool(stream), func(t *testing.T) {
			testCodexEntryNonOpenAIModel(t, stream)
		})
	}
}

func testCodexEntryNonOpenAIModel(t *testing.T, stream bool) {
	const model = river.DeepSeekFlash1017
	publicModel := river.PublicModel(model)
	var gotPath, gotAuth, gotAccept string
	fields := map[string]json.RawMessage{}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth, gotAccept = r.URL.Path, r.Header.Get("Authorization"), r.Header.Get("Accept")
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &fields); err != nil {
			t.Errorf("upstream body is not JSON: %s", body)
		}
		if !stream {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"resp_river","object":"response","status":"completed","output":[],"usage":{"input_tokens":9,"output_tokens":4}}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		first, last := atriaStream("responses")
		_, _ = io.WriteString(w, first)
		w.(http.Flusher).Flush()
		_, _ = io.WriteString(w, last)
	}))
	defer upstream.Close()

	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "codex-river.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE providers SET base_url=? WHERE id='builtin_river'`, upstream.URL+"/v1"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES('river-account','builtin_river','account','test-ref','2026-10-13T00:00:00Z','2026-10-13T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	providers, accounts := provider.NewService(db), account.NewService(db, &testStore{})
	observer := activity.New(db)
	service := New(proxy.Options{Observer: observer}, providers, accounts, nil)
	router := routes.New(db, providers, accounts)
	if err := router.SetProviderOrder(context.Background(), publicModel, []string{"builtin_river"}); err != nil {
		t.Fatal(err)
	}
	service.SetRouter(router)
	if _, err := service.Start(); err != nil {
		t.Fatal(err)
	}
	defer service.Stop(context.Background())

	body := `{"model":` + strconv.Quote(publicModel) + `,"input":[{"role":"user","content":"Hi"},` +
		`{"type":"function_call","call_id":"call_1","name":"exec_command","arguments":"{}"},` +
		`{"type":"function_call_output","call_id":"call_1","output":"ok"},` +
		`{"type":"function_call_output","name":"codex_apps_open_page","namespace":"agent","output":"{\"page_id\":null}"}],` +
		`"stream":` + strconv.FormatBool(stream) + `,"reasoning":{"effort":"xhigh"},` +
		`"tools":[{"type":"function","name":"exec_command","parameters":{"type":"object","properties":{}}},` +
		`{"type":"namespace","name":"mcp__codex_apps__pets","tools":[{"type":"function","name":"_adopt"}]}]}`
	response, err := http.Post(service.Info().Runtime.Address+"/codex/v1/responses", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.StatusCode, data)
	}
	if stream && !strings.Contains(string(data), "response.completed") {
		t.Fatalf("stream not relayed: %s", data)
	}
	if gotPath != "/v1/responses" || gotAuth != "Bearer placeholder-gateway-test-key" {
		t.Fatalf("upstream path = %q auth = %q", gotPath, gotAuth)
	}
	if stream && gotAccept != "text/event-stream" {
		t.Fatalf("upstream Accept = %q", gotAccept)
	}
	if string(fields["model"]) != strconv.Quote(model) {
		t.Fatalf("model not rewritten: %s", fields["model"])
	}
	var reasoning struct {
		Effort string `json:"effort"`
	}
	if json.Unmarshal(fields["reasoning"], &reasoning) != nil || reasoning.Effort != "max" {
		t.Fatalf("codex effort not restored: %s", fields["reasoning"])
	}
	var tools []struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	if json.Unmarshal(fields["tools"], &tools) != nil || len(tools) != 1 || tools[0].Type != "function" || tools[0].Name != "exec_command" {
		t.Fatalf("namespace tool not dropped cleanly: %s", fields["tools"])
	}
	// 桌面端的 Companion Page 快照是没有 call_id 的伪工具结果：留着会让上游在响应头
	// 之后中断连接；合法的一对工具调用与结果必须原样保留。
	var input []struct {
		Type   string `json:"type"`
		CallID string `json:"call_id"`
		Output string `json:"output"`
	}
	if json.Unmarshal(fields["input"], &input) != nil || len(input) != 3 {
		t.Fatalf("orphan tool output not dropped cleanly: %s", fields["input"])
	}
	outputs := 0
	for _, item := range input {
		if item.Type != "function_call_output" {
			continue
		}
		outputs++
		if item.CallID != "call_1" || item.Output != "ok" {
			t.Fatalf("legitimate tool output changed: %+v", item)
		}
	}
	if outputs != 1 || strings.Contains(string(fields["input"]), "codex_apps_open_page") {
		t.Fatalf("orphan tool output survived adaptation: %s", fields["input"])
	}
	item, err := observer.Request(context.Background(), response.Header.Get("X-Request-ID"))
	wantInput := "9"
	if stream {
		wantInput = "12"
	}
	if err != nil || item.Model != model || item.InputTokens != wantInput || item.OutputTokens != "4" {
		t.Fatalf("request record = %+v, %v", item, err)
	}
}
