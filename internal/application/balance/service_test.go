package balance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/deepseek"
)

type balanceStore struct{ last []byte }

func (s *balanceStore) Put([]byte) (string, error) { return "test-reference", nil }
func (s *balanceStore) Get(string) ([]byte, error) {
	s.last = []byte("placeholder-balance-key")
	return s.last, nil
}
func (s *balanceStore) Delete(string) error { return nil }

type balanceClient struct {
	calls int
	fail  bool
}

func (c *balanceClient) Balance(_ context.Context, key []byte) (deepseek.Balance, error) {
	c.calls++
	if string(key) != "placeholder-balance-key" {
		return deepseek.Balance{}, errors.New("bad-key")
	}
	if c.fail {
		return deepseek.Balance{}, errors.New("private-adapter-failure")
	}
	return deepseek.Balance{Available: true, Entries: []deepseek.BalanceInfo{{Currency: "CNY", Total: "110.00", Granted: "10.00", ToppedUp: "100.00"}}}, nil
}

func TestBalanceUsesProtectedKeyOnlyForDeepSeekAndClearsIt(t *testing.T) {
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "balance.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	store, client := &balanceStore{}, &balanceClient{}
	accounts := account.NewService(db, store)
	item, err := accounts.Create(context.Background(), account.CreateInput{ProviderID: "builtin_deepseek", Name: "余额测试", APIKey: "placeholder-balance-key"})
	if err != nil {
		t.Fatal(err)
	}
	service := New(accounts, provider.NewService(db), client)
	info, err := service.Read(context.Background(), item.ID)
	if err != nil || len(info.Entries) != 1 || info.Entries[0].Total != "110.00" || info.FetchedAt == "" {
		t.Fatal("balance query failed")
	}
	for _, b := range store.last {
		if b != 0 {
			t.Fatal("protected key not cleared")
		}
	}
	other, err := accounts.Create(context.Background(), account.CreateInput{ProviderID: "builtin_openai", Name: "其他账号", APIKey: "placeholder-balance-key"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Read(context.Background(), other.ID); err == nil || client.calls != 1 {
		t.Fatal("other provider credential reached DeepSeek")
	}
	client.fail = true
	if _, err := service.Read(context.Background(), item.ID); err == nil || err.Error() != "balance: query_failed" {
		t.Fatal("raw balance adapter failure exposed")
	}
	for _, b := range store.last {
		if b != 0 {
			t.Fatal("key not cleared after query failure")
		}
	}
}
