package gateway

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/application/provider"
	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
)

// TestOxionAIConvertsResponsesAndMessagesToChat 断言 Oxion AI 只有 Chat Completions：
// Responses／Messages 请求在网关内转换成 Chat 再转发，上游永远看到 /v1/chat/completions，
// 且不乱发官方未声明的参数（stream_options／reasoning_effort）。
func TestOxionAIConvertsResponsesAndMessagesToChat(t *testing.T) {
	for _, protocol := range []string{"chat", "responses", "messages"} {
		t.Run(protocol, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/chat/completions" {
					t.Errorf("unexpected upstream path %q", r.URL.Path)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer placeholder-gateway-test-key" {
					t.Errorf("Authorization = %q", got)
				}
				body, _ := io.ReadAll(r.Body)
				if !strings.Contains(string(body), `"model":"gemma-2-2b"`) {
					t.Errorf("model not rewritten: %s", body)
				}
				if !strings.Contains(string(body), `"messages"`) {
					t.Errorf("messages missing: %s", body)
				}
				for _, forbidden := range []string{"stream_options", "reasoning_effort"} {
					if strings.Contains(string(body), forbidden) {
						t.Errorf("%s must not be sent to Oxion AI: %s", forbidden, body)
					}
				}
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{"id":"chatcmpl-1","choices":[{"index":0,"message":{"role":"assistant","content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":12,"completion_tokens":4,"total_tokens":16}}`)
			}))
			defer upstream.Close()

			db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "gateway.db")})
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`UPDATE providers SET base_url=? WHERE id='builtin_oxionai'`, upstream.URL+"/v1"); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES('oxionai-account','builtin_oxionai','account','test-ref','2026-10-24T00:00:00Z','2026-10-24T00:00:00Z')`); err != nil {
				t.Fatal(err)
			}
			// 目录按账号发放：先写入官方 /models 的同步结果，模型才可路由。
			if _, err := db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version,metadata_json)
				SELECT 'oxionai-account','gemma-2-2b','Gemma 2 2B','now',v.version,'{"contextWindow":128000}'
				FROM account_catalog_versions v WHERE v.id='oxionai-account'`); err != nil {
				t.Fatal(err)
			}

			store := &testStore{}
			providers, accounts := provider.NewService(db), account.NewService(db, store)
			observer := activity.New(db)
			service := New(proxy.Options{Observer: observer}, providers, accounts, nil)
			router := routes.New(db, providers, accounts)
			if _, err := router.Create(context.Background(), routes.Input{Alias: "oxion-chat", ProviderID: "builtin_oxionai", Model: "gemma-2-2b", Enabled: true}); err != nil {
				t.Fatal(err)
			}
			service.SetRouter(router)
			if _, err := service.Start(); err != nil {
				t.Fatal(err)
			}
			defer service.Stop(context.Background())

			path, body := oxionInput(protocol)
			response, err := http.Post(service.Info().Runtime.Address+"/v1/"+path, "application/json", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			data, err := io.ReadAll(response.Body)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != http.StatusOK || !strings.Contains(string(data), "OK") {
				t.Fatalf("gateway rejected request: %d %s", response.StatusCode, data)
			}
			item, err := observer.Request(context.Background(), response.Header.Get("X-Request-ID"))
			if err != nil || item.Model != "gemma-2-2b" || item.State != "success" {
				t.Fatalf("request not recorded: %+v %v", item, err)
			}
			// Chat 原生透传可以读到 usage；Responses／Messages 由网关转换后回包，
			// 与 DeepSeek 转换路径一致，Token 用量保持未知而不是猜一个值。
			if protocol == "chat" {
				if item.InputTokens != "12" || item.OutputTokens != "4" {
					t.Fatalf("native usage lost: %+v", item)
				}
			} else if item.InputTokens != "0" || item.InputKnown {
				t.Fatalf("converted usage must stay unknown: %+v", item)
			}
			for _, value := range store.last {
				if value != 0 {
					t.Fatal("credential bytes not cleared")
				}
			}
		})
	}
}

func oxionInput(protocol string) (string, string) {
	switch protocol {
	case "responses":
		return "responses", `{"model":"oxion-chat","stream":false,"input":"hello"}`
	case "messages":
		return "messages", `{"model":"oxion-chat","stream":false,"max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`
	default:
		return "chat/completions", `{"model":"oxion-chat","stream":false,"messages":[{"role":"user","content":"hello"}]}`
	}
}
