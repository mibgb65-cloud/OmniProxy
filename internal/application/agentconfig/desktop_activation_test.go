package agentconfig

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeDesktopFixture(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

func readDesktopFixture(t *testing.T, path string) map[string]any {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]any
	if json.Unmarshal(body, &values) != nil {
		t.Fatal("invalid desktop fixture JSON")
	}
	return values
}

func TestDesktopOneClickActivationPreservesSettingsAndRestoresModes(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	change, err := s.desktopFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer change.clear()
	normal, thirdParty, meta := change.files[0].path, change.files[1].path, change.files[2].path
	writeDesktopFixture(t, normal, `{"deploymentMode":"1p","windowWidth":640}`)
	writeDesktopFixture(t, thirdParty, `{"deploymentMode":"1p","mcpServers":{"retained":{"command":"fixture"}}}`)
	writeDesktopFixture(t, meta, `{"appliedId":"official","entries":[{"id":"official","name":"Official"}],"other":"keep"}`)
	session := filepath.Join(filepath.Dir(normal), "sessions.jsonl")
	writeDesktopFixture(t, session, "original-history")
	if active, err := s.DesktopGatewayActive(); err != nil || active {
		t.Fatal("gateway active before configuration", err)
	}
	if _, err := s.Apply(ctx, Input{Client: "claude-desktop", Revision: digest(nil, false)}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{normal, thirdParty} {
		if readDesktopFixture(t, path)["deploymentMode"] != "3p" {
			t.Fatal("third-party startup mode not activated")
		}
	}
	if active, err := s.DesktopGatewayActive(); err != nil || !active {
		t.Fatal("gateway not ready after activation", err)
	}
	profile, err := snapshotFile(s.paths.ClaudeDesktop)
	if err != nil {
		t.Fatal(err)
	}
	defer clear(profile.body)
	if _, err := s.Apply(ctx, Input{Client: "claude-desktop", Revision: profile.revision}); err != nil {
		t.Fatal("repeat apply", err)
	}
	writeDesktopFixture(t, normal, `{"deploymentMode":"3p","windowWidth":800,"newSetting":true}`)
	if _, err := s.Restore(ctx, "claude-desktop"); err != nil {
		t.Fatal(err)
	}
	settings := readDesktopFixture(t, normal)
	if settings["deploymentMode"] != "1p" || settings["windowWidth"] != float64(800) || settings["newSetting"] != true {
		t.Fatal("startup baseline or later unrelated settings lost")
	}
	if settings := readDesktopFixture(t, thirdParty); settings["deploymentMode"] != "1p" || settings["mcpServers"] == nil {
		t.Fatal("third-party settings lost")
	}
	if settings := readDesktopFixture(t, meta); settings["appliedId"] != "official" || settings["other"] != "keep" {
		t.Fatal("original profile selection not restored")
	}
	if history, err := os.ReadFile(session); err != nil || string(history) != "original-history" {
		t.Fatal("session history changed")
	}
}

func TestDesktopRestoreRejectsExternallySelectedProfile(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	if _, err := s.Apply(ctx, Input{Client: "claude-desktop", Revision: digest(nil, false)}); err != nil {
		t.Fatal(err)
	}
	before, err := snapshotFile(s.paths.ClaudeDesktop)
	if err != nil {
		t.Fatal(err)
	}
	defer clear(before.body)
	meta := filepath.Join(filepath.Dir(s.paths.ClaudeDesktop), "_meta.json")
	writeDesktopFixture(t, meta, `{"appliedId":"external","entries":[{"id":"external","name":"External"}]}`)
	if _, err := s.Restore(ctx, "claude-desktop"); err == nil || err.Error() != "agent: config_changed" {
		t.Fatal("external selection overwritten", err)
	}
	after, err := snapshotFile(s.paths.ClaudeDesktop)
	if err != nil {
		t.Fatal(err)
	}
	defer clear(after.body)
	if before.revision != after.revision || readDesktopFixture(t, meta)["appliedId"] != "external" {
		t.Fatal("rejected restore modified configuration")
	}
}

func TestDesktopActivationFileConflictRollsBackEarlierFiles(t *testing.T) {
	s := fixture(t)
	change, err := s.prepareDesktopChange(context.Background(), true, false)
	if err != nil {
		t.Fatal(err)
	}
	defer change.clear()
	writeDesktopFixture(t, change.files[2].path, `{"appliedId":"external","entries":[]}`)
	if err := change.write(); err == nil {
		t.Fatal("configuration conflict accepted")
	}
	for _, file := range change.files[:2] {
		if _, err := os.Stat(file.path); !errors.Is(err, os.ErrNotExist) {
			t.Fatal("earlier startup change was not rolled back", err)
		}
	}
	if readDesktopFixture(t, change.files[2].path)["appliedId"] != "external" {
		t.Fatal("external metadata overwritten")
	}
}

func TestDesktopActivationCommitFailureRestoresAllFiles(t *testing.T) {
	s := fixture(t)
	ctx := context.Background()
	change, err := s.desktopFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer change.clear()
	for _, file := range change.files[:2] {
		writeDesktopFixture(t, file.path, `{"deploymentMode":"1p","retained":true}`)
	}
	writeDesktopFixture(t, change.files[2].path, `{"appliedId":"official","entries":[{"id":"official"}]}`)
	if _, err := s.db.Exec(`CREATE TRIGGER reject_desktop_activation BEFORE INSERT ON agent_desktop_activation BEGIN SELECT RAISE(FAIL,'synthetic failure'); END;`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Apply(ctx, Input{Client: "claude-desktop", Revision: digest(nil, false)}); err == nil {
		t.Fatal("failed activation transaction accepted")
	}
	for _, file := range change.files[:2] {
		if readDesktopFixture(t, file.path)["deploymentMode"] != "1p" {
			t.Fatal("startup mode not rolled back")
		}
	}
	if readDesktopFixture(t, change.files[2].path)["appliedId"] != "official" {
		t.Fatal("profile selection not rolled back")
	}
	if _, err := os.Stat(s.paths.ClaudeDesktop); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("failed profile was not rolled back", err)
	}
}
