//go:build windows

package wails

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"omniproxy/internal/application/account"
	"omniproxy/internal/infrastructure/database"
	"omniproxy/internal/platform/profile"
	"omniproxy/internal/provider/mimo"
)

func TestMiMoBindingsProtectPlanKeyAndOpenOnlyOfficialConsole(t *testing.T) {
	prof := &profile.Profile{Kind: profile.Test, DataDir: t.TempDir()}
	db, err := database.Open(database.Config{Path: filepath.Join(prof.DataDir, "mimo.db")})
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
	plan, err := s.CreateAccount(account.CreateInput{ProviderID: "builtin_mimo", Name: "Plan", APIKey: "ttp-placeholder-protected-key", MiMo: &mimo.Options{Region: "ams", BillingMode: "token_plan"}})
	if err != nil || plan.MiMo == nil || plan.MiMo.Region != "ams" {
		t.Fatal("binding lost plan options", err)
	}
	items, err := s.ListAccounts()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(items)
	if err != nil || strings.Contains(string(raw), "ttp-placeholder-protected-key") || strings.Contains(string(raw), "credential_ref") {
		t.Fatal("credential reached metadata binding")
	}
	if err := s.OpenMiMoConsole(plan.ID); err != nil || opened != "https://platform.xiaomimimo.com/token-plan" {
		t.Fatal("plan did not open fixed console", err)
	}
	opened = ""
	ordinary, err := s.CreateAccount(account.CreateInput{ProviderID: "builtin_mimo", Name: "Ordinary", APIKey: "sk-placeholder-key"})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{ordinary.ID, "https://untrusted.invalid"} {
		if s.OpenMiMoConsole(id) == nil || opened != "" {
			t.Fatal("ordinary account or arbitrary URL accepted")
		}
	}
	s.openAgentURL = nil
	if s.OpenMiMoConsole(plan.ID) == nil {
		t.Fatal("missing browser accepted")
	}
}
