package gateway

import (
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
	"omniproxy/internal/application/googlequota"
	"omniproxy/internal/application/provider"
	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
)

func TestGoogleGatewayRoutingUsageLimitsAndObservedFailure(t *testing.T) {
	for _, stream := range []bool{false, true} {
		t.Run(fmt.Sprint(stream), func(t *testing.T) {
			ctx := context.Background()
			calls := 0
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				body, _ := io.ReadAll(r.Body)
				if !strings.Contains(string(body), `"model":"gemini-test"`) {
					t.Error("alias not rewritten")
				}
				if strings.Contains(string(body), "fail-quota") {
					w.Header().Set("Retry-After", "30")
					w.WriteHeader(429)
					io.WriteString(w, `{"error":{"message":"private upstream text"}}`)
					return
				}
				if stream {
					w.Header().Set("Content-Type", "text/event-stream")
					io.WriteString(w, atriaStreamJSON())
				} else {
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(w, atriaJSON("chat"))
				}
			}))
			defer upstream.Close()
			db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "gateway.db")})
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`UPDATE providers SET base_url=? WHERE id='builtin_google'`, upstream.URL+"/v1beta"); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES('google-a','builtin_google','Google','test-ref','2026-10-14T00:00:00Z','2026-10-14T00:00:00Z')`); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version,metadata_json) VALUES('google-a','gemini-test','Gemini','now','test-ref:', '{"inputTokenLimit":1048576,"outputTokenLimit":32}')`); err != nil {
				t.Fatal(err)
			}
			providers, accounts := provider.NewService(db), account.NewService(db, &testStore{})
			service := New(proxy.Options{Observer: activity.New(db)}, providers, accounts, nil)
			router := routes.New(db, providers, accounts)
			if _, err := router.Create(ctx, routes.Input{Alias: "gemini-alias", ProviderID: "builtin_google", Model: "gemini-test", AccountID: "google-a", Enabled: true}); err != nil {
				t.Fatal(err)
			}
			service.SetRouter(router)
			resources := googlequota.New(db)
			service.SetGoogleResources(resources)
			if _, err := service.Start(); err != nil {
				t.Fatal(err)
			}
			defer service.Stop(ctx)
			post := func(path, body string) (int, string) {
				response, err := http.Post(service.Info().Runtime.Address+path, "application/json", strings.NewReader(body))
				if err != nil {
					t.Fatal(err)
				}
				defer response.Body.Close()
				raw, err := io.ReadAll(response.Body)
				if err != nil {
					t.Fatal(err)
				}
				return response.StatusCode, string(raw)
			}
			status, firstBody := post("/v1/chat/completions", fmt.Sprintf(`{"model":"gemini-alias","stream":%t,"max_tokens":16,"messages":[{"role":"user","content":"hello"}]}`, stream))
			if status != 200 || calls != 1 {
				t.Fatal("Google route failed", status, firstBody)
			}
			var input, output string
			if err := db.QueryRow(`SELECT input_tokens,output_tokens FROM request_records WHERE provider_id='builtin_google' LIMIT 1`).Scan(&input, &output); err != nil || input != "12" || output != "4" {
				t.Fatal("Google usage missing", err)
			}
			status, _ = post("/v1/chat/completions", `{"model":"gemini-alias","max_tokens":33,"messages":[{"role":"user","content":"hello"}]}`)
			if status != 400 || calls != 1 {
				t.Fatal("model output limit not enforced")
			}
			status, _ = post("/v1/responses", `{"model":"gemini-alias","input":"hello"}`)
			if status != 400 || calls != 1 {
				t.Fatal("unverified Agent protocol forwarded")
			}
			status, body := post("/v1/chat/completions", `{"model":"gemini-alias","messages":[{"role":"user","content":"fail-quota"}]}`)
			if status != 429 || strings.Contains(body, "private upstream text") {
				t.Fatal("upstream error leaked")
			}
			items, err := resources.List(ctx)
			if err != nil || len(items) != 1 || len(items[0].Observations) != 1 || items[0].Observations[0].Category != "rate_limited" {
				t.Fatal("quota observation missing", err)
			}
			entries, err := router.Catalog(ctx)
			if err != nil {
				t.Fatal(err)
			}
			for _, entry := range entries {
				if entry.ProviderKind == "google" && (len(entry.Protocols) != 1 || entry.Protocols[0] != "chat_completions") {
					t.Fatal("unsupported capabilities advertised")
				}
			}
		})
	}
}

func atriaStreamJSON() string { first, last := atriaStream("chat"); return first + last }
