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
	"omniproxy/internal/provider/kimi"
)

func TestKimiBalancePreservesIdentityAndAvailability(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "kimi.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	store := &balanceStore{}
	accounts := account.NewService(db, store)
	a, err := accounts.Create(ctx, account.CreateInput{ProviderID: "builtin_kimi", Name: "Kimi", APIKey: "placeholder-kimi-key", Kimi: &kimi.Options{Region: "cn"}})
	if err != nil {
		t.Fatal(err)
	}
	// 停用只影响转发资格，余额仍可查询。
	if _, err := accounts.SetEnabled(ctx, a.ID, false); err != nil {
		t.Fatal(err)
	}
	s := New(accounts, provider.NewService(db), nil)
	amount, mutate, calls, status := "0.000000000000000001", "", 0, 200
	var consumed []byte
	s.kimi = kimi.New(stepBalanceTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://api.moonshot.cn/v1/users/me/balance" {
			t.Fatal("wrong region")
		}
		if mutate != "" {
			current, err := accounts.Metadata(ctx, a.ID)
			if err != nil {
				t.Fatal(err)
			}
			input := account.UpdateInput{ID: a.ID, Name: a.Name}
			if mutate == "region" {
				o := *current.Kimi
				o.Region = "global"
				input.Kimi = &o
			} else {
				input.APIKey = "placeholder-replacement-key"
			}
			if _, err := accounts.Update(ctx, input); err != nil {
				t.Fatal(err)
			}
		}
		return &http.Response{StatusCode: status, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"code":0,"status":true,"scode":"0x0","data":{"available_balance":` + amount + `,"voucher_balance":1.00,"cash_balance":-3.00}}`))}, nil
	}))
	for _, tc := range []struct {
		amount    string
		available bool
	}{{"0.000000000000000001", true}, {"0", false}, {"-1.50", false}} {
		amount = tc.amount
		value, err := s.Read(ctx, a.ID)
		if err != nil || value.Kimi == nil || value.Kimi.Available != amount || value.Kimi.Cash != "-3.00" || value.Available != tc.available || len(value.Entries) != 0 || value.FetchedAt == "" {
			t.Fatalf("wrong balance semantics: %+v %v", value, err)
		}
		consumed = store.last
		for _, b := range consumed {
			if b != 0 {
				t.Fatal("protected key not cleared")
			}
		}
	}
	status = 401
	if _, err := s.Read(ctx, a.ID); err == nil || err.Error() != "balance: unauthorized" {
		t.Fatalf("unsafe error: %v", err)
	}
	status = 200
	mutate = "key"
	if _, err := s.Read(ctx, a.ID); err == nil || err.Error() != "balance: configuration_unavailable" {
		t.Fatal("returned balance after key replacement")
	}
	mutate = "region"
	if _, err := s.Read(ctx, a.ID); err == nil || err.Error() != "balance: configuration_unavailable" {
		t.Fatal("returned balance after platform change")
	}
	if calls != 6 {
		t.Fatalf("unexpected retries: %d", calls)
	}
}
