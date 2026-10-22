package gateway

import (
	"bufio"
	"context"
	"fmt"
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
	"omniproxy/internal/provider/river"
)

func TestRiverNativeGatewayStreamingRoutingAndUsage(t *testing.T) {
	for _, model := range []string{river.DeepSeekFlash, river.DeepSeekFlash1017} {
		t.Run(model, func(t *testing.T) {
			testRiverNativeGatewayStreamingRoutingAndUsage(t, model)
		})
	}
}

func testRiverNativeGatewayStreamingRoutingAndUsage(t *testing.T, model string) {
	publicModel := river.PublicModel(model)
	for _, protocol := range []string{"chat", "responses", "messages"} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", protocol, stream), func(t *testing.T) {
				release := make(chan struct{})
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, _ := io.ReadAll(r.Body)
					if !strings.Contains(string(body), fmt.Sprintf(`"model":%q`, model)) {
						t.Error("alias not rewritten")
					}
					path := "/v1/chat/completions"
					if protocol == "responses" {
						path = "/v1/responses"
					}
					if r.URL.Path != path || r.Header.Get("Authorization") != "Bearer placeholder-gateway-test-key" {
						t.Error("wrong protocol or credential")
					}
					if protocol == "messages" && (!strings.Contains(string(body), `"tool_call_id":"call_previous"`) || strings.Contains(string(body), `"thinking"`)) {
						t.Error("tool history lost or provider-specific thinking leaked")
					}
					if protocol == "messages" && !strings.Contains(string(body), `"reasoning_effort":"max"`) {
						t.Error("native Messages effort was discarded")
					}
					if protocol == "messages" && strings.Contains(string(body), "omniproxy-effort-") {
						t.Error("derived effort model leaked upstream")
					}
					if stream && protocol != "messages" {
						w.Header().Set("Content-Type", "text/event-stream")
						first, last := atriaStream(protocol)
						_, _ = io.WriteString(w, first)
						w.(http.Flusher).Flush()
						select {
						case <-release:
						case <-r.Context().Done():
							return
						}
						_, _ = io.WriteString(w, last)
					} else {
						w.Header().Set("Content-Type", "application/json")
						body := atriaJSON(protocol)
						if protocol == "messages" {
							body = `{"id":"r1","choices":[{"finish_reason":"tool_calls","message":{"content":"OK","tool_calls":[{"id":"call_next","type":"function","function":{"name":"lookup","arguments":"{\"value\":1}"}}]}}],"usage":{"prompt_tokens":12,"completion_tokens":4}}`
						}
						_, _ = io.WriteString(w, body)
					}
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
				if _, err := db.Exec(`UPDATE providers SET base_url=? WHERE id='builtin_river'`, upstream.URL+"/v1"); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES('river-account','builtin_river','account','test-ref','2026-10-13T00:00:00Z','2026-10-13T00:00:00Z')`); err != nil {
					t.Fatal(err)
				}
				store := &testStore{}
				providers, accounts := provider.NewService(db), account.NewService(db, store)
				observer := activity.New(db)
				service := New(proxy.Options{Observer: observer}, providers, accounts, nil)
				router := routes.New(db, providers, accounts)
				if _, err := router.Create(context.Background(), routes.Input{Alias: "research", ProviderID: "builtin_river", Model: model, Enabled: true}); err != nil {
					t.Fatal(err)
				}
				if err := router.SetProviderOrder(context.Background(), publicModel, []string{"builtin_river"}); err != nil {
					t.Fatal(err)
				}
				service.SetRouter(router)
				if _, err := service.Start(); err != nil {
					t.Fatal(err)
				}
				defer service.Stop(context.Background())
				path, body := atriaInput(protocol, stream)
				body = strings.ReplaceAll(body, `"research"`, fmt.Sprintf("%q", publicModel))
				if protocol == "messages" {
					// 客户端只用原生 effort 控件选择强度，模型 ID 保持同一个。
					body = fmt.Sprintf(`{"model":%q,"stream":%t,"max_tokens":16,"output_config":{"effort":"max"},"tools":[{"name":"lookup","input_schema":{"type":"object"}}],"messages":[{"role":"assistant","content":[{"type":"tool_use","id":"call_previous","name":"lookup","input":{"value":0}}]},{"role":"user","content":[{"type":"tool_result","tool_use_id":"call_previous","content":"done"}]}]}`, publicModel, stream)
				}
				response, err := http.Post(service.Info().Runtime.Address+"/v1/"+path, "application/json", strings.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				if stream && protocol != "messages" {
					reader := bufio.NewReader(response.Body)
					first, err := reader.ReadString('\n')
					close(release)
					if err != nil || !strings.HasPrefix(first, "data:") {
						t.Fatal("stream buffered or missing first event", err)
					}
					if _, err := io.Copy(io.Discard, reader); err != nil {
						t.Fatal(err)
					}
				} else {
					data, err := io.ReadAll(response.Body)
					if err != nil {
						t.Fatal(err)
					}
					if protocol == "messages" && (!strings.Contains(string(data), "call_next") || !strings.Contains(string(data), "tool_use")) {
						t.Fatal("converted tool call lost", string(data))
					}
				}
				if response.StatusCode != 200 {
					t.Fatal("gateway rejected request", response.StatusCode)
				}
				item, err := observer.Request(context.Background(), response.Header.Get("X-Request-ID"))
				if err != nil || item.Model != model || item.InputTokens != "12" || item.OutputTokens != "4" || !item.UsageComplete {
					t.Fatal("usage lost", item, err)
				}
				for _, value := range store.last {
					if value != 0 {
						t.Fatal("credential bytes not cleared")
					}
				}
			})
		}
	}
}
