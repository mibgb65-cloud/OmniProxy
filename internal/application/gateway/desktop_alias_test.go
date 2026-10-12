package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	routes "omniproxy/internal/application/routing"
	catalog "omniproxy/internal/core/modelcatalog"
	"omniproxy/internal/core/proxy"
)

func TestDesktopAliasMessagesReachActualModel(t *testing.T) {
	seen := []string{}
	service, db, _ := setupService(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct{ Model string }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		seen = append(seen, body.Model)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"choices":[{"message":{"content":"OK"},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1}}`)
	}))
	request := func(model string, expected int) {
		t.Helper()
		body, _ := json.Marshal(map[string]any{"model": model, "max_tokens": 1, "messages": []any{map[string]any{"role": "user", "content": "hello"}}})
		response, err := http.Post(service.Info().Runtime.Address+"/v1/messages", "application/json", strings.NewReader(string(body)))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		if _, err := io.Copy(io.Discard, response.Body); err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != expected {
			t.Fatalf("status=%d, expected=%d", response.StatusCode, expected)
		}
	}
	request("deepseek-flash", 200)
	request(catalog.DesktopAlias("deepseek-flash"), 200)
	request(catalog.DesktopAlias("missing-model"), 400)
	if _, err := db.Exec(`UPDATE accounts SET enabled=0`); err != nil {
		t.Fatal(err)
	}
	request(catalog.DesktopAlias("deepseek-flash"), 503)
	if len(seen) != 2 || seen[0] != "deepseek-flash" || seen[1] != "deepseek-flash" {
		t.Fatal("compatibility route changed the upstream model or bypassed availability")
	}
}

type desktopAliasRouter struct {
	entries []catalog.Entry
	input   string
}

func (r *desktopAliasRouter) Catalog(context.Context) ([]catalog.Entry, error) { return r.entries, nil }
func (r *desktopAliasRouter) Aliases(context.Context) ([]string, error)        { return nil, nil }
func (r *desktopAliasRouter) Resolve(_ context.Context, model string) (routes.Preview, error) {
	r.input = model
	return routes.Preview{Matched: true, Routable: true, Model: "deepseek-flash", ProviderID: "builtin_deepseek", AccountID: "acct_first"}, nil
}

func TestDesktopAliasPreservesUserRouteAndOpaqueFields(t *testing.T) {
	router := &desktopAliasRouter{entries: []catalog.Entry{{ID: "coding-route", Alias: true, TargetModel: "deepseek-flash"}}}
	service := &Service{router: router}
	alias := catalog.DesktopAlias("coding-route")
	body, _ := json.Marshal(map[string]any{"model": alias, "system": "keep", "nested": map[string]string{"model": "leave"}})
	request, err := service.RouteRequest(context.Background(), proxy.Request{Protocol: "messages", Model: alias, Body: body})
	if err != nil || router.input != "coding-route" || request.Model != "deepseek-flash" || request.ProviderID != "builtin_deepseek" || request.AccountID != "acct_first" {
		t.Fatal("desktop route bypassed the configured user route", err)
	}
	var result struct {
		Model, System string
		Nested        struct{ Model string }
	}
	if json.Unmarshal(request.Body, &result) != nil || result.Model != "deepseek-flash" || result.System != "keep" || result.Nested.Model != "leave" {
		t.Fatal("model routing changed opaque request fields")
	}
	router.entries = nil
	if _, err := service.RouteRequest(context.Background(), proxy.Request{Protocol: "messages", Model: alias, Body: body}); err == nil {
		t.Fatal("removed user route remained available")
	}
}
