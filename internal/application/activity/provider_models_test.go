package activity

import (
	"context"
	"omniproxy/internal/core/proxy"
	"omniproxy/internal/infrastructure/database"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestProviderModelIsolationAndDurability(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	first := event(1, "a@example.invalid", "a")
	first.Model = "same-model"
	first.Usage = proxy.UsageSnapshot{Input: 100, Output: 10, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, first)
	second := event(2, "", "b")
	second.Model = first.Model
	second.Target.ProviderID = "builtin_deepseek"
	second.Target.ProviderKind = "deepseek"
	second.Target.ProviderName = "DeepSeek"
	second.Target.AuthMethod = "api_key"
	second.Usage = proxy.UsageSnapshot{Input: 200, Output: 20, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, second)
	member := event(3, "b@example.invalid", "c")
	member.Model = first.Model
	member.Usage = proxy.UsageSnapshot{Input: 20, Output: 5, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, member)
	config := event(4, "a@example.invalid", "a")
	config.Model = first.Model
	config.Target.ProviderID = "other-openai-config"
	config.Target.ProviderName = "OpenAI Work"
	config.Usage = proxy.UsageSnapshot{Input: 3, Output: 1, InputKnown: true, OutputKnown: true, Complete: true}
	record(t, s, config)
	info, err := s.Usage(ctx)
	if err != nil || len(info.ProviderModels) != 3 {
		t.Fatal("provider models merged", err)
	}
	expected := map[string]string{"builtin_openai": "135", "builtin_deepseek": "220", "other-openai-config": "4"}
	for _, item := range info.ProviderModels {
		if item.Model != "same-model" || item.Summary.TotalTokens != expected[item.ProviderID] {
			t.Fatal("incorrect provider total", item)
		}
	}
	if err = s.Update(ctx, first); err != nil {
		t.Fatal(err)
	}
	s.db.Exec(`DELETE FROM request_records`)
	s.db.Exec(`DELETE FROM providers WHERE id='builtin_openai'`)
	persisted, err := s.Usage(ctx)
	if err != nil || len(persisted.ProviderModels) != 3 || persisted.AllTime.TotalTokens != info.AllTime.TotalTokens {
		t.Fatal("deletion lost provider model history", err)
	}
}

func TestProviderModelMigrationPreservesHistoricalBuckets(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "old.db")})
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
		if strings.HasPrefix(entry.Name(), "009_") {
			continue
		}
		data, err := os.ReadFile(filepath.Join("../../../migrations", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		files[entry.Name()] = &fstest.MapFile{Data: data}
	}
	if err = database.Migrate(ctx, db, files); err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`INSERT INTO usage_buckets VALUES('2026-10-09','known','same-model','OpenAI','Member','','',1,1,10,2),('2026-10-09','gone','old-model','Removed vendor','Member','','',20,20,100,1),('2026-10-09','mixed','same-model','Shared vendor','Member','','',2,2,30,6);
 INSERT INTO request_records(id,started_at,day,protocol,transport,subject,provider_id,model) VALUES('known','now','2026-10-09','chat','http','known','builtin_openai','same-model'),('mix1','now','2026-10-09','chat','http','mixed','provider-one','same-model'),('mix2','now','2026-10-09','chat','http','mixed','provider-two','same-model');DELETE FROM providers WHERE id='builtin_openai';`)
	if err != nil {
		t.Fatal(err)
	}
	if err = database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	rows, err := New(db).providerModels(ctx)
	if err != nil || len(rows) != 3 {
		t.Fatal("migration failed", err)
	}
	expected := map[string]string{"builtin_openai": "12", "legacy:Removed vendor": "101", "legacy:Shared vendor": "36"}
	for _, item := range rows {
		if expected[item.ProviderID] != item.Summary.TotalTokens {
			t.Fatal("historical attribution changed", item)
		}
	}
}
