package routing

import (
	"context"
	"encoding/json"
	catalog "omniproxy/internal/core/modelcatalog"
	"testing"
)

func TestNativeAliasesKeepOnlySelectedAccountServiceTiers(t *testing.T) {
	s, db := fixture(t)
	_, err := db.Exec(`UPDATE accounts SET provider_id='builtin_openai',auth_method='subscription_login',auth_state='authenticated',auth_account_id=id WHERE id IN ('acct_old','acct_other');
INSERT INTO account_models(account_id,model,name,fetched_at,account_version,metadata_json)
SELECT id,'gpt-test','GPT Test','now',version,
CASE WHEN id='acct_other' THEN '{"serviceTiers":[{"id":"priority","name":"Fast","description":"Official usage"}]}' ELSE '{}' END
FROM account_catalog_versions WHERE id IN ('acct_old','acct_other')`)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for alias, accountID := range map[string]string{"fast-route": "acct_other", "unknown-route": "acct_old"} {
		_, err := s.Create(ctx, Input{Alias: alias, ProviderID: "builtin_openai", Model: "gpt-test", AccountID: accountID, Enabled: true})
		if err != nil {
			t.Fatal(err)
		}
	}
	entries, err := s.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	body, err := catalog.CodexJSON(entries)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Models []struct {
			Slug         string                `json:"slug"`
			ServiceTiers []catalog.ServiceTier `json:"service_tiers"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, model := range result.Models {
		switch model.Slug {
		case "fast-route":
			found++
			if len(model.ServiceTiers) != 1 || model.ServiceTiers[0].ID != "priority" || model.ServiceTiers[0].Description != "Official usage" {
				t.Fatal("native alias lost target Fast capability")
			}
		case "unknown-route":
			found++
			if len(model.ServiceTiers) != 0 {
				t.Fatal("alias borrowed another account's capability")
			}
		}
	}
	if found != 2 {
		t.Fatal("native aliases missing from catalog")
	}
}
