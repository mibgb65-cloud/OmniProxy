package gateway

import (
	"context"
	"database/sql"
	"net/http"
	"omniproxy/internal/application/account"
	"omniproxy/internal/application/activity"
	"omniproxy/internal/application/provider"
	routes "omniproxy/internal/application/routing"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/deepseek"
	"os"
	"path/filepath"
	"testing"
)

type failoverStore struct{}
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func (failoverStore) Put([]byte) (string, error)                          { return "unused", nil }
func (failoverStore) Delete(string) error                                 { return nil }
func (failoverStore) Get(ref string) ([]byte, error) {
	return []byte("placeholder-failover-" + ref), nil
}

func failoverFixture(t *testing.T, transport http.RoundTripper) (*Service, *sql.DB, *activity.Service, *routes.Service) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "failover.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err = database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"a", "b", "c", "d"} {
		if _, err = db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at) VALUES(?,'builtin_deepseek',?,?,1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z')`, id, "Account "+id, id); err != nil {
			t.Fatal(err)
		}
	}
	accounts := account.NewService(db, failoverStore{})
	providers := provider.NewService(db)
	observer := activity.New(db)
	router := routes.New(db, providers, accounts)
	s := New(proxy.Options{Observer: observer, FailoverEnabled: NewFailoverSettingsService(db).Enabled}, providers, accounts, deepseek.New(transport))
	s.SetRouter(router)
	if _, err = s.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Stop(context.Background()) })
	return s, db, observer, router
}
