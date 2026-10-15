package minimaxquota

import (
	"context"
	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/minimax"
	"os"
	"path/filepath"
	"testing"
)

type testStore struct{}

func (testStore) Put([]byte) (string, error) { return "protected-test-ref", nil }
func (testStore) Get(string) ([]byte, error) { return []byte("placeholder-minimax-key"), nil }
func (testStore) Delete(string) error        { return nil }

type quotaClient func(context.Context, minimax.Options, []byte) (minimax.Quota, error)

func (f quotaClient) ReadQuota(ctx context.Context, o minimax.Options, k []byte) (minimax.Quota, error) {
	return f(ctx, o, k)
}

func TestQueryCannotPublishResourcesAfterRegionOrKeyTypeChanges(t *testing.T) {
	for _, field := range []string{"region", "billing", "key"} {
		t.Run(field, func(t *testing.T) {
			ctx := context.Background()
			db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "minimax.db")})
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
				t.Fatal(err)
			}
			accounts := account.NewService(db, testStore{})
			a, err := accounts.Create(ctx, account.CreateInput{ProviderID: "builtin_minimax", Name: "test", APIKey: "placeholder-minimax-key", MiniMax: &minimax.Options{Region: "cn", BillingMode: "subscription"}})
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			client := quotaClient(func(ctx context.Context, o minimax.Options, key []byte) (minimax.Quota, error) {
				calls++
				if o.Region != "cn" || o.BillingMode != "subscription" || string(key) != "placeholder-minimax-key" {
					t.Fatal("incorrect credential scope")
				}
				next := *a.MiniMax
				replacement := ""
				switch field {
				case "region":
					next.Region = "global"
				case "billing":
					next.BillingMode = "payg"
				case "key":
					replacement = "placeholder-replacement"
				}
				if _, err := accounts.Update(ctx, account.UpdateInput{ID: a.ID, Name: a.Name, APIKey: replacement, MiniMax: &next}); err != nil {
					t.Fatal(err)
				}
				return minimax.Quota{FetchedAt: "old", Buckets: []minimax.QuotaBucket{}}, nil
			})
			service := New(accounts, provider.NewService(db), client)
			if q, err := service.Read(ctx, a.ID); err == nil || q.FetchedAt != "" || calls != 1 {
				t.Fatal("stale resource published", q, err)
			}
		})
	}
}

func TestPaygAndOtherProvidersDoNotQuerySubscription(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "minimax.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	accounts := account.NewService(db, testStore{})
	service := New(accounts, provider.NewService(db), quotaClient(func(context.Context, minimax.Options, []byte) (minimax.Quota, error) {
		t.Fatal("unsupported query read key")
		return minimax.Quota{}, nil
	}))
	for _, id := range []string{"builtin_minimax", "builtin_deepseek"} {
		a, err := accounts.Create(ctx, account.CreateInput{ProviderID: id, Name: "test", APIKey: "placeholder-minimax-key"})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.Read(ctx, a.ID); err == nil {
			t.Fatal("unsupported account queried")
		}
	}
}
