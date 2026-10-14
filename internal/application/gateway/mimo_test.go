package gateway

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/application/provider"
	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
)

func TestMiMoNativeGatewayStreamsRoutingAndUsage(t *testing.T) {
	paths := map[string]string{"chat": "/v1/chat/completions", "responses": "/v1/responses", "messages": "/anthropic/v1/messages"}
	for protocol, path := range paths {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", protocol, stream), func(t *testing.T) {
				release := make(chan struct{})
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var fields map[string]json.RawMessage
					if json.NewDecoder(r.Body).Decode(&fields) != nil || string(fields["model"]) != `"mimo-v2.6-pro"` || r.URL.Path != path {
						t.Error("native endpoint or alias routing incorrect")
					}
					if r.Header.Get("Api-Key") != "placeholder-gateway-test-key" || r.Header.Get("Authorization") != "" {
						t.Error("credential authentication incorrect")
					}
					if stream {
						w.Header().Set("Content-Type", "text/event-stream")
						// 三种标准原生用量事件与 Atria 相同，复用协议夹具。
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
						_, _ = io.WriteString(w, atriaJSON(protocol))
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
				if _, err := db.Exec(`UPDATE providers SET base_url=? WHERE id='builtin_mimo'`, upstream.URL+"/v1"); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES('mimo-account','builtin_mimo','MiMo','test-ref','2026-10-14T00:00:00Z','2026-10-14T00:00:00Z')`); err != nil {
					t.Fatal(err)
				}
				store := &testStore{}
				providers, accounts := provider.NewService(db), account.NewService(db, store)
				observer := activity.New(db)
				service := New(proxy.Options{Observer: observer}, providers, accounts, nil)
				router := routes.New(db, providers, accounts)
				if _, err := router.Create(context.Background(), routes.Input{Alias: "research", ProviderID: "builtin_mimo", Model: "mimo-v2.6-pro", Enabled: true}); err != nil {
					t.Fatal(err)
				}
				service.SetRouter(router)
				if _, err := service.Start(); err != nil {
					t.Fatal(err)
				}
				defer service.Stop(context.Background())
				requestPath, body := atriaInput(protocol, stream)
				// 同时覆盖 Claude 的百万上下文后缀，经目录校验后才允许去后缀。
				if protocol == "messages" {
					body = strings.Replace(body, `"research"`, `"research[1m]"`, 1)
				}
				client := &http.Client{Timeout: 3 * time.Second}
				response, err := client.Post(service.Info().Runtime.Address+"/v1/"+requestPath, "application/json", strings.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				if stream {
					reader := bufio.NewReader(response.Body)
					first, err := reader.ReadString('\n')
					close(release)
					if err != nil || !strings.HasPrefix(first, "data:") {
						t.Fatal("stream buffered or first event missing", err)
					}
					if _, err := io.Copy(io.Discard, reader); err != nil {
						t.Fatal(err)
					}
				} else if _, err := io.Copy(io.Discard, response.Body); err != nil {
					t.Fatal(err)
				}
				if response.StatusCode != 200 {
					t.Fatal("MiMo request rejected", response.StatusCode)
				}
				item, err := observer.Request(context.Background(), response.Header.Get("X-Request-ID"))
				if err != nil || item.InputTokens != "12" || item.OutputTokens != "4" || !item.UsageComplete {
					t.Fatal("native usage lost", item, err)
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
