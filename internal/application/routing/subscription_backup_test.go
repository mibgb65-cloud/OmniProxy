package routing

import (
	"context"
	"testing"
)

func TestSubscriptionBackupWithoutCatalogDoesNotExpandDiscovery(t *testing.T) {
	s, db := fixture(t)
	_, err := db.Exec(`DELETE FROM accounts WHERE id='acct_other'; UPDATE accounts SET provider_id='builtin_openai',auth_method='subscription_login',auth_state='authenticated',auth_email=id||'@example.invalid',auth_account_id=id; INSERT INTO account_models(account_id,model,name,fetched_at,account_version) SELECT 'acct_old','gpt-test','GPT','2026-01-01T00:00:00Z',version FROM account_catalog_versions WHERE id='acct_old'`)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := s.ResolvePlan(context.Background(), "gpt-test")
	if err != nil || len(plan.Candidates) != 2 || plan.Candidates[0].AccountID != "acct_old" || plan.Candidates[1].AccountID != "acct_new" {
		t.Fatal("newly logged in backup omitted", plan, err)
	}
	_, err = db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version) SELECT 'acct_new','other-model','Other','2026-01-01T00:00:00Z',version FROM account_catalog_versions WHERE id='acct_new'`)
	if err != nil {
		t.Fatal(err)
	}
	plan, err = s.ResolvePlan(context.Background(), "gpt-test")
	if err != nil || len(plan.Candidates) != 1 {
		t.Fatal("known unsupported capability widened", plan, err)
	}
	plan, err = s.ResolvePlan(context.Background(), "not-discovered")
	if err != nil || plan.Decision.Routable {
		t.Fatal("backup invented advertised model", plan, err)
	}
}
