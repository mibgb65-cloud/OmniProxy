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
	"omniproxy/internal/provider/mimo"
)

type mimoStore struct {
	last  []byte
	reads int
}

func (s *mimoStore) Get(string) ([]byte, error) {
	s.reads++
	s.last = []byte("not-a-real-mimo-key")
	return s.last, nil
}
func (*mimoStore) Put([]byte) (string, error) { return "protected-ref", nil }
func (*mimoStore) Delete(string) error        { return nil }

func TestMiMoPresetDiscoveryFailureAndCredentialRotation(t *testing.T) {
	ctx := context.Background()
	fail := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" || r.Header.Get("Api-Key") != "not-a-real-mimo-key" {
			t.Error("unsafe discovery contract")
		}
		if fail {
			w.WriteHeader(403)
			fmt.Fprint(w, `{"error":{"message":"private-key-detail"}}`)
			return
		}
		fmt.Fprint(w, `{"data":[{"id":"mimo-v2.6-flash"},{"id":"mimo-v2.5-asr"}]}`)
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
	if _, err := db.Exec(`UPDATE providers SET base_url=? WHERE id='builtin_mimo'`, upstream.URL+"/v1"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES('a','builtin_mimo','MiMo','protected-ref','now','now'); INSERT INTO mimo_account_options(account_id,region,billing_mode) VALUES('a','','payg')`); err != nil {
		t.Fatal(err)
	}
	store := &mimoStore{}
	s := New(db, account.NewService(db, store), provider.NewService(db), nil)
	items, err := s.List(ctx)
	if err != nil || len(items) != len(mimo.Models()) || store.reads != 0 {
		t.Fatal("preset reads keys or lost models", err)
	}
	for _, item := range items {
		if item.Source != "preset" || item.FetchedAt != "" {
			t.Fatal("preset pretends to be discovery")
		}
	}
	items, err = s.Sync(ctx, "a")
	if err != nil || len(items) != 1 || items[0].Model != "mimo-v2.6-flash" || items[0].Source == "preset" || items[0].FetchedAt == "" {
		t.Fatal("discovered models not authoritative", items, err)
	}
	for _, value := range store.last {
		if value != 0 {
			t.Fatal("credential was not cleared")
		}
	}
	fail = true
	if _, err := s.Sync(ctx, "a"); err != ErrUnavailable {
		t.Fatal("discovery failure exposed upstream details", err)
	}
	items, err = s.List(ctx)
	if err != nil || len(items) != 1 {
		t.Fatal("last good discovery snapshot lost")
	}
	if _, err := db.Exec(`UPDATE accounts SET credential_ref='rotated-ref' WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	items, err = s.List(ctx)
	if err != nil || len(items) != len(mimo.Models()) {
		t.Fatal("new key preset missing", err)
	}
	for _, item := range items {
		if item.FetchedAt != "" || item.Source != "preset" {
			t.Fatal("old key discovery retained")
		}
	}
}
