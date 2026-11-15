package routing

import (
	"context"
	"testing"

	"omniproxy/internal/provider/howtofix"
)

// TestHowToFixDiscoveredModelsBecomeRoutable 断言 HowToFix 没有内置预设：
// 只有同步到的账号目录模型才能解析，且不能路由到别的服务商。
func TestHowToFixDiscoveredModelsBecomeRoutable(t *testing.T) {
	s, db := fixture(t)
	ctx := context.Background()
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at)
		VALUES('howtofix-account','builtin_howtofix','HowToFix account','protected-ref',1,'2026-11-15T00:00:00Z','2026-11-15T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if preview, err := s.Resolve(ctx, "glm-5.3-flash"); err != nil || preview.Routable {
		t.Fatal("model must stay unroutable before discovery", preview, err)
	}
	if _, err := db.Exec(`INSERT INTO account_models(account_id,model,name,fetched_at,account_version,metadata_json)
		SELECT 'howtofix-account','glm-5.3-flash','GLM-5.3-Flash','now',v.version,'{"contextWindow":1000000,"inputModalities":["text","image"],"toolUse":true}'
		FROM account_catalog_versions v WHERE v.id='howtofix-account'`); err != nil {
		t.Fatal(err)
	}
	preview, err := s.Resolve(ctx, "glm-5.3-flash")
	if err != nil || !preview.Routable || preview.ProviderID != "builtin_howtofix" || preview.AccountID != "howtofix-account" {
		t.Fatal("discovered model not routable", preview, err)
	}
	if _, err := s.Create(ctx, Input{Alias: "h2f-chat", ProviderID: "builtin_howtofix", Model: "glm-5.3-flash", AccountID: "howtofix-account", Enabled: true}); err != nil {
		t.Fatal(err)
	}
	preview, err = s.Resolve(ctx, "h2f-chat")
	if err != nil || !preview.Routable || !preview.Matched || preview.Model != "glm-5.3-flash" {
		t.Fatal("alias failed", preview, err)
	}
	entries, err := s.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, entry := range entries {
		if entry.ProviderKind == howtofix.Kind {
			found++
			if entry.Metadata.ContextWindow == nil || *entry.Metadata.ContextWindow != 1000000 {
				t.Fatal("discovered metadata lost", entry)
			}
		}
	}
	if found != 2 {
		t.Fatalf("model and alias count = %d, want 2", found)
	}
	// 受管客户端的模型目录来自已配置的路由，因此上面两条条目会同时出现在 agent 目录里。
	for _, change := range []string{`UPDATE accounts SET enabled=0 WHERE id='howtofix-account'`, `UPDATE accounts SET enabled=1,auth_method='subscription_login' WHERE id='howtofix-account'`, `UPDATE providers SET enabled=0 WHERE id='builtin_howtofix'`} {
		if _, err := db.Exec(change); err != nil {
			t.Fatal(err)
		}
		if preview, err := s.Resolve(ctx, "glm-5.3-flash"); err != nil || preview.Routable {
			t.Fatal("ineligible account routed", preview, err)
		}
	}
	if _, err := s.Create(ctx, Input{Alias: "wrong", ProviderID: "builtin_deepseek", Model: "glm-5.3-flash", Enabled: true}); err == nil {
		t.Fatal("model routed to wrong provider")
	}
}
