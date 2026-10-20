package agentconfig

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type loginTrialStore struct {
	*memoryStore
	onPut func()
	fail  bool
}

func (s *loginTrialStore) Put(body []byte) (string, error) {
	if s.fail {
		return "", errors.New("synthetic-store-failure")
	}
	ref, err := s.memoryStore.Put(body)
	if s.onPut != nil {
		s.onPut()
	}
	return ref, err
}

func TestCodexLoginTrialBackupFailureDoesNotWriteCredentials(t *testing.T) {
	service, config, original := loginTrialFixture(t, true)
	service.store = &loginTrialStore{memoryStore: service.store.(*memoryStore), fail: true}
	if err := service.TryCodexFileLogin([]byte(trialAuth), "synthetic-account"); err == nil {
		t.Fatal("backup failure accepted")
	}
	authPath, manifestPath := service.loginTrialPaths()
	for path, want := range map[string][]byte{service.paths.Codex: config, authPath: original} {
		body, err := os.ReadFile(path)
		if err != nil || string(body) != string(want) {
			t.Fatal("backup failure changed a client file")
		}
	}
	if _, err := os.Stat(manifestPath); !os.IsNotExist(err) {
		t.Fatal("backup failure created a trial receipt")
	}
}

func TestCodexLoginTrialRollsBackAuthWhenConfigurationChanges(t *testing.T) {
	service, _, original := loginTrialFixture(t, true)
	service.store = &loginTrialStore{memoryStore: service.store.(*memoryStore), onPut: func() {
		if err := os.WriteFile(service.paths.Codex, []byte("external-change"), 0600); err != nil {
			t.Fatal(err)
		}
	}}
	if err := service.TryCodexFileLogin([]byte(trialAuth), "synthetic-account"); err == nil {
		t.Fatal("concurrent config change accepted")
	}
	authPath, manifestPath := service.loginTrialPaths()
	auth, err := os.ReadFile(authPath)
	if err != nil || string(auth) != string(original) {
		t.Fatal("failed configuration update did not roll back our auth write")
	}
	body, err := os.ReadFile(service.paths.Codex)
	if err != nil || string(body) != "external-change" {
		t.Fatal("external config was overwritten during rollback")
	}
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatal("recovery receipt lost after external conflict")
	}
}

func TestCodexLoginTrialRestoreRejectsBackupFromAnotherHome(t *testing.T) {
	service, _, _ := loginTrialFixture(t, true)
	other, config, original := loginTrialFixture(t, true)
	other.store = service.store
	if err := service.TryCodexFileLogin([]byte(trialAuth), "synthetic-account"); err != nil {
		t.Fatal(err)
	}
	_, manifestPath := service.loginTrialPaths()
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(other.paths.Codex), loginTrialName), body, 0600); err != nil {
		t.Fatal(err)
	}
	if err := other.RestoreCodexFileLogin(); err == nil {
		t.Fatal("backup for another target was accepted")
	}
	for path, want := range map[string][]byte{other.paths.Codex: config, filepath.Join(filepath.Dir(other.paths.Codex), "auth.json"): original} {
		actual, err := os.ReadFile(path)
		if err != nil || string(actual) != string(want) {
			t.Fatal("cross-home restore changed a file")
		}
	}
	var receipt loginTrialManifest
	if json.Unmarshal(body, &receipt) != nil || len(service.store.(*memoryStore).values) != 1 {
		t.Fatal("original backup lost")
	}
}
