package modelcatalog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/infrastructure/database"
)

type googleStore struct{}

func (googleStore) Get(string) ([]byte, error) { return []byte("not-a-real-google-key"), nil }
func (googleStore) Put([]byte) (string, error) { return "protected-ref", nil }
func (googleStore) Delete(string) error        { return nil }

func TestGoogleSnapshotSyncFailureAndRotation(t *testing.T) {
	ctx := context.Background()
	fail := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if fail {
			w.WriteHeader(403)
			fmt.Fprint(w, `{"error":{"message":"private-key-detail"}}`)
			return
		}
		if r.URL.Path == "/openai/models" {
			fmt.Fprint(w, `{"data":[{"id":"models/gemini-test"}]}`)
			return
		}
		fmt.Fprint(w, `{"models":[{"name":"models/gemini-test","displayName":"Gemini Test","inputTokenLimit":1048576,"outputTokenLimit":65536,"supportedGenerationMethods":["generateContent"]},{"name":"models/other-native-only","supportedGenerationMethods":["generateContent"]}]}`)
	}))
	defer upstream.Close()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "models.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE providers SET base_url=? WHERE id='builtin_google'`, upstream.URL); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES('a','builtin_google','A','protected-ref','now','now')`); err != nil {
		t.Fatal(err)
	}
	s := New(db, account.NewService(db, googleStore{}), provider.NewService(db), nil)
	items, err := s.Sync(ctx, "a")
	if err != nil || len(items) != 1 {
		t.Fatal("Google sync failed", err)
	}
	if items[0].Model != "gemini-test" || items[0].Metadata.ContextWindow != nil || *items[0].Metadata.InputTokenLimit != 1048576 {
		t.Fatal("official metadata missing")
	}
	fail = true
	if _, err := s.Sync(ctx, "a"); err != ErrUnavailable {
		t.Fatal("unsafe discovery failure", err)
	}
	items, err = s.List(ctx)
	if err != nil || len(items) != 1 {
		t.Fatal("last successful snapshot lost")
	}
	if _, err := db.Exec(`UPDATE accounts SET credential_ref='new-ref' WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	items, err = s.List(ctx)
	if err != nil || len(items) != 0 {
		t.Fatal("old-key snapshot retained")
	}
}
