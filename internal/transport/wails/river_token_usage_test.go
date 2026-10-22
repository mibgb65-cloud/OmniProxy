package wails

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"omniproxy/internal/application/activity"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/platform/profile"
)

func TestRiverAccountTokenUsageReadsOnlyLocalMetadata(t *testing.T) {
	prof, err := profile.New(profile.Test)
	if err != nil {
		t.Fatal(err)
	}
	prof.DataDir = t.TempDir()
	db, err := database.Open(database.Config{Path: prof.DatabasePath()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	id := "acct_" + strings.Repeat("a", 32)
	if _, err := db.Exec(`INSERT INTO accounts(id,provider_id,name,credential_ref,enabled,created_at,updated_at,auth_method)
		VALUES(?, 'builtin_river','测试账号','must-not-be-read',0,'now','now','api_key')`, id); err != nil {
		t.Fatal(err)
	}
	s := NewAppService(prof, nil, db, nil, nil)
	s.activity = activity.New(db)
	got, err := s.RiverAccountTokenUsage(id)
	if err != nil || got == nil || got.InputTokens != "0" || got.CachedInputTokens != "0" {
		t.Fatal("disabled key or absent credential store blocked local statistics", got, err)
	}
	for _, invalid := range []string{"", strings.Repeat("a", 129), "acct_\n", "acct_' OR 1=1 --"} {
		if _, err := s.RiverAccountTokenUsage(invalid); err == nil {
			t.Fatal("invalid binding input accepted")
		}
	}
	if _, err := db.Exec(`UPDATE accounts SET provider_id='builtin_deepseek' WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.RiverAccountTokenUsage(id); !errors.Is(err, activity.ErrInvalid) {
		t.Fatal("unsupported provider accepted", err)
	}
	s.activity = nil
	if _, err := s.RiverAccountTokenUsage(id); !errors.Is(err, activity.ErrUnavailable) {
		t.Fatal("unavailable service hidden", err)
	}
}
