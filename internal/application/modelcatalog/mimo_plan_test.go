package modelcatalog

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/mimo"
)

type mimoPlanTransport func(*http.Request) (*http.Response, error)

func (f mimoPlanTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type mimoPlanStore struct{}

func (*mimoPlanStore) Put([]byte) (string, error) { return "protected-plan-ref", nil }
func (*mimoPlanStore) Get(string) ([]byte, error) { return []byte("tp-placeholder-plan-key"), nil }
func (*mimoPlanStore) Delete(string) error        { return nil }

func TestMiMoPlanCatalogDiscoveryAndConcurrentClusterChange(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "plan.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	accounts := account.NewService(db, &mimoPlanStore{})
	a, err := accounts.Create(ctx, account.CreateInput{ProviderID: "builtin_mimo", Name: "Plan", APIKey: "tp-placeholder-plan-key", MiMo: &mimo.Options{Region: "ams", BillingMode: "token_plan"}})
	if err != nil {
		t.Fatal(err)
	}
	s := New(db, accounts, provider.NewService(db), nil)
	items, err := s.List(ctx)
	if err != nil || len(items) != 2 || items[0].Source != "preset" || items[0].FetchedAt != "" {
		t.Fatal("plan presets invented discovery or models", items, err)
	}
	change, fail := false, false
	s.mimo = mimo.New(mimoPlanTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://token-plan-ams.xiaomimimo.com/v1/models" || r.Header.Get("Api-Key") != "tp-placeholder-plan-key" {
			t.Fatal("wrong plan discovery endpoint")
		}
		if change {
			o := *a.MiMo
			o.Region = "cn"
			if _, err := accounts.Update(ctx, account.UpdateInput{ID: a.ID, Name: a.Name, MiMo: &o}); err != nil {
				t.Fatal(err)
			}
		}
		status, body := 200, `{"data":[{"id":"mimo-v2.6-flash"},{"id":"mimo-v2.6-pro-ultraspeed"}]}`
		if fail {
			status = 403
			body = `private-failure`
		}
		return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(body))}, nil
	}))
	items, err = s.Sync(ctx, a.ID)
	if err != nil || len(items) != 1 || items[0].Model != "mimo-v2.6-flash" || items[0].FetchedAt == "" {
		t.Fatal("discovery not authoritative", items, err)
	}
	fail = true
	if _, err := s.Sync(ctx, a.ID); err != ErrUnavailable {
		t.Fatal("discovery failure exposed", err)
	}
	items, err = s.List(ctx)
	if err != nil || len(items) != 1 || items[0].FetchedAt == "" {
		t.Fatal("failure lost last good discovery")
	}
	fail = false
	change = true
	if _, err := s.Sync(ctx, a.ID); err != ErrUnavailable {
		t.Fatal("old cluster results saved after config change")
	}
	items, err = s.List(ctx)
	if err != nil || len(items) != 2 || items[0].Source != "preset" || items[0].FetchedAt != "" {
		t.Fatal("old cluster directory survived change")
	}
}
