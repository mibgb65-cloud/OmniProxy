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
	"omniproxy/internal/provider/stepfun"
)

type stepOrderStore struct{}

func (stepOrderStore) Put([]byte) (string, error) { panic("test uses metadata fixture") }
func (stepOrderStore) Delete(string) error        { panic("test must not delete credentials") }
func (stepOrderStore) Get(ref string) ([]byte, error) {
	if ref == "ref-plan" {
		return []byte("placeholder-plan-key"), nil
	}
	return []byte("placeholder-api-key"), nil
}

func TestStepFunGatewayOrderModelDeduplicationAndLegacyCompatibility(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "step-order.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at)VALUES
 ('step-api','builtin_stepfun','API','ref-api',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z'),
 ('step-plan','builtin_stepfun','Plan','ref-plan',1,'2026-01-02T00:00:00Z','2026-01-02T00:00:00Z');
 INSERT INTO stepfun_account_options(account_id,region,billing_mode)VALUES('step-api','global','payg'),('step-plan','cn','step_plan')`); err != nil {
		t.Fatal(err)
	}
	providers, accounts := provider.NewService(db), account.NewService(db, stepOrderStore{})
	router := routes.New(db, providers, accounts)
	s := New(proxy.Options{}, providers, accounts, nil)
	s.SetRouter(router)
	wantMode, protocol, status, calls := "step_plan", "chat", 200, 0
	s.stepfun = stepfun.New(stepTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		host, key, prefix := "api.stepfun.com", "placeholder-plan-key", "/step_plan/v1/"
		if wantMode == "payg" {
			host, key, prefix = "api.stepfun.ai", "placeholder-api-key", "/v1/"
		}
		if r.URL.Host != host || !strings.HasPrefix(r.URL.Path, prefix) || r.Header.Get("Authorization") != "Bearer "+key {
			t.Fatal("wrong billing platform, endpoint or credential")
		}
		var body struct{ Model string }
		if json.NewDecoder(r.Body).Decode(&body) != nil || body.Model != "step-5-preview" {
			t.Fatal("upstream model changed")
		}
		return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(atriaJSON(protocol)))}, nil
	}))
	if _, err := s.Start(); err != nil {
		t.Fatal(err)
	}
	defer s.Stop(ctx)
	for _, mode := range []string{"step_plan", "payg", "step_plan"} {
		wantMode = mode
		if _, err := router.SetStepFunPreferredBillingMode(ctx, mode); err != nil {
			t.Fatal(err)
		}
		for _, p := range []string{"chat", "responses", "messages"} {
			protocol = p
			path, body := atriaInput(p, false)
			body = strings.ReplaceAll(body, `"research"`, `"step-5-preview"`)
			response, err := http.Post(s.Info().Runtime.Address+"/v1/"+path, "application/json", strings.NewReader(body))
			if err != nil {
				t.Fatal(err)
			}
			_, readErr := io.Copy(io.Discard, response.Body)
			closeErr := response.Body.Close()
			if response.StatusCode != 200 || readErr != nil || closeErr != nil {
				t.Fatal("native request failed", response.StatusCode, readErr, closeErr)
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
			t.Fatal("duplicate channel", m.ID)
		}
		seen[m.ID] = true
	}
	if len(seen) != 5 || !seen["step-router-v1"] {
		t.Fatal("supported model lost", seen)
	}
	for _, model := range []string{"stepfun-cn-plan/step-5-preview[1m]", catalog.DesktopAlias("stepfun-cn-plan/step-5-preview[1m]")} {
		r, err := s.RouteRequest(ctx, proxy.Request{Model: model, Protocol: "messages", Body: []byte(`{"model":"` + model + `","messages":[{"role":"user","content":"hello"}]}`)})
		if err != nil || r.AccountID != "step-plan" || r.Model != "step-5-preview" {
			t.Fatal("legacy Claude model lost", r.Model, err)
		}
	}
	status, protocol = 503, "chat"
	response, err = http.Post(s.Info().Runtime.Address+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"step-5-preview","messages":[{"role":"user","content":"hello"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode < 400 || calls != 10 {
		t.Fatal("failed request retried across channels", response.StatusCode, calls)
	}
}
