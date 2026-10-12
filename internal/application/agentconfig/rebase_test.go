package agentconfig

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestReinstalledClaudeUsesFreshSnapshotPreservesSettingsAndArchivesBackup(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	if err := os.MkdirAll(filepath.Dir(s.paths.Claude), 0700); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"theme":"dark","env":{"CUSTOM":"old"}}`)
	if err := os.WriteFile(s.paths.Claude, original, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, Input{Client: "claude", Revision: digest(original, true)}); err != nil {
		t.Fatal(err)
	}
	var oldRef string
	if err := s.db.QueryRow(`SELECT credential_ref FROM agent_backups WHERE client='claude'`).Scan(&oldRef); err != nil {
		t.Fatal(err)
	}
	fresh := []byte(`{"theme":"light","env":{"CUSTOM":"new"},"permissions":{"allow":["Read"]}}`)
	if err := os.WriteFile(s.paths.Claude, fresh, 0600); err != nil {
		t.Fatal(err)
	}
	input := Input{Client: "claude", Revision: digest(fresh, true)}
	if _, err := s.Apply(ctx, input); err == nil {
		t.Fatal("external change accepted without reread confirmation")
	}
	input.AcceptCurrentConfig = true
	if _, err := s.PreviewPlan(ctx, input); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, input); err != nil {
		t.Fatal(err)
	}
	patched, err := os.ReadFile(s.paths.Claude)
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if json.Unmarshal(patched, &config) != nil || config["theme"] != "light" || config["env"].(map[string]any)["CUSTOM"] != "new" || config["permissions"] == nil {
		t.Fatal("fresh settings lost")
	}
	var archived string
	if err := s.db.QueryRow(`SELECT credential_ref FROM agent_backup_history WHERE client='claude'`).Scan(&archived); err != nil || archived != oldRef {
		t.Fatal("previous backup not archived")
	}
	if _, err := s.store.Get(oldRef); err != nil {
		t.Fatal("previous encrypted backup deleted")
	}
	if _, err := s.Restore(ctx, "claude"); err != nil {
		t.Fatal(err)
	}
	restored, err := os.ReadFile(s.paths.Claude)
	if err != nil || string(restored) != string(fresh) {
		t.Fatal("restore did not use reread baseline")
	}
}

func TestRebaseCommitFailurePreservesCurrentFileAndOriginalBackup(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	if err := os.MkdirAll(filepath.Dir(s.paths.Claude), 0700); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"theme":"dark"}`)
	if err := os.WriteFile(s.paths.Claude, original, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, Input{Client: "claude", Revision: digest(original, true)}); err != nil {
		t.Fatal(err)
	}
	var oldRef string
	if err := s.db.QueryRow(`SELECT credential_ref FROM agent_backups WHERE client='claude'`).Scan(&oldRef); err != nil {
		t.Fatal(err)
	}
	fresh := []byte(`{"theme":"light"}`)
	if err := os.WriteFile(s.paths.Claude, fresh, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`CREATE TRIGGER reject_rebase BEFORE INSERT ON agent_backup_history BEGIN SELECT RAISE(FAIL,'private error'); END;`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, Input{Client: "claude", Revision: digest(fresh, true), AcceptCurrentConfig: true}); err == nil {
		t.Fatal("failed commit accepted")
	}
	current, err := os.ReadFile(s.paths.Claude)
	if err != nil || string(current) != string(fresh) {
		t.Fatal("external settings lost on failure")
	}
	var ref string
	if err := s.db.QueryRow(`SELECT credential_ref FROM agent_backups WHERE client='claude'`).Scan(&ref); err != nil || ref != oldRef {
		t.Fatal("original backup changed on failure")
	}
}
