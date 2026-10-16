package gateway

import (
	"context"
	"encoding/json"
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
	"omniproxy/internal/provider/mimo"
)

type mimoOrderStore struct{}

func (mimoOrderStore) Put([]byte) (string, error) { panic("test uses metadata fixture") }
func (mimoOrderStore) Delete(string) error        { panic("test must not delete credentials") }
func (mimoOrderStore) Get(ref string) ([]byte, error) {
	if ref == "ref-plan" {
		return []byte("tp-placeholder-order-key"), nil
	}
	return []byte("sk-placeholder-order-key"), nil
}

func TestMiMoGatewayBillingOrderUpdatesWithoutRestartAndDeduplicatesModels(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "order.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at)VALUES
 ('order-api','builtin_mimo','API','ref-api',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z'),
 ('order-plan','builtin_mimo','Plan','ref-plan',1,'2026-01-02T00:00:00Z','2026-01-02T00:00:00Z');
 INSERT INTO mimo_account_options(account_id,region,billing_mode)VALUES('order-api','','payg'),('order-plan','sgp','token_plan')`); err != nil {
		t.Fatal(err)
	}
	providers, accounts := provider.NewService(db), account.NewService(db, mimoOrderStore{})
	router := routes.New(db, providers, accounts)
	s := New(proxy.Options{}, providers, accounts, nil)
	s.SetRouter(router)
	wantMode, protocol, upstreamStatus, calls := "token_plan", "chat", 200, 0
	s.mimo = mimo.New(kimiRoundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		host, key := "token-plan-sgp.xiaomimimo.com", "tp-placeholder-order-key"
		if wantMode == "payg" {
			host, key = "api.xiaomimimo.com", "sk-placeholder-order-key"
		}
		if r.URL.Host != host || r.Header.Get("Api-Key") != key {
			t.Fatal("wrong billing host or protected credential")
		}
		return &http.Response{StatusCode: upstreamStatus, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(atriaJSON(protocol)))}, nil
	}))
	if _, err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop(ctx)
	for _, mode := range []string{"token_plan", "payg", "token_plan"} {
		wantMode = mode
		if _, err := router.SetMiMoPreferredBillingMode(ctx, mode); err != nil {
			t.Fatal(err)
		}
		for _, p := range []string{"chat", "responses", "messages"} {
			protocol = p
			path, body := atriaInput(p, false)
			body = strings.ReplaceAll(body, `"research"`, `"mimo-v2.6-pro"`)
			response, err := http.Post(s.Info().Runtime.Address+"/v1/"+path, "application/json", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			_, readErr := io.Copy(io.Discard, response.Body)
			closeErr := response.Body.Close()
			if response.StatusCode != 200 || readErr != nil || closeErr != nil {
				t.Fatal("native billing route failed", response.StatusCode, readErr, closeErr)
			}
		}
	}
	response, err := http.Get(s.Info().Runtime.Address + "/v1/models")
	if err != nil {
		t.Fatal(err)
	}
	var models struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeErr := json.NewDecoder(response.Body).Decode(&models)
	closeErr := response.Body.Close()
	if decodeErr != nil || closeErr != nil {
		t.Fatal(decodeErr, closeErr)
	}
	seen := map[string]bool{}
	for _, m := range models.Data {
		if seen[m.ID] || strings.Contains(m.ID, "/") {
			t.Fatal("duplicate channel in model list", m.ID)
		}
		seen[m.ID] = true
	}
	if len(seen) != 3 {
		t.Fatal("native model list missing", seen)
	}
	for _, oldModel := range []string{"mimo-sgp-plan/mimo-v2.6-pro[1m]", catalog.DesktopAlias("mimo-sgp-plan/mimo-v2.6-pro[1m]")} {
		request, err := s.RouteRequest(ctx, proxy.Request{Model: oldModel, Protocol: "messages", Body: []byte(`{"model":"` + oldModel + `","messages":[{"role":"user","content":"hello"}]}`)})
		if err != nil || request.AccountID != "order-plan" || request.Model != "mimo-v2.6-pro" {
			t.Fatal("legacy Claude channel/context model no longer resolves", request.Model, err)
		}
	}
	upstreamStatus, protocol = 503, "chat"
	response, err = http.Post(s.Info().Runtime.Address+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"mimo-v2.6-pro","messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusBadGateway || calls != 10 {
		t.Fatal("failed request retried through another billing channel", response.StatusCode, calls)
	}
}
