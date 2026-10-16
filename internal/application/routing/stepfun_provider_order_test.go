package routing

import (
	"context"
	"testing"
)

func TestStepFunBillingOrderIsWithinProviderPriority(t *testing.T) {
	s, db := stepOrderFixture(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO providers(id,kind,source,name,base_url,enabled,created_at,updated_at) VALUES('stepfun-backup','stepfun','custom','Backup','',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z');
 INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at) VALUES('backup-api','stepfun-backup','Backup API','protected-backup',1,'2026-01-04T00:00:00Z','2026-01-04T00:00:00Z');
 INSERT INTO stepfun_account_options(account_id,region,billing_mode)VALUES('backup-api','cn','payg')`); err != nil {
		t.Fatal(err)
	}
	if err := s.SetProviderOrder(ctx, "step-5-preview", []string{"stepfun-backup", "builtin_stepfun"}); err != nil {
		t.Fatal(err)
	}
	preview, err := s.Resolve(ctx, "step-5-preview")
	if err != nil || !preview.Routable || preview.AccountID != "backup-api" || !preview.PreferenceApplied {
		t.Fatal("billing priority overrode provider priority", preview, err)
	}
	if _, err := db.Exec(`UPDATE providers SET enabled=0 WHERE id='stepfun-backup'`); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "step-5-preview")
	if err != nil || !preview.Routable || preview.AccountID != "step-plan" {
		t.Fatal("provider fallback did not apply billing order", preview, err)
	}
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id IN ('step-plan','step-plan-global')`); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "step-5-preview")
	if err != nil || !preview.Routable || preview.AccountID != "step-api" {
		t.Fatal("disabled plan prevented API fallback", preview, err)
	}
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id='step-api'`); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "step-5-preview")
	if err != nil || preview.Routable || preview.ErrorCode != "no_enabled_account" {
		t.Fatal("ineligible account selected", preview, err)
	}
}
