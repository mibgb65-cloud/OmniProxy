package activity

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/database"
)

// 迁移 045 必须把每日桶里已经累计的 Token 原样搬到身份累计表：
// 升级后账号卡片首次打开就显示历史累计，不能从零开始或有任何缩减。
func TestAccountUsageMigrationBackfillsLifetimeTotals(t *testing.T) {
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "migration.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	files := fstest.MapFS{}
	entries, err := os.ReadDir("../../../migrations")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() >= "045_account_usage_totals.sql" {
			continue
		}
		raw, err := os.ReadFile("../../../migrations/" + entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		files[entry.Name()] = &fstest.MapFile{Data: raw}
	}
	ctx := context.Background()
	if err := database.Migrate(ctx, db, files); err != nil {
		t.Fatal(err)
	}
	item := account.Item{ID: "river-a", ProviderID: "builtin_river", AuthMethod: "api_key"}
	key, err := subject(usageIdentity(item, "river"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO usage_buckets(day,subject,model,provider_name,account_name,email,workspace_id,requests,completed_usage,input_tokens,output_tokens) VALUES
		('2026-10-19',?,'m1','River','Member','','',10,10,1000,100),
		('2026-10-20',?,'m2','River','Member','','',5,4,500,50),
		('2026-10-20','unassigned','m1','','','','',3,0,0,0)`, key, key); err != nil {
		t.Fatal(err)
	}
	// 缓存明细只存在于仍在保留期内的请求明细：回填后仍是已确认部分，未知部分由界面显示下界。
	if _, err := db.Exec(`INSERT INTO request_records(id,started_at,day,protocol,transport,state,subject,input_tokens,output_tokens,cached_input_tokens) VALUES
		('aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa','2026-10-20T01:00:00.000000000Z','2026-10-20','responses','http','success',?,100,10,80),
		('bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb','2026-10-20T02:00:00.000000000Z','2026-10-20','responses','http','success',?,100,10,20)`, key, key); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	var requests, completed, input, output, inputKnown, outputKnown, cached, cachedKnown int64
	if err := db.QueryRow(`SELECT requests,completed_usage,input_tokens,output_tokens,input_known,output_known,cached_input_tokens,cached_known FROM usage_accounts WHERE subject=?`, key).
		Scan(&requests, &completed, &input, &output, &inputKnown, &outputKnown, &cached, &cachedKnown); err != nil {
		t.Fatal(err)
	}
	if requests != 15 || completed != 14 || input != 1500 || output != 150 || cached != 100 || cachedKnown != 2 {
		t.Fatalf("bucket totals were not carried over: %d/%d/%d/%d/%d/%d", requests, completed, input, output, cached, cachedKnown)
	}
	// 桶里没有「明细是否已知」的标记，用完整用量请求数作已知数，不把已有数值标成未知。
	if inputKnown != 14 || outputKnown != 14 {
		t.Fatalf("known counters were fabricated as %d/%d", inputKnown, outputKnown)
	}
	var unassigned int
	if err := db.QueryRow(`SELECT COUNT(*) FROM usage_accounts WHERE subject='unassigned'`).Scan(&unassigned); err != nil || unassigned != 0 {
		t.Fatal("unassigned traffic entered an account row", err, unassigned)
	}
	usage, err := New(db).tokenUsage(ctx, item, "river")
	if err != nil || usage.InputTokens != "1500" || usage.Requests != 15 || usage.UnknownCacheRequests != 13 {
		t.Fatal("backfilled lifetime usage unreadable", usage, err)
	}
}
