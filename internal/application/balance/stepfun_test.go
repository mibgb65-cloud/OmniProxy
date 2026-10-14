package balance

import (
	"context"
	"io"
	"net/http"
	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/stepfun"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type stepBalanceTransport func(*http.Request) (*http.Response, error)

func (f stepBalanceTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestStepBalanceIsIndependentFromPlanAndConfigurationChanges(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "step.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	accounts := account.NewService(db, &balanceStore{})
	a, err := accounts.Create(ctx, account.CreateInput{ProviderID: "builtin_stepfun", Name: "Step", APIKey: "placeholder-step-key", StepFun: &stepfun.Options{Region: "global", BillingMode: "payg"}})
	if err != nil {
		t.Fatal(err)
	}
	s := New(accounts, provider.NewService(db), nil)
	calls := 0
	switchMode := false
	s.stepfun = stepfun.New(stepBalanceTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != "https://api.stepfun.ai/v1/accounts" {
			t.Fatal("wrong regional balance query")
		}
		if switchMode {
			o := *a.StepFun
			o.BillingMode = "step_plan"
			if _, err := accounts.Update(ctx, account.UpdateInput{ID: a.ID, Name: a.Name, StepFun: &o}); err != nil {
				t.Fatal(err)
			}
		}
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"object":"account","type":"postpaid","balance":-2,"total_cash_balance":100,"total_voucher_balance":20}`))}, nil
	}))
	value, err := s.Read(ctx, a.ID)
	if err != nil || value.StepFun == nil || value.StepFun.Amount != "-2" || len(value.Entries) != 0 {
		t.Fatalf("wrong balance semantics: %+v %v", value, err)
	}
	switchMode = true
	if _, err := s.Read(ctx, a.ID); err == nil {
		t.Fatal("old ordinary balance returned after switching to subscription")
	}
	if _, err := s.Read(ctx, a.ID); err == nil || calls != 2 {
		t.Fatal("subscription used ordinary balance API")
	}
}
