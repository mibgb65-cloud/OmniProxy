package routing

import (
	"context"
	"database/sql"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
)

func preferenceFixture(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	s, db := fixture(t)
	_, err := db.Exec(`INSERT INTO providers(id,kind,source,name,base_url,enabled,created_at,updated_at) SELECT 'backup_ds',kind,'custom','Backup DeepSeek',base_url,1,created_at,updated_at FROM providers WHERE id='builtin_deepseek';
INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at) VALUES('acct_backup','backup_ds','Backup account','unused',1,'2026-02-01T00:00:00Z','now');
INSERT INTO account_models(account_id,model,name,fetched_at,account_version,metadata_json) VALUES('acct_backup','deepseek-flash','Backup Model','now','unused:','{"contextWindow":222000}');`)
	if err != nil {
		t.Fatal(err)
	}
	return s, db
}

func TestDefaultRoutesAreDerivedWithoutCreatingRules(t *testing.T) {
	s, db := preferenceFixture(t)
	ctx := context.Background()
	items, err := s.DefaultRoutes(ctx)
	if err != nil || len(items) != 2 {
		t.Fatalf("defaults: %+v %v", items, err)
	}
	if items[0].Model != "deepseek-flash" || !items[0].Preview.Routable || len(items[0].Providers) != 2 || items[0].Preview.AccountID != "acct_old" {
		t.Fatalf("default mapping not ready: %+v", items[0])
	}
	var count int
	if err := db.QueryRow(`SELECT (SELECT COUNT(*) FROM model_routes)+(SELECT COUNT(*) FROM model_provider_preferences)`).Scan(&count); err != nil || count != 0 {
		t.Fatal("reading defaults wrote configuration", err)
	}
	if _, err := db.Exec(`UPDATE accounts SET auth_method='subscription_login',auth_state='authenticated' WHERE id='acct_other'; INSERT INTO account_models(account_id,model,name,fetched_at,account_version) VALUES('acct_other','gpt-synced','Synced GPT','now','unused:')`); err != nil {
		t.Fatal(err)
	}
	items, err = s.DefaultRoutes(ctx)
	if err != nil || len(items) != 3 || items[2].Model != "gpt-synced" || items[2].Name != "Synced GPT" || items[2].Preview.ProviderID != "builtin_openai" {
		t.Fatalf("synced model not automatically mapped: %+v %v", items, err)
	}
}

func TestPreferredProviderPersistsAndChangesCatalogMetadata(t *testing.T) {
	s, db := preferenceFixture(t)
	ctx := context.Background()
	if err := s.SetPreferredProvider(ctx, "deepseek-flash", "backup_ds"); err != nil {
		t.Fatal(err)
	}
	s = New(db, provider.NewService(db), account.NewService(db, unreadStore{}))
	preview, err := s.Resolve(ctx, "deepseek-flash")
	if err != nil || !preview.PreferenceApplied || preview.ProviderID != "backup_ds" || preview.AccountID != "acct_backup" || preview.CandidateCount != 3 {
		t.Fatalf("preferred newer account not selected: %+v %v", preview, err)
	}
	entries, err := s.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.ID == "deepseek-flash" && (entry.ProviderID != "backup_ds" || entry.Metadata.ContextWindow == nil || *entry.Metadata.ContextWindow != 222000) {
			t.Fatal("catalog metadata did not follow the selected provider")
		}
	}
	items, err := s.DefaultRoutes(ctx)
	if err != nil || items[0].PreferredProviderID != "backup_ds" || items[0].PreferredProviderName != "Backup DeepSeek" {
		t.Fatalf("preference not visible: %+v %v", items, err)
	}
}

func TestPreferredProviderAvailabilityFallbacks(t *testing.T) {
	for name, mutation := range map[string]string{
		"provider disabled": `UPDATE providers SET enabled=0 WHERE id='backup_ds'`,
		"account disabled":  `UPDATE accounts SET enabled=0 WHERE id='acct_backup'`,
		"account deleted":   `DELETE FROM accounts WHERE id='acct_backup'`,
		"provider deleted":  `DELETE FROM accounts WHERE id='acct_backup'; DELETE FROM providers WHERE id='backup_ds'`,
	} {
		t.Run(name, func(t *testing.T) {
			s, db := preferenceFixture(t)
			ctx := context.Background()
			if err := s.SetPreferredProvider(ctx, "deepseek-flash", "backup_ds"); err != nil {
				t.Fatal(err)
			}
			if _, err := db.Exec(mutation); err != nil {
				t.Fatal(err)
			}
			preview, err := s.Resolve(ctx, "deepseek-flash")
			if err != nil || !preview.Routable || preview.PreferenceApplied || preview.ProviderID != "builtin_deepseek" || preview.PreferredProviderID != "backup_ds" {
				t.Fatalf("fallback: %+v %v", preview, err)
			}
			items, err := s.DefaultRoutes(ctx)
			if err != nil || items[0].PreferredProviderID != "backup_ds" || len(items[0].Providers) != 1 {
				t.Fatal("unavailable preference lost or advertised as available", err)
			}
		})
	}
}

func TestExplicitRuleWinsUntilDisabledAndPreferenceCanBeCleared(t *testing.T) {
	s, _ := preferenceFixture(t)
	ctx := context.Background()
	if err := s.SetPreferredProvider(ctx, "deepseek-flash", "backup_ds"); err != nil {
		t.Fatal(err)
	}
	value := input("deepseek-flash")
	value.Model, value.AccountID = "deepseek-v4-pro", "acct_new"
	rule, err := s.Create(ctx, value)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := s.Resolve(ctx, "deepseek-flash")
	if err != nil || !preview.Matched || preview.ProviderID != "builtin_deepseek" || preview.AccountID != "acct_new" || preview.PreferenceApplied {
		t.Fatalf("explicit scope not respected: %+v %v", preview, err)
	}
	if _, err := s.db.Exec(`UPDATE providers SET enabled=0 WHERE id='builtin_deepseek'`); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "deepseek-flash")
	if err != nil || preview.Routable || preview.ErrorCode != "provider_disabled" {
		t.Fatal("invalid explicit rule silently used the preferred provider", err)
	}
	if _, err := s.db.Exec(`UPDATE providers SET enabled=1 WHERE id='builtin_deepseek'`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetEnabled(ctx, rule.ID, false); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "deepseek-flash")
	if err != nil || preview.Matched || !preview.PreferenceApplied {
		t.Fatal("disabled override did not restore preference", err)
	}
	if err := s.SetPreferredProvider(ctx, "deepseek-flash", ""); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "deepseek-flash")
	if err != nil || preview.ProviderID != "builtin_deepseek" || preview.PreferredProviderID != "" {
		t.Fatal("automatic choice not restored", err)
	}
}
