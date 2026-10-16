package modelcatalog

import (
	"context"
	"omniproxy/internal/application/account"
	"omniproxy/internal/application/provider"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/provider/kimi"
	"os"
	"path/filepath"
	"testing"
)

func TestKimiCodeUsesHonestPresetsAndDoesNotQueryUnconfirmedDiscovery(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(database.Config{Path: filepath.Join(t.TempDir(), "code.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(ctx, db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	accounts := account.NewService(db, googleStore{})
	a, err := accounts.Create(ctx, account.CreateInput{ProviderID: "builtin_kimi", Name: "Code", APIKey: "placeholder-key", Kimi: &kimi.Options{Region: "global", BillingMode: "coding"}})
	if err != nil {
		t.Fatal(err)
	}
	s := New(db, accounts, provider.NewService(db), nil)
	items, err := s.List(ctx)
	if err != nil || len(items) != 4 {
		t.Fatal("subscription presets missing", err)
	}
	for _, m := range items {
		if m.Source != "preset" || m.FetchedAt != "" {
			t.Fatal("subscription preset pretends to be discovery")
		}
		if m.Model == "kimi-k3" && (m.Metadata.ContextWindow == nil || *m.Metadata.ContextWindow != 262144) {
			t.Fatal("unknown membership tier treated as 1M")
		}
	}
	if _, err := s.Sync(ctx, a.ID); err == nil {
		t.Fatal("unconfirmed discovery accepted")
	}
}
