package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	routes "omniproxy/internal/application/routing"
)

func TestLocalGatewayAliasRewriteAndDynamicModels(t *testing.T) {
	var seen string
	service, db, store := setupService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model  string
			Stream bool
			Extra  struct{ Model string }
		}
		if json.NewDecoder(r.Body).Decode(&body) != nil {
			t.Error("invalid upstream JSON")
		}
		seen = body.Model
		if body.Extra.Model != "nested" {
			t.Error("opaque parameter changed")
		}
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"content":"OK"}}]}`)
	}))
	sql, err := os.ReadFile("../../../migrations/003_model_routes.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(sql)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE account_models(account_id TEXT,model TEXT,name TEXT,fetched_at TEXT,account_version TEXT,metadata_json TEXT NOT NULL DEFAULT '{}');`); err != nil {
		t.Fatal(err)
	}
	preferences, err := os.ReadFile("../../../migrations/012_model_provider_preferences.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(preferences)); err != nil {
		t.Fatal(err)
	}
	order, err := os.ReadFile("../../../migrations/015_model_provider_order.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(order)); err != nil {
		t.Fatal(err)
	}
	router := routes.New(db, provider.NewService(db), account.NewService(db, store))
	if _, err := service.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	service.SetRouter(router)
	if _, err := service.Start(); err != nil {
		t.Fatal(err)
	}
	rule, err := router.Create(context.Background(), routes.Input{Alias: "coding-fast", ProviderID: "builtin_deepseek", Model: "deepseek-v4-pro", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.Post(service.Info().Runtime.Address+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"coding-fast","messages":[{"role":"user","content":"hello"}],"extra":{"model":"nested"},"providerId":"attacker"}`))
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != 200 || seen != "deepseek-v4-pro" {
		t.Fatalf("alias not forwarded: %d %s %v", response.StatusCode, body, err)
	}
	response, err = http.Get(service.Info().Runtime.Address + "/v1/models")
	if err != nil {
		t.Fatal(err)
	}
	body, err = io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || !strings.Contains(string(body), "coding-fast") || !strings.Contains(string(body), "deepseek-flash") {
		t.Fatal("model discovery not synchronized")
	}
	if _, err := router.Update(context.Background(), rule.ID, routes.Input{Alias: "coding-fast", ProviderID: "builtin_deepseek", Model: "deepseek-flash", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	response, err = http.Post(service.Info().Runtime.Address+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"coding-fast","messages":[{"role":"user","content":"hello"}],"extra":{"model":"nested"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.ReadAll(response.Body); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 200 || seen != "deepseek-flash" {
		t.Fatal("saved route change required a gateway restart")
	}
	if _, err := router.SetEnabled(context.Background(), rule.ID, false); err != nil {
		t.Fatal(err)
	}
	response, err = http.Post(service.Info().Runtime.Address+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"coding-fast","messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 400 {
		t.Fatal("disabled alias remained supported")
	}
}
