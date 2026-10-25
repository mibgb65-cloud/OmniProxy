package gateway

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	routes "omniproxy/internal/application/routing"
)

func TestGatewayUsesModelProviderPreferenceWithoutRestart(t *testing.T) {
	handler := func(name string) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			var request struct{ Model string }
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Error(err)
			}
			w.Header().Set("Content-Type", "application/json")
			if _, err := io.WriteString(w, `{"model":"`+request.Model+`","choices":[{"message":{"content":"`+name+`"}}]}`); err != nil {
				t.Error(err)
			}
		}
	}
	service, db, store := setupService(t, handler("primary"))
	backup := httptest.NewServer(handler("preferred"))
	defer backup.Close()
	second := httptest.NewServer(handler("second"))
	defer second.Close()
	for _, name := range []string{"003_model_routes.sql", "012_model_provider_preferences.sql", "015_model_provider_order.sql", "026_mimo_settings.sql", "027_stepfun_settings.sql"} {
		sql, err := os.ReadFile("../../../migrations/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(string(sql)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`CREATE TABLE account_models(account_id TEXT,model TEXT,name TEXT,fetched_at TEXT,account_version TEXT,metadata_json TEXT NOT NULL DEFAULT '{}');
INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at) VALUES('acct_backup','backup','Backup','test-ref',1,'2026-02-01T00:00:00Z','now');`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO providers VALUES('backup','deepseek','custom','Backup',?,1,'2026-02-01T00:00:00Z','now','inherit')`, backup.URL); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO providers VALUES('second','deepseek','custom','Second',?,1,'2026-03-01T00:00:00Z','now','inherit');
INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at) VALUES('acct_second','second','Second','test-ref',1,'2026-03-01T00:00:00Z','now');`, second.URL); err != nil {
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
	request := func(expected string) {
		t.Helper()
		response, err := http.Post(service.Info().Runtime.Address+"/v1/chat/completions", "application/json", strings.NewReader(`{"model":"deepseek-flash","messages":[{"role":"user","content":"hello"}]}`))
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil || response.StatusCode != 200 || !strings.Contains(string(body), `"content":"`+expected+`"`) {
			t.Fatalf("request did not use %s: %d %s %v", expected, response.StatusCode, body, err)
		}
	}
	request("primary")
	if err := router.SetPreferredProvider(context.Background(), "deepseek-flash", "backup"); err != nil {
		t.Fatal(err)
	}
	request("preferred")
	if _, err := db.Exec(`UPDATE providers SET enabled=0 WHERE id='backup'`); err != nil {
		t.Fatal(err)
	}
	request("primary")
	if _, err := db.Exec(`UPDATE providers SET enabled=1 WHERE id='backup'`); err != nil {
		t.Fatal(err)
	}
	request("preferred")
	if err := router.SetProviderOrder(context.Background(), "deepseek-flash", []string{"backup", "second", "builtin_deepseek"}); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE providers SET enabled=0 WHERE id='backup'`); err != nil {
		t.Fatal(err)
	}
	request("second")
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id='acct_second'`); err != nil {
		t.Fatal(err)
	}
	request("primary")
	if _, err := db.Exec(`UPDATE providers SET enabled=1 WHERE id='backup'`); err != nil {
		t.Fatal(err)
	}
	request("preferred")
}
