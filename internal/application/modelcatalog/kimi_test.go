package modelcatalog

import (
	"context"
	"io"
	"net/http"
	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/kimi"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type kimiTransport func(*http.Request) (*http.Response, error)

func (f kimiTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestKimiSyncUsesCurrentPlatformAndCannotSaveOldRegionResults(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "kimi.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	accounts := account.NewService(db, googleStore{})
	a, err := accounts.Create(ctx, account.CreateInput{ProviderID: "builtin_kimi", Name: "Kimi", APIKey: "placeholder-kimi-key"})
	if err != nil {
		t.Fatal(err)
	}
	s := New(db, accounts, provider.NewService(db), nil)
	items, err := s.List(ctx)
	if err != nil || len(items) != 4 || items[0].Source != "preset" || items[0].FetchedAt != "" {
		t.Fatal("preset presented as live discovery")
	}
	change := false
	s.kimi = kimi.New(kimiTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://api.moonshot.cn/v1/models" {
			t.Fatal("wrong model discovery platform")
		}
		if change {
			options := *a.Kimi
			options.Region = "global"
			if _, err := accounts.Update(ctx, account.UpdateInput{ID: a.ID, Name: a.Name, Kimi: &options}); err != nil {
				t.Fatal(err)
			}
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"kimi-k3","context_length":1048576}]}`))}, nil
	}))
	items, err = s.Sync(ctx, a.ID)
	if err != nil || len(items) != 1 || items[0].Source == "preset" || items[0].FetchedAt == "" {
		t.Fatal("real account catalog not saved")
	}
	change = true
	if _, err := s.Sync(ctx, a.ID); err == nil {
		t.Fatal("old region result committed")
	}
	items, err = s.List(ctx)
	if err != nil || len(items) != 4 || items[0].Source != "preset" {
		t.Fatal("new region reused old models")
	}
}
