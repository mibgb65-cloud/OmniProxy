package routing

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

func kimiOrderFixture(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	s, db := fixture(t)
	for i, pool := range []struct{ id, region, mode string }{{"api-cn", "cn", "payg"}, {"api-global", "global", "payg"}, {"code-cn", "cn", "coding"}, {"code-global", "global", "coding"}} {
		stamp := "2026-01-0" + string(rune('1'+i)) + "T00:00:00Z"
		if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at)VALUES(?,'builtin_kimi',?,'protected-ref',1,?,?)`, pool.id, pool.id, stamp, stamp); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO kimi_account_options(account_id,region,billing_mode)VALUES(?,?,?)`, pool.id, pool.region, pool.mode); err != nil {
			t.Fatal(err)
		}
	}
	return s, db
}

func TestKimiBillingThenRegionPriorityAndFixedCompatibility(t *testing.T) {
	s, db := kimiOrderFixture(t)
	ctx := context.Background()
	check := func(model, want string) {
		t.Helper()
		p, err := s.Resolve(ctx, model)
		if err != nil || !p.Routable || p.AccountID != want {
			t.Fatalf("%s %+v %v", model, p, err)
		}
	}
	check("kimi-k3", "code-cn")
	if _, err := s.SetKimiPreferredRegion(ctx, "global"); err != nil {
		t.Fatal(err)
	}
	check("kimi-k3", "code-global")
	if _, err := s.SetKimiPreferredBillingMode(ctx, "payg"); err != nil {
		t.Fatal(err)
	}
	check("kimi-k3", "api-global")
	check("k3", "code-global")
	check("kimi-for-coding-highspeed", "code-global")
	check("kimi-for-coding", "code-global")
	check("kimi-cn/kimi-k3", "api-cn")
	check("kimi-code-cn/kimi-k3", "code-cn")
	if _, err := s.Create(ctx, Input{Alias: "fixed-code", ProviderID: "builtin_kimi", AccountID: "code-cn", Model: "kimi-k3", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	check("fixed-code", "code-cn")
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id='api-global'`); err != nil {
		t.Fatal(err)
	}
	check("kimi-k3", "api-cn")
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id='api-cn'`); err != nil {
		t.Fatal(err)
	}
	check("kimi-k3", "code-global")
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id='code-cn'`); err != nil {
		t.Fatal(err)
	}
	p, err := s.Resolve(ctx, "fixed-code")
	if err != nil || p.Routable {
		t.Fatal("fixed disabled account fell back")
	}
}

func TestKimiDeduplicationAndSelectedAccountCapabilities(t *testing.T) {
	s, _ := kimiOrderFixture(t)
	ctx := context.Background()
	for _, mode := range []string{"coding", "payg"} {
		if _, err := s.SetKimiPreferredBillingMode(ctx, mode); err != nil {
			t.Fatal(err)
		}
		entries, err := s.Catalog(ctx)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for _, e := range entries {
			if e.ProviderKind != "kimi" {
				continue
			}
			if seen[e.ID] || strings.Contains(e.ID, "/") {
				t.Fatal("duplicate region or billing model", e.ID)
			}
			seen[e.ID] = true
			if e.ID == "kimi-k3" {
				want := int64(1048576)
				if mode == "coding" {
					want = 262144
				}
				if e.Metadata.ContextWindow == nil || *e.Metadata.ContextWindow != want {
					t.Fatal("wrong target context", e.ID)
				}
			}
			if e.ID == "kimi-k2.7-code-highspeed" {
				want := 1
				if mode == "coding" {
					want = 3
				}
				if len(e.Protocols) != want {
					t.Fatal("cross-pool protocol capabilities mixed")
				}
			}
		}
		if len(seen) != 6 || !seen["kimi-for-coding"] || !seen["k3-256k"] {
			t.Fatal("distinct subscription models lost", seen)
		}
	}
}

func TestKimiSettingsValidatePersistAndStayIndependent(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	v, err := s.GetKimiSettings(ctx)
	if err != nil || v.PreferredBillingMode != "coding" || v.PreferredRegion != "cn" {
		t.Fatal("default missing", v, err)
	}
	if _, err := s.SetKimiPreferredRegion(ctx, "global"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetKimiPreferredBillingMode(ctx, "payg"); err != nil {
		t.Fatal(err)
	}
	v, err = New(db, nil, nil).GetKimiSettings(ctx)
	if err != nil || v.PreferredBillingMode != "payg" || v.PreferredRegion != "global" {
		t.Fatal("not persisted", v, err)
	}
	for _, bad := range []string{"", "subscription", "CN", " payg", strings.Repeat("a", 8192)} {
		if _, err := s.SetKimiPreferredBillingMode(ctx, bad); err == nil {
			t.Fatal("invalid billing accepted")
		}
		if _, err := s.SetKimiPreferredRegion(ctx, bad); err == nil {
			t.Fatal("invalid region accepted")
		}
	}
	if m, err := s.GetMiMoSettings(ctx); err != nil || m.PreferredBillingMode != "token_plan" {
		t.Fatal("MiMo preference changed")
	}
	if m, err := s.GetStepFunSettings(ctx); err != nil || m.PreferredBillingMode != "step_plan" {
		t.Fatal("StepFun preference changed")
	}
	if _, err := db.Exec(`DELETE FROM kimi_settings`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetKimiSettings(ctx); err == nil {
		t.Fatal("missing settings defaulted")
	}
	if _, err := s.SetKimiPreferredRegion(ctx, "cn"); err == nil {
		t.Fatal("missing settings save accepted")
	}
}
