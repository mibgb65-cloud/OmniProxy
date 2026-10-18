package routing

import (
	"context"
	"omniproxy/internal/application/account"
	"omniproxy/internal/provider/mimo"
	"testing"
)

func TestCandidatePlanPreservesFixedAndConfigurationBoundaries(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	plan, err := s.ResolvePlan(ctx, "deepseek-flash")
	if err != nil || plan.Fixed || len(plan.Candidates) != 2 || plan.Candidates[0].AccountID != "acct_old" || plan.Candidates[1].AccountID != "acct_new" {
		t.Fatalf("wrong plan: %+v %v", plan, err)
	}
	i := input("fixed")
	i.AccountID = "acct_new"
	if _, err = s.Create(ctx, i); err != nil {
		t.Fatal(err)
	}
	plan, err = s.ResolvePlan(ctx, "fixed")
	if err != nil || !plan.Fixed || len(plan.Candidates) != 1 || plan.Candidates[0].AccountID != "acct_new" {
		t.Fatal("fixed account widened", plan, err)
	}
	if _, err = db.Exec(`UPDATE accounts SET enabled=0 WHERE id='acct_new'`); err != nil {
		t.Fatal(err)
	}
	plan, err = s.ResolvePlan(ctx, "deepseek-flash")
	if err != nil || len(plan.Candidates) != 1 {
		t.Fatal("disabled account included", plan, err)
	}
}

func TestAccountPoolSeparatesBillingRegionAndAuthentication(t *testing.T) {
	a := account.Item{ID: "a", ProviderID: "p", AuthMethod: "api_key", MiMo: &mimo.Options{BillingMode: "token_plan", Region: "sgp"}}
	b := a
	b.ID = "b"
	if !SameAccountPool(a, b) {
		t.Fatal("same channel excluded")
	}
	for _, change := range []func(*account.Item){
		func(b *account.Item) { b.ProviderID = "another" },
		func(b *account.Item) { b.AuthMethod = "subscription_login" },
		func(b *account.Item) { b.MiMo = &mimo.Options{BillingMode: "payg"} },
		func(b *account.Item) { b.MiMo = &mimo.Options{BillingMode: "token_plan", Region: "cn"} },
	} {
		b := a
		b.ID = "b"
		change(&b)
		if SameAccountPool(a, b) {
			t.Fatal("crossed account pool", b)
		}
	}
}
