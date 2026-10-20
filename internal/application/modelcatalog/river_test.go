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
	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/river"
)

type riverStore struct {
	last  []byte
	reads int
}

func (s *riverStore) Get(string) ([]byte, error) {
	s.reads++
	s.last = []byte("not-a-real-river-key")
	return s.last, nil
}
func (*riverStore) Put([]byte) (string, error) { return "protected-ref", nil }
func (*riverStore) Delete(string) error        { return nil }

func TestRiverPresetDiscoveryFailureAndCredentialRotation(t *testing.T) {
	ctx := context.Background()
	fail := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" || r.Header.Get("Authorization") != "Bearer not-a-real-river-key" {
			t.Error("unsafe discovery contract")
		}
		if fail {
			w.WriteHeader(403)
			fmt.Fprint(w, `{"error":{"message":"private-key-detail"}}`)
			return
		}
		fmt.Fprint(w, `{"data":[{"id":"deepseek-ai/DeepSeek-V4.1-Flash"},{"id":"deepseek-ai/DeepSeek-V4.1-Flash-1017"},{"id":"unsupported-model"}]}`)
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
	if _, err := db.Exec(`UPDATE providers SET base_url=? WHERE id='builtin_river'`, upstream.URL+"/v1"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,created_at,updated_at) VALUES('a','builtin_river','River','protected-ref','2026-10-17T00:00:00Z','2026-10-17T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	store := &riverStore{}
	s := New(db, account.NewService(db, store), provider.NewService(db), nil)
	items, err := s.List(ctx)
	if err != nil || len(items) != len(river.Models()) || store.reads != 0 {
		t.Fatal("preset reads keys or lost models", err)
	}
	for _, item := range items {
		if item.Source != "preset" || item.FetchedAt != "" {
			t.Fatal("preset pretends to be discovery")
		}
	}
	items, err = s.Sync(ctx, "a")
	if err != nil || len(items) != 2 || items[0].Model != "deepseek-flash" || items[0].Name != "DeepSeek-V4.1-Flash" || items[0].Source == "preset" || items[0].FetchedAt == "" {
		t.Fatal("discovered models not authoritative", items, err)
	}
	if items[1].Model != "DeepSeek-V4.1-Flash-1017" || items[1].Name != items[1].Model || items[1].Source == "preset" || items[1].FetchedAt == "" {
		t.Fatal("versioned discovered model was merged or hidden", items[1])
	}
	router := routes.New(db, s.providers, s.accounts)
	entries, err := router.Catalog(ctx)
	if err != nil || len(entries) != 2 || entries[0].ID == entries[1].ID {
		t.Fatal("routing disagrees with discovered account catalog", entries, err)
	}
	for _, entry := range entries {
		if entry.TargetModel != river.UpstreamModel(entry.ID) || entry.Name != river.ModelName(entry.ID) {
			t.Fatal("persisted discovery lost model identity", entry)
		}
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
	if err != nil || len(items) != 2 {
		t.Fatal("last good discovery snapshot lost")
	}
	if _, err := db.Exec(`UPDATE accounts SET credential_ref='rotated-ref' WHERE id='a'`); err != nil {
		t.Fatal(err)
	}
	items, err = s.List(ctx)
	if err != nil || len(items) != len(river.Models()) {
		t.Fatal("new key preset missing", err)
	}
	for _, item := range items {
		if item.FetchedAt != "" || item.Source != "preset" {
			t.Fatal("old key discovery retained")
		}
	}
}
