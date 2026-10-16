//go:build windows

package wails

import (
	"context"
	"encoding/json"
	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/platform/profile"
	"omniproxy/internal/provider/kimi"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKimiCodeBindingsProtectKeyAndOpenOnlyFixedConsole(t *testing.T) {
	prof := &profile.Profile{Kind: profile.Test, DataDir: t.TempDir()}
	db, err := database.Open(database.Config{Path: filepath.Join(prof.DataDir, "code.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := database.Migrate(context.Background(), db, os.DirFS("../../../migrations")); err != nil {
		t.Fatal(err)
	}
	s := &AppService{prof: prof, db: db}
	opened := ""
	s.openAgentURL = func(address string) error { opened = address; return nil }
	a, err := s.CreateAccount(account.CreateInput{ProviderID: "builtin_kimi", Name: "Code", APIKey: "placeholder-protected-code-key", Kimi: &kimi.Options{Region: "cn", BillingMode: "coding"}})
	if err != nil {
		t.Fatal(err)
	}
	items, err := s.ListAccounts()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(items)
	if err != nil || strings.Contains(string(raw), "placeholder-protected-code-key") || strings.Contains(string(raw), "credential_ref") {
		t.Fatal("credential leaked to metadata")
	}
	if err := s.OpenKimiCodeConsole(a.ID); err != nil || opened != "https://www.kimi.com/code/console" {
		t.Fatal("console not fixed", err)
	}
	ordinary, err := s.CreateAccount(account.CreateInput{ProviderID: "builtin_kimi", Name: "API", APIKey: "placeholder-api-key"})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{ordinary.ID, "https://untrusted.invalid"} {
		opened = ""
		if s.OpenKimiCodeConsole(id) == nil || opened != "" {
			t.Fatal("arbitrary console target accepted")
		}
	}
	s.openAgentURL = nil
	if s.OpenKimiCodeConsole(a.ID) == nil {
		t.Fatal("missing opener accepted")
	}
}
