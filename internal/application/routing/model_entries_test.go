package routing

import (
	"context"
	"testing"
)

func TestDirectoryAliasInheritsTheSelectedAccountMetadata(t *testing.T) {
	s, db := fixture(t)
	_, err := db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version,metadata_json) VALUES
 ('acct_old','deepseek-flash','Old Model','now','unused:','{"description":"old","contextWindow":100000}'),
 ('acct_new','deepseek-flash','New Model','now','unused:','{"description":"new","contextWindow":200000}')`)
	if err != nil {
		t.Fatal(err)
	}
	rule := input("coding-route")
	rule.AccountID = "acct_new"
	if _, err := s.Create(context.Background(), rule); err != nil {
		t.Fatal(err)
	}
	entries, err := s.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, entry := range entries {
		if entry.ID != "coding-route" {
			continue
		}
		found = true
		if !entry.Alias || entry.TargetModel != "deepseek-flash" || entry.Metadata.Description != "new" || entry.Metadata.ContextWindow == nil || *entry.Metadata.ContextWindow != 200000 || len(entry.Protocols) != 3 {
			t.Fatal("alias metadata came from a different target")
		}
	}
	if !found {
		t.Fatal("alias missing")
	}
	if _, err := db.Exec(`UPDATE accounts SET enabled=0 WHERE id='acct_new'`); err != nil {
		t.Fatal(err)
	}
	entries, err = s.Catalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.ID == "coding-route" {
			t.Fatal("unavailable fixed account remained in directory")
		}
	}
}
