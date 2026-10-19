package quotaactivation

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/gateway"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/application/subscription"
	"omniproxy/internal/infrastructure/database"
)

type testAccounts struct {
	item     account.Item
	metadata *account.Item
}

func (a *testAccounts) List(context.Context) ([]account.Item, error) {
	return []account.Item{a.item}, nil
}
func (a *testAccounts) Metadata(context.Context, string) (account.Item, error) {
	if a.metadata != nil {
		return *a.metadata, nil
	}
	return a.item, nil
}

type testProviders struct{ enabled bool }

func (p *testProviders) List(context.Context) ([]provider.Configuration, error) {
	return []provider.Configuration{{ID: "builtin_openai", Kind: "openai", Enabled: p.enabled}}, nil
}

type testQuotas struct {
	mu    sync.Mutex
	quota subscription.Quota
	err   error
	reads atomic.Int32
}

func (q *testQuotas) ReadQuota(context.Context, string) (subscription.Quota, error) {
	q.reads.Add(1)
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.quota, q.err
}
func (q *testQuotas) set(quota subscription.Quota) { q.mu.Lock(); defer q.mu.Unlock(); q.quota = quota }

type sendFunc func(context.Context, account.Item, SendPlan) error

func (f sendFunc) Send(ctx context.Context, item account.Item, plan SendPlan) error {
	return f(ctx, item, plan)
}

type activationFixture struct {
	db        *sql.DB
	service   *Service
	accounts  *testAccounts
	providers *testProviders
	quotas    *testQuotas
	sends     atomic.Int32
}

func unstartedQuota() subscription.Quota {
	return subscription.Quota{FiveHourEnabled: true, FiveHour: &subscription.QuotaWindow{RemainingPercent: 100}, Weekly: &subscription.QuotaWindow{RemainingPercent: 100}}
}
func activeQuota() subscription.Quota {
	return subscription.Quota{FiveHourEnabled: true,
		FiveHour: &subscription.QuotaWindow{RemainingPercent: 99, ResetsAt: time.Now().Add(5 * time.Hour).UTC().Format(time.RFC3339)},
		Weekly:   &subscription.QuotaWindow{RemainingPercent: 99, ResetsAt: time.Now().Add(7 * 24 * time.Hour).UTC().Format(time.RFC3339)}}
}
func newFixture(t *testing.T) *activationFixture {
	t.Helper()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "activation.db")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at) VALUES('account-one','builtin_openai','Test','unused',1,'2026-01-01T00:00:00Z','now')`); err != nil {
		t.Fatal(err)
	}
	f := &activationFixture{db: db, accounts: &testAccounts{item: account.Item{ID: "account-one", ProviderID: "builtin_openai", Enabled: true, AuthMethod: account.AuthSubscription, AuthState: "authenticated", AuthAccountID: "official-one", AuthEmail: "one@example.test"}}, providers: &testProviders{true}, quotas: &testQuotas{quota: unstartedQuota()}}
	f.service = New(db, f.accounts, f.providers, f.quotas, sendFunc(func(_ context.Context, item account.Item, _ SendPlan) error {
		if item.ID != "account-one" || item.AuthAccountID != "official-one" {
			t.Error("wrong account")
		}
		f.sends.Add(1)
		f.quotas.set(activeQuota())
		return nil
	}))
	return f
}
func (f *activationFixture) enable(t *testing.T) {
	t.Helper()
	if _, err := gateway.NewCodexSettingsService(f.db).SetAutoQuotaActivation(context.Background(), true); err != nil {
		t.Fatal(err)
	}
}
