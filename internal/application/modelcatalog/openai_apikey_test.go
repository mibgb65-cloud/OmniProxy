package modelcatalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/infrastructure/database"
)

type apiKeyStore struct{}

func (apiKeyStore) Put([]byte) (string, error) { return "ref-api-key", nil }
func (apiKeyStore) Delete(string) error        { return nil }
func (apiKeyStore) Get(string) ([]byte, error) { return []byte("sk-test"), nil }

func TestSyncOpenAIAPIKeyDiscoversModels(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "openai-api-key.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method)
		VALUES('a','builtin_openai','Key','ref-api-key',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z','api_key')`); err != nil {
		t.Fatal(err)
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer sk-test" {
			t.Fatalf("unexpected request: %s %q", r.URL.Path, r.Header.Get("Authorization"))
		}
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-5.3-codex"},{"id":"text-embedding-3-large"}]}`))
	}))
	defer upstream.Close()
	if _, err := db.Exec(`UPDATE providers SET base_url=? WHERE id='builtin_openai'`, upstream.URL+"/v1"); err != nil {
		t.Fatal(err)
	}

	s := New(db, account.NewService(db, apiKeyStore{}), provider.NewService(db), nil)
	items, err := s.Sync(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Model != "gpt-5.3-codex" {
		t.Fatalf("unexpected catalog: %+v", items)
	}
}
