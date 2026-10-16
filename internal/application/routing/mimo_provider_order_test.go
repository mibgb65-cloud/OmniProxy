package routing

import (
	"context"
	"testing"
)

func TestMiMoBillingOrderIsWithinProviderPriority(t *testing.T) {
	s, db := mimoOrderFixture(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO providers(id,kind,source,name,base_url,enabled,created_at,updated_at) VALUES('mimo-backup','mimo','custom','Backup','',1,'2026-01-01T00:00:00Z','2026-01-01T00:00:00Z');
 INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at) VALUES('backup-api','mimo-backup','Backup API','protected-backup',1,'2026-01-04T00:00:00Z','2026-01-04T00:00:00Z');
 INSERT INTO mimo_account_options(account_id,region,billing_mode)VALUES('backup-api','','payg')`); err != nil {
		t.Fatal(err)
	}
	if err := s.SetProviderOrder(ctx, "mimo-v2.6-pro", []string{"mimo-backup", "builtin_mimo"}); err != nil {
		t.Fatal(err)
	}
	preview, err := s.Resolve(ctx, "mimo-v2.6-pro")
	if err != nil || !preview.Routable || preview.AccountID != "backup-api" || !preview.PreferenceApplied {
		t.Fatal("billing priority overrode provider priority", preview, err)
	}
	if _, err := db.Exec(`UPDATE providers SET enabled=0 WHERE id='mimo-backup'`); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "mimo-v2.6-pro")
	if err != nil || !preview.Routable || preview.AccountID != "mimo-plan" {
		t.Fatal("provider fallback did not apply billing order", preview, err)
	}
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id IN ('mimo-plan','mimo-plan-new')`); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "mimo-v2.6-pro")
	if err != nil || !preview.Routable || preview.AccountID != "mimo-payg" {
		t.Fatal("disabled plan prevented API fallback", preview, err)
	}
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id='mimo-payg'`); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "mimo-v2.6-pro")
	if err != nil || preview.Routable || preview.ErrorCode != "no_enabled_account" {
		t.Fatal("ineligible account selected", preview, err)
	}
}
