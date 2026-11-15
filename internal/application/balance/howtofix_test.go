package balance

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
	"omniproxy/internal/provider/howtofix"
)

type howtofixTransport struct {
	subscription string
	status       int
	calls        []string
}

func (t *howtofixTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	t.calls = append(t.calls, r.URL.Path)
	if r.Header.Get("Authorization") != "Bearer placeholder-balance-key" {
		return &http.Response{StatusCode: http.StatusUnauthorized, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
	}
	if r.URL.Path != "/v1/dashboard/billing/subscription" {
		return &http.Response{StatusCode: http.StatusNotFound, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(""))}, nil
	}
	status := t.status
	if status == 0 {
		status = http.StatusOK
	}
	return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(t.subscription))}, nil
}

// TestHowToFixBalanceReadsPrepaidBilling 断言余额来自账号自己的 Key，
// 剩余为零不算可用，上游错误按固定码返回，且凭据用完后被清零。
func TestHowToFixBalanceReadsPrepaidBilling(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "howtofix.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	store := &balanceStore{}
	accounts := account.NewService(db, store)
	item, err := accounts.Create(ctx, account.CreateInput{ProviderID: "builtin_howtofix", Name: "HowToFix", APIKey: "placeholder-howtofix-key"})
	if err != nil {
		t.Fatal(err)
	}
	// 查询用的是可注入的假主机，避免测试连真实上游。
	if _, err := db.Exec(`UPDATE providers SET base_url='https://billing.example.test/v1' WHERE id='builtin_howtofix'`); err != nil {
		t.Fatal(err)
	}
	transport := &howtofixTransport{subscription: `{"object":"billing_subscription","hard_limit_usd":12.345,"system_hard_limit_usd":12.345}`}
	s := New(accounts, provider.NewService(db), nil)
	s.howtofix = howtofix.New(transport)

	value, err := s.Read(ctx, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if value.HowToFix == nil || value.HowToFix.Remaining != "12.35" || value.HowToFix.Currency != "USD" {
		t.Fatalf("balance = %+v", value)
	}
	if !value.Available || value.FetchedAt == "" || len(value.Entries) != 0 {
		t.Fatalf("availability = %+v", value)
	}
	for _, value := range store.last {
		if value != 0 {
			t.Fatal("protected key not cleared")
		}
	}
	if len(transport.calls) != 1 {
		t.Fatalf("billing calls = %v, want only the subscription endpoint", transport.calls)
	}

	transport.subscription = `{"hard_limit_usd":0}`
	value, err = s.Read(ctx, item.ID)
	if err != nil || value.HowToFix == nil || value.Available {
		t.Fatalf("zero balance must not be available: %+v %v", value, err)
	}

	transport.status = http.StatusUnauthorized
	if _, err := s.Read(ctx, item.ID); err == nil || err.Error() != "balance: unauthorized" {
		t.Fatalf("unsafe error: %v", err)
	}

	transport.status = http.StatusInternalServerError
	if _, err := s.Read(ctx, item.ID); err == nil || err.Error() != "balance: service_unavailable" {
		t.Fatalf("unsafe error: %v", err)
	}
}
