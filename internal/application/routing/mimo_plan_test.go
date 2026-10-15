package routing

import (
	"context"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/provider/mimo"
)

func TestMiMoPlanRoutingSeparatesClustersAndBillingPools(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	accounts := account.NewService(db, &mimoRoutingStore{})
	cnID := ""
	for _, region := range []string{"cn", "sgp", "ams"} {
		a, err := accounts.Create(ctx, account.CreateInput{ProviderID: "builtin_mimo", Name: region, APIKey: "tp-placeholder-key", MiMo: &mimo.Options{Region: region, BillingMode: "token_plan"}})
		if err != nil {
			t.Fatal(err)
		}
		preview, err := s.Resolve(ctx, "mimo-"+region+"-plan/mimo-v2.6-pro")
		if err != nil || !preview.Routable || preview.AccountID != a.ID || preview.Model != "mimo-v2.6-pro" {
			t.Fatalf("preview=%+v %v", preview, err)
		}
		if region == "cn" {
			cnID = a.ID
			if _, err := s.Create(ctx, Input{Alias: "fixed-cn", ProviderID: a.ProviderID, AccountID: a.ID, Model: "mimo-v2.6-pro", Enabled: true}); err != nil {
				t.Fatal(err)
			}
		}
	}
	preview, err := s.Resolve(ctx, "mimo-v2.6-pro")
	if err != nil || preview.Routable || preview.ErrorCode != "ambiguous_mimo_pool" {
		t.Fatal("ambiguous cluster guessed", preview, err)
	}
	preview, err = s.Resolve(ctx, "fixed-cn")
	if err != nil || !preview.Routable {
		t.Fatal("fixed account lost", err)
	}
	entries, err := s.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.ProviderKind == mimo.Kind && entry.TargetModel == "mimo-v2.6-pro-ultraspeed" {
			t.Fatal("plan advertises ordinary-only model")
		}
	}
	if _, err := accounts.Create(ctx, account.CreateInput{ProviderID: "builtin_mimo", Name: "Payg", APIKey: "sk-placeholder-key"}); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "mimo-api/mimo-v2.6-pro-ultraspeed")
	if err != nil || !preview.Routable {
		t.Fatal("payg capability lost", preview, err)
	}
	preview, err = s.Resolve(ctx, "mimo-cn-plan/mimo-v2.6-pro-ultraspeed")
	if err != nil || preview.Routable {
		t.Fatal("unsupported model routed to plan", err)
	}
	if _, err := s.Create(ctx, Input{Alias: "mimo-cn-plan/mimo-v2.6-pro", ProviderID: "builtin_mimo", AccountID: cnID, Model: "mimo-v2.6-flash", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "mimo-cn-plan/mimo-v2.6-pro")
	if err != nil || !preview.Routable || preview.Model != "mimo-v2.6-flash" || preview.AccountID != cnID {
		t.Fatal("explicit rule did not override generated alias", preview, err)
	}
}

type mimoRoutingStore struct{}

func (*mimoRoutingStore) Put([]byte) (string, error) { return "protected-ref", nil }
func (*mimoRoutingStore) Get(string) ([]byte, error) { return []byte("tp-placeholder-key"), nil }
func (*mimoRoutingStore) Delete(string) error        { return nil }
