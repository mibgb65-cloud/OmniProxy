package account

import (
	"context"
	"errors"
	"omniproxy/internal/provider/stepfun"
	"testing"
)

func TestStepOptionsLifecycleAndVersionIsolation(t *testing.T) {
	s, store := testService(t)
	ctx := context.Background()
	o := &stepfun.Options{Region: "global", BillingMode: "step_plan"}
	a, err := s.Create(ctx, CreateInput{ProviderID: "builtin_stepfun", Name: "Step", APIKey: "placeholder-step-key", StepFun: o})
	if err != nil || a.StepFun == nil || a.StepFun.Version != 1 {
		t.Fatalf("create: %+v %v", a, err)
	}
	var version string
	if err := s.db.QueryRow(`SELECT version FROM account_catalog_versions WHERE id=?`, a.ID).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version)VALUES(?,'step-5-preview','Step','now',?)`, a.ID, version); err != nil {
		t.Fatal(err)
	}
	next := *a.StepFun
	next.BillingMode = "payg"
	updated, err := s.Update(ctx, UpdateInput{ID: a.ID, Name: "Step renamed", StepFun: &next})
	if err != nil || updated.Account.StepFun.Version != 2 {
		t.Fatalf("update: %+v %v", updated, err)
	}
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM account_models WHERE account_id=?`, a.ID).Scan(&count); err != nil || count != 0 {
		t.Fatal("old model cache survived channel switch")
	}
	if _, err := s.Update(ctx, UpdateInput{ID: a.ID, Name: "stale", StepFun: a.StepFun}); !errors.Is(err, ErrConflict) {
		t.Fatal("stale options overwrote new billing")
	}
	if err := s.UseAPIKeyAt(ctx, a.ID, a.UpdatedAt, func([]byte) error { t.Fatal("stale target read key"); return nil }); !errors.Is(err, ErrConflict) {
		t.Fatal("stale gateway snapshot accepted")
	}
	if string(store.secret) != "placeholder-step-key" {
		t.Fatal("option change replaced credential")
	}
	unchanged, err := s.Update(ctx, UpdateInput{ID: a.ID, Name: "only name"})
	if err != nil || unchanged.Account.StepFun.Version != 2 {
		t.Fatal("name change changed channel identity")
	}
}

func TestStepOptionsValidationAndBatchRollback(t *testing.T) {
	s, _ := testService(t)
	store := &batchStore{values: make(map[string][]byte)}
	s.store = store
	ctx := context.Background()
	for _, input := range []CreateInput{
		{ProviderID: "builtin_stepfun", APIKey: "placeholder-step-key", StepFun: &stepfun.Options{Region: "other", BillingMode: "payg"}},
		{ProviderID: "builtin_openai", APIKey: "placeholder-step-key", StepFun: &stepfun.Options{Region: "cn", BillingMode: "payg"}},
		{ProviderID: "builtin_stepfun", APIKey: "placeholder key", StepFun: &stepfun.Options{Region: "cn", BillingMode: "payg"}},
	} {
		if _, err := s.Create(ctx, input); err == nil {
			t.Fatal("invalid Step configuration accepted")
		}
	}
	if store.puts != 0 {
		t.Fatal("invalid configuration reached protected storage")
	}
	if _, err := s.db.Exec(`CREATE TRIGGER reject_second_step BEFORE INSERT ON stepfun_account_options WHEN (SELECT COUNT(*) FROM stepfun_account_options)>0 BEGIN SELECT RAISE(ABORT,'test options failure'); END`); err != nil {
		t.Fatal(err)
	}
	_, err := s.CreateBatch(ctx, BatchCreateInput{ProviderID: "builtin_stepfun", APIKeys: []string{"placeholder-key-1", "placeholder-key-2"}, StepFun: &stepfun.Options{Region: "cn", BillingMode: "step_plan"}})
	if err == nil || len(store.values) != 0 {
		t.Fatal("failed option batch retained credentials")
	}
	var count int
	if err := s.db.QueryRow(`SELECT count(*) FROM accounts`).Scan(&count); err != nil || count != 0 {
		t.Fatal("failed options batch retained accounts")
	}
}
