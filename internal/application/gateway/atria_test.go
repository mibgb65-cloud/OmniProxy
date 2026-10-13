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
)

func TestAtriaNativeGatewayStreamingRoutingAndUsage(t *testing.T) {
	for _, protocol := range []string{"chat", "responses", "messages"} {
		for _, stream := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", protocol, stream), func(t *testing.T) {
				release := make(chan struct{})
				upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					body, _ := io.ReadAll(r.Body)
					if !strings.Contains(string(body), `"model":"Atria-Dawn-Preview"`) {
						t.Error("alias not rewritten")
					}
					if stream {
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
				if _, err := db.Exec(`UPDATE providers SET base_url=? WHERE id='builtin_atria'`, upstream.URL+"/v1"); err != nil {
					t.Fatal(err)
				}
				if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES('atria-account','builtin_atria','account','test-ref','2026-10-13T00:00:00Z','2026-10-13T00:00:00Z')`); err != nil {
					t.Fatal(err)
				}
				store := &testStore{}
				providers, accounts := provider.NewService(db), account.NewService(db, store)
				observer := activity.New(db)
				service := New(proxy.Options{Observer: observer}, providers, accounts, nil)
				router := routes.New(db, providers, accounts)
				if _, err := router.Create(context.Background(), routes.Input{Alias: "research", ProviderID: "builtin_atria", Model: "Atria-Dawn-Preview", Enabled: true}); err != nil {
					t.Fatal(err)
				}
				service.SetRouter(router)
				if _, err := service.Start(); err != nil {
					t.Fatal(err)
				}
				defer service.Stop(context.Background())
				path, body := atriaInput(protocol, stream)
				response, err := http.Post(service.Info().Runtime.Address+"/v1/"+path, "application/json", strings.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				if stream {
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
					if _, err := io.Copy(io.Discard, response.Body); err != nil {
						t.Fatal(err)
					}
				}
				if response.StatusCode != 200 {
					t.Fatal("gateway rejected request", response.StatusCode)
				}
				item, err := observer.Request(context.Background(), response.Header.Get("X-Request-ID"))
				if err != nil || item.InputTokens != "12" || item.OutputTokens != "4" || !item.UsageComplete {
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

func atriaInput(protocol string, stream bool) (string, string) {
	path, input := "chat/completions", `"messages":[{"role":"user","content":"hello"}]`
	if protocol == "responses" {
		path, input = "responses", `"input":"hello"`
	}
	if protocol == "messages" {
		path, input = "messages", `"max_tokens":16,"messages":[{"role":"user","content":"hello"}]`
	}
	return path, fmt.Sprintf(`{"model":"research","stream":%t,%s}`, stream, input)
}

func atriaJSON(protocol string) string {
	switch protocol {
	case "chat":
		return `{"choices":[{"message":{"content":"OK"}}],"usage":{"prompt_tokens":12,"completion_tokens":4}}`
	case "responses":
		return `{"output":[{"type":"message","content":[{"type":"output_text","text":"OK"}]}],"usage":{"input_tokens":12,"output_tokens":4}}`
	default:
		return `{"content":[{"type":"text","text":"OK"}],"usage":{"input_tokens":12,"output_tokens":4}}`
	}
}

func atriaStream(protocol string) (string, string) {
	switch protocol {
	case "chat":
		return "data: {\"choices\":[{\"delta\":{\"content\":\"OK\"}}]}\n\n", "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":4}}\n\ndata: [DONE]\n\n"
	case "responses":
		return "data: {\"type\":\"response.output_text.delta\",\"delta\":\"OK\"}\n\n", "data: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":12,\"output_tokens\":4}}}\n\n"
	default:
		return "data: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":12,\"output_tokens\":0}}}\n\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"OK\"}}\n\n", "data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":4}}\n\ndata: {\"type\":\"message_stop\"}\n\n"
	}
}
