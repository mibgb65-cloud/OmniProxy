package routing

import (
	"context"
	"testing"

	"omniproxy/internal/provider/oxionai"
)

// TestOxionDiscoveredModelsBecomeRoutable 断言 Oxion AI 没有内置预设：
// 只有同步到的官方目录模型才能解析，且不能路由到别的服务商。
func TestOxionDiscoveredModelsBecomeRoutable(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at)
		VALUES('oxionai-account','builtin_oxionai','Oxion account','protected-ref',1,'2026-10-24T00:00:00Z','2026-10-24T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if preview, err := s.Resolve(ctx, "gemma-2-2b"); err != nil || preview.Routable {
		t.Fatal("model must stay unroutable before discovery", preview, err)
	}
	if _, err := db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version,metadata_json)
		SELECT 'oxionai-account','gemma-2-2b','Gemma 2 2B','now',v.version,'{"contextWindow":128000,"inputModalities":["text","image"],"toolUse":true}'
		FROM account_catalog_versions v WHERE v.id='oxionai-account'`); err != nil {
		t.Fatal(err)
	}
	preview, err := s.Resolve(ctx, "gemma-2-2b")
	if err != nil || !preview.Routable || preview.ProviderID != "builtin_oxionai" || preview.AccountID != "oxionai-account" {
		t.Fatal("discovered model not routable", preview, err)
	}
	if _, err := s.Create(ctx, Input{Alias: "oxion-chat", ProviderID: "builtin_oxionai", Model: "gemma-2-2b", AccountID: "oxionai-account", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "oxion-chat")
	if err != nil || !preview.Routable || !preview.Matched || preview.Model != "gemma-2-2b" {
		t.Fatal("alias failed", preview, err)
	}
	entries, err := s.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, entry := range entries {
		if entry.ProviderKind == oxionai.Kind {
			found++
			if entry.Metadata.ContextWindow == nil || *entry.Metadata.ContextWindow != 128000 {
				t.Fatal("discovered metadata lost", entry)
			}
		}
	}
	if found != 2 {
		t.Fatalf("model and alias count = %d, want 2", found)
	}
	for _, change := range []string{`UPDATE accounts SET enabled=0 WHERE id='oxionai-account'`, `UPDATE accounts SET enabled=1,auth_method='subscription_login' WHERE id='oxionai-account'`, `UPDATE providers SET enabled=0 WHERE id='builtin_oxionai'`} {
		if _, err := db.Exec(change); err != nil {
			t.Fatal(err)
		}
		if preview, err := s.Resolve(ctx, "gemma-2-2b"); err != nil || preview.Routable {
			t.Fatal("ineligible account routed", preview, err)
		}
	}
	if _, err := s.Create(ctx, Input{Alias: "wrong", ProviderID: "builtin_deepseek", Model: "gemma-2-2b", Enabled: true}); err == nil {
		t.Fatal("model routed to wrong provider")
	}
}
